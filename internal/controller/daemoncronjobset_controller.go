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
	"reflect"
	"slices"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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

// DaemonCronJobSetReconciler reconciles a DaemonCronJobSet object
type DaemonCronJobSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemoncronjobsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemoncronjobsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=daemonjob.berquerant.github.io,resources=daemoncronjobsets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs/status,verbs=get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DaemonCronJobSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *DaemonCronJobSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	logger.Info("Start to reconcile")
	daemonCronJobSet, err := Get[*daemonjobv1.DaemonCronJobSet](ctx, r.Client, req.NamespacedName)
	if apierrors.IsNotFound(err) {
		logger.Info("DaemonCronJobSet is deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "Unable to get DaemonCronJobSet")
		return ctrl.Result{}, err
	}
	if !daemonCronJobSet.DeletionTimestamp.IsZero() {
		logger.Info("Deleting the DaemonCronJobSet")
		// Finalizers are deleting the DaemonCronJobSet.
		// https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
		return ctrl.Result{}, nil
	}

	if err := r.deleteCronJobsOnMismatchedNode(ctx, daemonCronJobSet); err != nil {
		logger.Error(err, "Unable to delete the worker cronjobs on deleted nodes")
	}

	if err := r.applyCronJobs(ctx, daemonCronJobSet); err != nil {
		logger.Error(err, "Unable to apply CronJobs")
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, daemonCronJobSet); err != nil {
		logger.Error(err, "Unable to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DaemonCronJobSetReconciler) args(daemonCronJobSet *daemonjobv1.DaemonCronJobSet) *DaemonCronJobSetArgs {
	return NewDaemonCronJobSetArgs(daemonCronJobSet)
}

func (r *DaemonCronJobSetReconciler) deleteCronJobsOnMismatchedNode(ctx context.Context, daemonCronJobSet *daemonjobv1.DaemonCronJobSet) error {
	logger := logf.FromContext(ctx)

	nodeNames, err := r.args(daemonCronJobSet).ListNodes(ctx, r.Client)
	if err != nil {
		return err
	}

	cronJobs, err := ListDaemonCronJobSetCronJobs(ctx, r.Client, daemonCronJobSet.Namespace, daemonCronJobSet.Name)
	if err != nil {
		return err
	}

	for _, cronJob := range cronJobs.Items {
		if node, ok := cronJob.Labels[DaemonJobLabelNode]; ok {
			l := logger.WithValues("cronjob", cronJob.Name, "node", node)
			if !slices.Contains(nodeNames, node) {
				if err := r.Delete(ctx, &cronJob); err != nil {
					l.Error(err, "Unable to delete cronjob")
					return err
				}
				l.Info("CronJob is deleted")
			}
		}
	}

	return nil
}

func (r *DaemonCronJobSetReconciler) applyCronJobs(ctx context.Context, daemonCronJobSet *daemonjobv1.DaemonCronJobSet) error {
	var (
		logger = logf.FromContext(ctx)
		arg    = r.args(daemonCronJobSet)
	)

	cronJobs, err := arg.CronJobs(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Unable to generate CronJobs")
		return err
	}
	nodeNames := make([]string, len(cronJobs))
	for i, cronJob := range cronJobs {
		nodeNames[i] = cronJob.Labels[DaemonJobLabelNode]
	}
	logger.Info("Desired node", "numberNodes", len(cronJobs), "nodes", nodeNames)

	for _, cronJob := range cronJobs {
		l := logger.WithValues("cronjob", cronJob.Name)
		if err := CreateOrUpdate(ctx, r.Client, r.Scheme, daemonCronJobSet, cronJob); err != nil {
			l.Error(err, "Unable to create or update CronJob")
			return err
		}
		l.Info("Created or updated")
	}

	return nil
}

func (r *DaemonCronJobSetReconciler) updateStatus(ctx context.Context, daemonCronJobSet *daemonjobv1.DaemonCronJobSet) error {
	var (
		status = NewStatus(
			&daemonCronJobSet.Status.Conditions,
			daemonjobv1.TypeDaemonCronJobSetAvailable,
			daemonjobv1.TypeDaemonCronJobSetDegraded, // unused
			daemonjobv1.TypeDaemonCronJobSetDegraded,
			daemonjobv1.ReasonReconciling,
		)
		updateStatus = func() error {
			return r.Status().Update(ctx, daemonCronJobSet)
		}
	)

	nodeNames, err := r.args(daemonCronJobSet).ListNodes(ctx, r.Client)
	if err != nil {
		return err
	}
	daemonCronJobSet.Status.Desired = len(nodeNames)

	cronJobs, err := ListDaemonCronJobSetCronJobs(ctx, r.Client, daemonCronJobSet.Namespace, daemonCronJobSet.Name)
	if err != nil {
		return err
	}
	daemonCronJobSet.Status.Available = len(cronJobs.Items)

	switch {
	case daemonCronJobSet.Status.Desired < daemonCronJobSet.Status.Available:
		status.SetPair(false, daemonjobv1.ReasonReconciling, "Too much cronjobs")
	case daemonCronJobSet.Status.Desired > daemonCronJobSet.Status.Available:
		status.SetPair(false, daemonjobv1.ReasonReconciling, "Too few cronjobs")
	default:
		status.SetPair(true, daemonjobv1.ReasonOK, "")
	}

	return updateStatus()
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonCronJobSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&daemonjobv1.DaemonCronJobSet{}).
		Named("daemoncronjobset").
		Owns(&batchv1.CronJob{}).
		Watches(
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				logger := logf.FromContext(ctx).WithValues("node", obj.GetName())

				if !obj.GetDeletionTimestamp().IsZero() {
					//
					// Node is deleted.
					//
					logger.Info("Node deletion is detected!")
					xs, err := ListDaemonCronJobSetCronJobsByNode(ctx, r.Client, obj.GetName())
					if err != nil {
						logger.Error(err, "Unable to list worker cronjobs by node")
						return []reconcile.Request{}
					}
					var requests []reconcile.Request
					for _, x := range xs.Items {
						if daemonCronJobSetName, ok := x.Labels[DaemonJobLabelDaemonCronJobSetName]; ok {
							requests = append(requests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Namespace: x.Namespace,
									Name:      daemonCronJobSetName,
								},
							})
						}
					}
					logger.Info("Node deletion will trigger reconcile", "numberReconcile", len(requests))
					return requests
				}
				//
				// Node is added or updated.
				//
				logger.Info("Node addition or update is detected!")
				xs, err := List[*daemonjobv1.DaemonCronJobSetList](ctx, r.Client)
				if err != nil {
					logger.Error(err, "Unable to list DaemonCronJobSet")
					return []reconcile.Request{}
				}
				requests := make([]reconcile.Request, len(xs.Items))
				for i, x := range xs.Items {
					requests[i] = reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: x.Namespace,
							Name:      x.Name,
						},
					}
				}
				logger.Info("Node addition or update will trigger reconcile", "numberReconcile", len(requests))
				return requests
			}),
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					// Watch the changes of node labels to process DaemonCronJobSet.spec.nodeSelector.
					oldLabels := e.ObjectOld.GetLabels()
					newLabels := e.ObjectNew.GetLabels()
					return !reflect.DeepEqual(oldLabels, newLabels)
				},
				CreateFunc: func(_ event.CreateEvent) bool {
					return true
				},
				DeleteFunc: func(_ event.DeleteEvent) bool {
					return true
				},
				GenericFunc: func(_ event.GenericEvent) bool {
					return false
				},
			}),
		).
		Complete(r)
}
