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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
)

// DaemonCronJobReconciler reconciles a DaemonCronJob object
type DaemonCronJobReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	BroadcastImage string
	BroadcastRole  string
}

const daemonCronJobReconcilerFinalizer = "daemoncronjobs.daemonjob.berquerant.github.io/finalizer"

// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemoncronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemoncronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemoncronjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs/status,verbs=get;watch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DaemonCronJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *DaemonCronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	logger.Info("Start to reconcile")
	daemonCronJob, err := Get[*daemonjobv1.DaemonCronJob](ctx, r.Client, req.NamespacedName)
	if apierrors.IsNotFound(err) {
		logger.Info("DaemonCronJob is deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "Unable to get DaemonCronJob")
		return ctrl.Result{}, err
	}
	if !daemonCronJob.DeletionTimestamp.IsZero() {
		logger.Info("Deleting the DaemonCronJob")
		// Finalizers are deleting the DaemonCronJob.
		// https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
		if ContainsFinalizer(daemonCronJob, daemonCronJobReconcilerFinalizer) {
			if err := r.onDelete(ctx, daemonCronJob); err != nil {
				logger.Error(err, "Unable to process on deletion")
				return ctrl.Result{}, err
			}
		}
		if err := RemoveFinalizerIfExist(ctx, r.Client, daemonCronJob, daemonCronJobReconcilerFinalizer); err != nil {
			logger.Error(err, "Unable to remove finalizer")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if err := AddFinalizerIfNotExist(ctx, r.Client, daemonCronJob, daemonCronJobReconcilerFinalizer); err != nil {
		logger.Error(err, "Unable to ensure finalizer")
		return ctrl.Result{}, err
	}

	if err := r.deleteWorkerJobsOnNotExistNode(ctx, daemonCronJob); err != nil {
		logger.Error(err, "Unable to delete the worker Jobs on deleted nodes")
	}

	if err := r.applyCronJob(ctx, daemonCronJob); err != nil {
		logger.Error(err, "Unable to apply CronJob")
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, daemonCronJob); err != nil {
		logger.Error(err, "Unable to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DaemonCronJobReconciler) onDelete(ctx context.Context, daemonCronJob *daemonjobv1.DaemonCronJob) error {
	logger := logf.FromContext(ctx)
	// Delete ClusterRoleBinding using a finalizer.
	// Since it's a cluster-scoped resource, it cannot set a controller as its owner reference.
	crbs, err := ListDaemonCronJobClusterRoleBindings(ctx, r.Client, daemonCronJob.Namespace, daemonCronJob.Name)
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

func (r *DaemonCronJobReconciler) deleteWorkerJobsOnNotExistNode(ctx context.Context, daemonCronJob *daemonjobv1.DaemonCronJob) error {
	nodes, err := List[*corev1.NodeList](ctx, r.Client)
	if err != nil {
		return err
	}
	nodeNames := make([]string, len(nodes.Items))
	for i, x := range nodes.Items {
		nodeNames[i] = x.Name
	}

	jobs, err := ListDaemonCronJobWorkerJobs(ctx, r.Client, daemonCronJob.Namespace, daemonCronJob.Name)
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

func (r *DaemonCronJobReconciler) broadcastArgs(daemonCronJob *daemonjobv1.DaemonCronJob) *DaemonCronJobBroadcastArgs {
	return NewDaemonCronJobBroadcastArgs(daemonCronJob, r.BroadcastRole, r.BroadcastImage)
}

func (r *DaemonCronJobReconciler) applyCronJob(ctx context.Context, daemonCronJob *daemonjobv1.DaemonCronJob) error {
	var (
		logger         = logf.FromContext(ctx)
		createOrUpdate = func(obj client.Object, kind string) error {
			l := logger.WithValues("kind", kind, "name", obj.GetName())

			if err := CreateOrUpdate(ctx, r.Client, r.Scheme, daemonCronJob, obj); err != nil {
				l.Error(err, "Unable to create or update")
				return err
			}
			l.Info("Created or updated")
			return nil
		}
		args = r.broadcastArgs(daemonCronJob)
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
	if err := createOrUpdate(args.CronJob(), "CronJob"); err != nil {
		return err
	}
	return nil
}

func (r *DaemonCronJobReconciler) updateStatus(ctx context.Context, daemonCronJob *daemonjobv1.DaemonCronJob) error {
	var (
		status = NewStatus(
			&daemonCronJob.Status.Conditions,
			daemonjobv1.TypeDaemonCronJobAvailable,
			daemonjobv1.TypeDaemonCronJobAvailable, // unused
			daemonjobv1.TypeDaemonCronJobDegraded,
			daemonjobv1.ReasonReconciling,
		)
		updateStatus = func() error {
			return r.Status().Update(ctx, daemonCronJob)
		}
		args = r.broadcastArgs(daemonCronJob)
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
	if exist, err := Exist[*batchv1.CronJob](ctx, r.Client, client.ObjectKeyFromObject(args.CronJob())); !exist {
		if err != nil {
			return err
		}
		status.SetNotFound("CronJob")

		return updateStatus()
	}

	status.SetPair(true, daemonjobv1.ReasonOK, "")
	return updateStatus()
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonCronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&daemonjobv1.DaemonCronJob{}).
		Named("daemoncronjob").
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&batchv1.CronJob{}).
		Watches(
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				logger := logf.FromContext(ctx).WithValues("node", obj.GetName())
				xs, err := ListDaemonCronJobWorkerJobsByNode(ctx, r.Client, obj.GetName())
				if err != nil {
					logger.Error(err, "Unable to list worker jobs by node")
					return []reconcile.Request{}
				}

				var requests []reconcile.Request
				for _, x := range xs.Items {
					if daemonCronJobName, ok := x.Labels[DaemonJobLabelDaemonCronJobName]; ok {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: x.Namespace,
								Name:      daemonCronJobName,
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
