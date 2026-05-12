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
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = daemonJobControllerTest{}.run()

type daemonJobControllerTest struct{}

func (d daemonJobControllerTest) run() bool {
	return ControllerTest{
		Name:            "DaemonJob Controller",
		NamespacePrefix: "daemonjob",
		Contexts: []ControllerTestContext{
			d.reconcileNormalTest(),
		},
		BeforeEach: func() {
			By("Creating ClusterRole broad-cast")
			Expect(k8sClient.Create(ctx, d.broadcastRole())).To(Succeed())
		},
		AfterEach: func() {
			By("Cleanup ClusterRole broad-cast")
			Expect(k8sClient.Delete(ctx, d.broadcastRole())).To(Succeed())
		},
	}.Run()
}

func (daemonJobControllerTest) broadcastImage() string {
	return "ghcr.io/berquerant/daemonjob/broadcast:dev"
}

func (daemonJobControllerTest) broadcastRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "broadcast-role",
		},
	}
}

func (d daemonJobControllerTest) newReconciler() *DaemonJobReconciler {
	return &DaemonJobReconciler{
		Client:         k8sClient,
		Scheme:         k8sClient.Scheme(),
		BroadcastImage: d.broadcastImage(),
		BroadcastRole:  d.broadcastRole().Name,
	}
}

func (daemonJobControllerTest) assertShouldHaveCommonLabels(labels map[string]string, daemonJobName string) {
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(DaemonJobLabelDaemonJobName, daemonJobName))
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(DaemonJobLabelRole, DaemonJobLabelRoleBroadcast))
}

func (daemonJobControllerTest) newNormalDaemonJob(namespace, name string) *daemonjobv1.DaemonJob {
	return &daemonjobv1.DaemonJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: daemonjobv1.DaemonJobSpec{
			BroadcastJobSpec: daemonjobv1.DaemonJobBroadcastJobSpec{
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("100Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("100Mi"),
					},
				},
			},
			JobTemplate: daemonjobv1.DaemonJobTemplateSpec{
				Metadata: daemonjobv1.DaemonJobTemplateMeta{
					Annotations: map[string]string{
						"app-name": "tmpl-sample",
					},
					Labels: map[string]string{
						"app-name": "tmpl-sample",
					},
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: new(int32(0)),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:  "main",
									Image: "debian:trixie-slim",
									Command: []string{
										"bash",
										"-c",
										`sleep 2
echo Done!`,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d daemonJobControllerTest) expectChilrenCreated(daemonJob *daemonjobv1.DaemonJob) {
	var (
		namespace    = daemonJob.Namespace
		resourceName = daemonJob.Name
		// namespacedName = types.NamespacedName{
		// 	Namespace: namespace,
		// 	Name:      resourceName,
		// }
		by = func(msg string) {
			By(fmt.Sprintf("%s for DaemonJob %s in namespace %s", msg, resourceName, namespace))
		}
	)

	by("Should successfully create a ServiceAccount")
	saList, err := ListDaemonJobBroadcastServiceAccounts(ctx, k8sClient, namespace, resourceName)
	Expect(err).To(Succeed())
	Expect(saList.Items).To(HaveLen(1))
	sa := saList.Items[0]
	d.assertShouldHaveCommonLabels(sa.Labels, resourceName)

	by("Should successfully create a ClusterRoleBinding")
	crbList, err := ListDaemonJobBroadcastClusterRoleBindings(ctx, k8sClient, namespace, resourceName)
	Expect(err).To(Succeed())
	Expect(crbList.Items).To(HaveLen(1))
	crb := crbList.Items[0]
	d.assertShouldHaveCommonLabels(crb.Labels, resourceName)
	Expect(crb.RoleRef.Name).To(Equal(d.broadcastRole().Name))
	Expect(crb.Subjects).To(HaveLen(1))
	subject := crb.Subjects[0]
	Expect(subject.Namespace).To(Equal(sa.Namespace))
	Expect(subject.Name).To(Equal(sa.Name))

	by("Should successfully create a broadcast Job")
	broadcastJobList, err := ListDaemonJobBroadcastJobs(ctx, k8sClient, namespace, resourceName)
	Expect(err).To(Succeed())
	Expect(broadcastJobList.Items).To(HaveLen(1))
	broadcastJob := broadcastJobList.Items[0]

	Expect(broadcastJob.Name).To(Equal(resourceName + "-" + DaemonJobResourceSuffix))
	Expect(broadcastJob.Annotations).To(Equal(daemonJob.Spec.JobTemplate.Metadata.Annotations))
	d.assertShouldHaveCommonLabels(broadcastJob.Labels, resourceName)
	for k, v := range daemonJob.Spec.JobTemplate.Metadata.Labels {
		Expect(broadcastJob.Labels).To(HaveKeyWithValue(k, v))
	}

	Expect(broadcastJob.Spec.Template.Annotations).To(Equal(daemonJob.Spec.JobTemplate.Metadata.Annotations))
	d.assertShouldHaveCommonLabels(broadcastJob.Spec.Template.Labels, resourceName)
	for k, v := range daemonJob.Spec.JobTemplate.Metadata.Labels {
		Expect(broadcastJob.Spec.Template.Labels).To(HaveKeyWithValue(k, v))
	}

	Expect(broadcastJob.Spec.Template.Spec.ServiceAccountName).To(Equal(sa.Name))

	by("The broadcast Job should have the appropriate container")
	Expect(broadcastJob.Spec.Template.Spec.Containers).To(HaveLen(1))
	container := broadcastJob.Spec.Template.Spec.Containers[0]
	Expect(container.Name).To(Equal(DaemonJobBroadcastContainerName))
	Expect(container.Image).To(Equal(d.broadcastImage()))
	Expect(container.Resources).To(Equal(daemonJob.Spec.BroadcastJobSpec.Resources))
}
