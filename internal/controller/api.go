// Copyright 2026 berquerant.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"slices"

	"github.com/berquerant/daemonjob/internal/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Get[T client.Object](ctx context.Context, c client.Client, key client.ObjectKey, opts ...client.GetOption) (T, error) {
	t := util.EnsureInstance[T]()
	if err := c.Get(ctx, key, t, opts...); err != nil {
		return t, err
	}
	return t, nil
}

func List[T client.ObjectList](ctx context.Context, c client.Client, opts ...client.ListOption) (T, error) {
	t := util.EnsureInstance[T]()
	if err := c.List(ctx, t, opts...); err != nil {
		return t, err
	}
	return t, nil
}

func ListDaemonJobClusterRoleBindings(ctx context.Context, c client.Client, namespace, daemonJobName string) (*rbacv1.ClusterRoleBindingList, error) {
	return List[*rbacv1.ClusterRoleBindingList](ctx, c, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonJobName: daemonJobName,
			DaemonJobLabelRole:          DaemonJobLabelRoleBroadcast,
			DaemonJobLabelNamespace:     namespace,
		}),
	})
}

func ListDaemonCronJobClusterRoleBindings(ctx context.Context, c client.Client, namespace, daemonCronJobName string) (*rbacv1.ClusterRoleBindingList, error) {
	return List[*rbacv1.ClusterRoleBindingList](ctx, c, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonCronJobName: daemonCronJobName,
			DaemonJobLabelRole:              DaemonJobLabelRoleBroadcast,
			DaemonJobLabelNamespace:         namespace,
		}),
	})
}

func ListJobPods(ctx context.Context, c client.Client, job *batchv1.Job) (*corev1.PodList, error) {
	const uidLabel = "batch.kubernetes.io/controller-uid"
	return List[*corev1.PodList](ctx, c, &client.ListOptions{
		Namespace: job.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			uidLabel: job.GetLabels()[uidLabel],
		}),
	})
}

func ListDaemonJobBroadcastJobs(ctx context.Context, c client.Client, namespace, daemonJobName string) (*batchv1.JobList, error) {
	return List[*batchv1.JobList](ctx, c, &client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonJobName: daemonJobName,
			DaemonJobLabelRole:          DaemonJobLabelRoleBroadcast,
		}),
	})
}

func ListDaemonCronJobBroadcastCronJobs(ctx context.Context, c client.Client, namespace, daemonCronJobName string) (*batchv1.CronJobList, error) {
	return List[*batchv1.CronJobList](ctx, c, &client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonCronJobName: daemonCronJobName,
			DaemonJobLabelRole:              DaemonJobLabelRoleBroadcast,
		}),
	})
}

func ListDaemonCronJobSetCronJobs(ctx context.Context, c client.Client, namespace, daemonCronJobSetName string) (*batchv1.CronJobList, error) {
	return List[*batchv1.CronJobList](ctx, c, &client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonCronJobSetName: daemonCronJobSetName,
			DaemonJobLabelRole:                 DaemonJobLabelRoleWorker,
		}),
	})
}

func ListDaemonJobWorkerJobs(ctx context.Context, c client.Client, namespace, daemonJobName string) (*batchv1.JobList, error) {
	return List[*batchv1.JobList](ctx, c, &client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonJobName: daemonJobName,
			DaemonJobLabelRole:          DaemonJobLabelRoleWorker,
		}),
	})
}

func ListDaemonCronJobWorkerJobs(ctx context.Context, c client.Client, namespace, daemonCronJobName string) (*batchv1.JobList, error) {
	return List[*batchv1.JobList](ctx, c, &client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonCronJobName: daemonCronJobName,
			DaemonJobLabelRole:              DaemonJobLabelRoleWorker,
		}),
	})
}

func ListWorkerJobsByNode(ctx context.Context, c client.Client, nodeName string) (*batchv1.JobList, error) {
	return List[*batchv1.JobList](ctx, c, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelRole: DaemonJobLabelRoleWorker,
			DaemonJobLabelNode: nodeName,
		}),
	})
}

func ListDaemonJobWorkerJobsByNode(ctx context.Context, c client.Client, nodeName string) (*batchv1.JobList, error) {
	xs, err := ListWorkerJobsByNode(ctx, c, nodeName)
	if err != nil {
		return nil, err
	}
	xs.Items = slices.DeleteFunc(xs.Items, func(x batchv1.Job) bool {
		_, ok := x.Labels[DaemonJobLabelDaemonJobName]
		return !ok
	})
	return xs, nil
}

func ListDaemonCronJobWorkerJobsByNode(ctx context.Context, c client.Client, nodeName string) (*batchv1.JobList, error) {
	xs, err := ListWorkerJobsByNode(ctx, c, nodeName)
	if err != nil {
		return nil, err
	}
	xs.Items = slices.DeleteFunc(xs.Items, func(x batchv1.Job) bool {
		_, ok := x.Labels[DaemonJobLabelDaemonCronJobName]
		return !ok
	})
	return xs, nil
}

func ListWorkerCronJobsByNode(ctx context.Context, c client.Client, nodeName string) (*batchv1.CronJobList, error) {
	return List[*batchv1.CronJobList](ctx, c, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelRole: DaemonJobLabelRoleWorker,
			DaemonJobLabelNode: nodeName,
		}),
	})
}

func ListDaemonCronJobSetCronJobsByNode(ctx context.Context, c client.Client, nodeName string) (*batchv1.CronJobList, error) {
	xs, err := List[*batchv1.CronJobList](ctx, c, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelRole: DaemonJobLabelRoleWorker,
			DaemonJobLabelNode: nodeName,
		}),
	})
	if err != nil {
		return nil, err
	}
	xs.Items = slices.DeleteFunc(xs.Items, func(x batchv1.CronJob) bool {
		_, ok := x.Labels[DaemonJobLabelDaemonCronJobSetName]
		return !ok
	})
	return xs, nil
}

func ListDaemonJobBroadcastServiceAccounts(ctx context.Context, c client.Client, namespace, daemonJobName string) (*corev1.ServiceAccountList, error) {
	return List[*corev1.ServiceAccountList](ctx, c, &client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonJobName: daemonJobName,
			DaemonJobLabelRole:          DaemonJobLabelRoleBroadcast,
		}),
	})
}

func ListDaemonCronJobBroadcastServiceAccounts(ctx context.Context, c client.Client, namespace, daemonCronJobName string) (*corev1.ServiceAccountList, error) {
	return List[*corev1.ServiceAccountList](ctx, c, &client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonCronJobName: daemonCronJobName,
			DaemonJobLabelRole:              DaemonJobLabelRoleBroadcast,
		}),
	})
}

func ListDaemonJobBroadcastClusterRoleBindings(ctx context.Context, c client.Client, namespace, daemonJobName string) (*rbacv1.ClusterRoleBindingList, error) {
	return List[*rbacv1.ClusterRoleBindingList](ctx, c, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonJobName: daemonJobName,
			DaemonJobLabelRole:          DaemonJobLabelRoleBroadcast,
			DaemonJobLabelNamespace:     namespace,
		}),
	})
}

func ListDaemonCronJobBroadcastClusterRoleBindings(ctx context.Context, c client.Client, namespace, daemonCronJobName string) (*rbacv1.ClusterRoleBindingList, error) {
	return List[*rbacv1.ClusterRoleBindingList](ctx, c, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			DaemonJobLabelDaemonCronJobName: daemonCronJobName,
			DaemonJobLabelRole:              DaemonJobLabelRoleBroadcast,
			DaemonJobLabelNamespace:         namespace,
		}),
	})
}

func Exist[T client.Object](ctx context.Context, c client.Client, key client.ObjectKey, opts ...client.GetOption) (bool, error) {
	_, err := Get[T](ctx, c, key, opts...)
	switch {
	case err == nil:
		return true, nil
	case apierrors.IsNotFound(err):
		return false, nil
	default:
		return false, err
	}
}

// CreateOrUpdate with SetControllerReference.
func CreateOrUpdate(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner metav1.Object, obj client.Object) error {
	_, err := ctrl.CreateOrUpdate(ctx, c, obj, func() error {
		return ctrl.SetControllerReference(owner, obj, scheme)
	})
	return err
}

// Create with SetControllerReference.
func Create(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner metav1.Object, obj client.Object) error {
	if err := c.Create(ctx, obj); err != nil {
		return err
	}
	return ctrl.SetControllerReference(owner, obj, scheme)
}

func ContainsFinalizer(o client.Object, finalizer string) bool {
	return controllerutil.ContainsFinalizer(o, finalizer)
}

func RemoveFinalizerIfExist(ctx context.Context, c client.Client, o client.Object, finalizer string) error {
	if ContainsFinalizer(o, finalizer) {
		controllerutil.RemoveFinalizer(o, finalizer)
		return c.Update(ctx, o)
	}
	return nil
}

func AddFinalizerIfNotExist(ctx context.Context, c client.Client, o client.Object, finalizer string) error {
	if ContainsFinalizer(o, finalizer) {
		return nil
	}
	controllerutil.AddFinalizer(o, finalizer)
	return c.Update(ctx, o)
}
