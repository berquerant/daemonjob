/*
Copyright 2026 berquerant.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"slices"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	batchv1 "k8s.io/api/batch/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
)

// DaemonJobReconciler reconciles a DaemonJob object
type DaemonJobReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	BroadcastImage string
	BroadcastRole  string
}

const daemonJobReconcilerFinalizer = "daemonjobs.daemonjob.berquerant.github.io/finalizer"

// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemonjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemonjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemonjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;watch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DaemonJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *DaemonJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	logger.Info("Start to reconcile")
	daemonJob, err := Get[*daemonjobv1.DaemonJob](ctx, r.Client, req.NamespacedName)
	if apierrors.IsNotFound(err) {
		logger.Info("DaemonJob is deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "Unable to get DaemonJob")
		return ctrl.Result{}, err
	}
	if !daemonJob.DeletionTimestamp.IsZero() {
		logger.Info("Deleting the DaemonJob")
		// Finalizers are deleting the DaemonJob.
		// https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
		if ContainsFinalizer(daemonJob, daemonJobReconcilerFinalizer) {
			if err := r.onDelete(ctx, daemonJob); err != nil {
				logger.Error(err, "Unable to process on deletion")
				return ctrl.Result{}, err
			}
		}
		if err := RemoveFinalizerIfExist(ctx, r.Client, daemonJob, daemonJobReconcilerFinalizer); err != nil {
			logger.Error(err, "Unable to remove finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := AddFinalizerIfNotExist(ctx, r.Client, daemonJob, daemonJobReconcilerFinalizer); err != nil {
		logger.Error(err, "Unable to ensure finalizer")
		return ctrl.Result{}, err
	}

	if err := r.deleteWorkerJobsOnNotExistNode(ctx, daemonJob); err != nil {
		logger.Error(err, "Unable to delete the worker Jobs on deleted nodes")
	}

	if err := r.applyJob(ctx, daemonJob); err != nil {
		logger.Error(err, "Unable to apply Job")
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, daemonJob); err != nil {
		logger.Error(err, "Unable to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DaemonJobReconciler) onDelete(ctx context.Context, daemonJob *daemonjobv1.DaemonJob) error {
	logger := logf.FromContext(ctx)
	// Delete ClusterRoleBinding using a finalizer.
	// Since it's a cluster-scoped resource, it cannot set a controller as its owner reference.
	crbs, err := ListDaemonJobClusterRoleBindings(ctx, r.Client, daemonJob.Namespace, daemonJob.Name)
	if err != nil {
		return err
	}
	for _, crb := range crbs.Items {
		if err := r.Delete(ctx, &crb); err != nil {
			logger.Error(err, "Unable to delete ClusterRoleBinding for the broadcast Job", "clusterrolebinding", crb.Name)
			return err
		}
	}
	return nil
}

func (r *DaemonJobReconciler) deleteWorkerJobsOnNotExistNode(ctx context.Context, daemonJob *daemonjobv1.DaemonJob) error {
	nodes, err := List[*corev1.NodeList](ctx, r.Client)
	if err != nil {
		return err
	}
	nodeNames := make([]string, len(nodes.Items))
	for i, x := range nodes.Items {
		nodeNames[i] = x.Name
	}

	jobs, err := ListDaemonJobWorkerJobs(ctx, r.Client, daemonJob.Namespace, daemonJob.Name)
	if err != nil {
		return err
	}
	for _, x := range jobs.Items {
		if node, ok := x.Labels[DaemonJobLabelNode]; ok {
			if !slices.Contains(nodeNames, node) {
				if err := r.Delete(ctx, &x); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *DaemonJobReconciler) broadcastArgs(daemonJob *daemonjobv1.DaemonJob) *DaemonJobBroadcastArgs {
	return NewDaemonJobBroadcastArgs(daemonJob, r.BroadcastRole, r.BroadcastImage)
}

func (r *DaemonJobReconciler) applyJob(ctx context.Context, daemonJob *daemonjobv1.DaemonJob) error {
	var (
		logger         = logf.FromContext(ctx)
		createOrUpdate = func(obj client.Object, kind string) error {
			l := logger.WithValues("kind", kind, "name", obj.GetName())

			if err := CreateOrUpdate(ctx, r.Client, r.Scheme, daemonJob, obj); err != nil {
				l.Error(err, "Unable to create or update")
				return err
			}
			l.Info("Created or updated")
			return nil
		}
		args = r.broadcastArgs(daemonJob)
	)

	if err := createOrUpdate(args.ServiceAccount(), "ServiceAccount"); err != nil {
		return err
	}
	{
		crb := args.ClusterRoleBinding()
		l := logger.WithValues("kind", "ClusterRoleBinding", "name", crb.Name)
		// Create ClusterRoleBinding if not exist.
		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, crb, nil); err != nil {
			l.Error(err, "Unable to create or update")
			return err
		}
		l.Info("Created or updated")
	}
	if err := createOrUpdate(args.Job(), "Job"); err != nil {
		return err
	}
	return nil
}

func (r *DaemonJobReconciler) updateStatus(ctx context.Context, daemonJob *daemonjobv1.DaemonJob) error {
	var (
		status = NewStatus(
			&daemonJob.Status.Conditions,
			daemonjobv1.TypeDaemonJobComplete,
			daemonjobv1.TypeDaemonJobProgressing,
			daemonjobv1.TypeDaemonJobDegraded,
			daemonjobv1.ReasonReconciling,
		)
		setState = func() {
			state, ok := status.State()
			if !ok {
				state = daemonjobv1.TypeDaemonJobUnknown
			}
			daemonJob.Status.State = state
		}
		updateStatus = func() error {
			setState()
			return r.Status().Update(ctx, daemonJob)
		}
		args = r.broadcastArgs(daemonJob)
	)
	//
	// Ensure that the required resources exist.
	//
	if exist, err := Exist[*corev1.ServiceAccount](ctx, r.Client, client.ObjectKeyFromObject(args.ServiceAccount())); !exist {
		if err != nil {
			return err
		}
		status.SetNotFound("ServiceAccount")
	}
	if exist, err := Exist[*rbacv1.ClusterRoleBinding](ctx, r.Client, client.ObjectKeyFromObject(args.ClusterRoleBinding())); !exist {
		if err != nil {
			return err
		}
		status.SetNotFound("ClusterRoleBinding")
	}
	if exist, err := Exist[*batchv1.Job](ctx, r.Client, client.ObjectKeyFromObject(args.Job())); !exist {
		if err != nil {
			return err
		}
		status.SetNotFound("Job")

		return updateStatus()
	}
	//
	// Check created Job status.
	//
	job, err := Get[*batchv1.Job](ctx, r.Client, client.ObjectKeyFromObject(args.Job()))
	if err != nil {
		return err
	}
	jobStatus := CheckJobStatus(job.Status.Conditions)
	switch jobStatus.Type {
	case JobStatusProgressing:
		status.SetProgressing(daemonjobv1.ReasonBroadcasting, "Broadcast is running")
	case JobStatusFailed:
		msg := jobStatus.Reason + ": " + jobStatus.Message
		if pods, err := ListJobPods(ctx, r.Client, job); err == nil {
			ss := []string{}
			for _, p := range pods.Items {
				if x := CollectTerminationLogs(&p.Status).String(); x != "" {
					ss = append(ss, p.Name+": "+x)
				}
			}
			if len(ss) > 0 {
				msg += "\n" + strings.Join(ss, "\n")
			}
		}
		status.SetPair(false, daemonjobv1.ReasonFailed, msg)
	case JobStatusComplete:
		// The broadcast job is completed.
		// Check the worker jobs created by the job.
		workerJobs, err := ListDaemonJobWorkerJobs(ctx, r.Client, daemonJob.Namespace, daemonJob.Name)
		if err != nil {
			return err
		}
		type element struct {
			name   string
			status *JobStatus
		}
		elems := make([]element, len(workerJobs.Items))
		for i, x := range workerJobs.Items {
			elems[i] = element{
				name:   x.Name,
				status: CheckJobStatus(x.Status.Conditions),
			}
		}
		switch {
		case slices.ContainsFunc(elems, func(x element) bool { return x.status.Type == JobStatusFailed }):
			// Some worker jobs are failed.
			xs := slices.DeleteFunc(elems, func(x element) bool { return x.status.Type != JobStatusFailed })
			ss := make([]string, len(xs))
			for i, x := range xs {
				ss[i] = x.name + " failed," + x.status.Reason + ": " + x.status.Message
			}
			msg := "Some jobs failed\n" + strings.Join(ss, "\n")
			status.SetPair(false, daemonjobv1.ReasonFailed, msg)
		case slices.ContainsFunc(elems, func(x element) bool { return x.status.Type == JobStatusProgressing }):
			// Some worker jobs are progressing.
			xs := slices.DeleteFunc(elems, func(x element) bool { return x.status.Type != JobStatusProgressing })
			ss := make([]string, len(xs))
			for i, x := range xs {
				ss[i] = x.name
			}
			msg := "Some jobs are progressing: " + strings.Join(ss, ", ")
			status.SetProgressing(daemonjobv1.ReasonProgressing, msg)
		default:
			status.SetPair(true, daemonjobv1.ReasonOK, "All jobs are completed")
		}
	default:
		status.SetPair(false, daemonjobv1.ReasonUnknown, "Unknown")
	}

	return updateStatus()
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&daemonjobv1.DaemonJob{}).
		Named("daemonjob").
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Watches(
			&batchv1.Job{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
				// Jobs created by DaemonJob should have the 'daemonjob-name' label.
				if daemonJobName, ok := obj.GetLabels()[DaemonJobLabelDaemonJobName]; ok && daemonJobName != "" {
					return []reconcile.Request{
						{
							NamespacedName: types.NamespacedName{
								Namespace: obj.GetNamespace(),
								Name:      daemonJobName,
							},
						},
					}
				}
				// Ignore this object.
				return []reconcile.Request{}
			}),
		).
		Watches(
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				logger := logf.FromContext(ctx).WithValues("node", obj.GetName())
				xs, err := ListDaemonJobWorkerJobsByNode(ctx, r.Client, obj.GetName())
				if err != nil {
					logger.Error(err, "Unable to list worker jobs by node")
					return []reconcile.Request{}
				}

				var requests []reconcile.Request
				for _, x := range xs.Items {
					if daemonJobName, ok := x.Labels[DaemonJobLabelDaemonJobName]; ok {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: x.Namespace,
								Name:      daemonJobName,
							},
						})
					}
				}

				return requests
			}),
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(_ event.UpdateEvent) bool {
					return false
				},
				CreateFunc: func(_ event.CreateEvent) bool {
					return false
				},
				DeleteFunc: func(_ event.DeleteEvent) bool {
					// Watch node deleted events.
					return true
				},
				GenericFunc: func(_ event.GenericEvent) bool {
					return false
				},
			}),
		).
		Complete(r)
}
