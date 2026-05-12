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

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = daemonCronJobControllerTest{}.run()

type daemonCronJobControllerTest struct{}

func (d daemonCronJobControllerTest) run() bool {
	return ControllerTest{
		Name:            "DaemonCronJob Controller",
		NamespacePrefix: "daemoncronjob",
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

func (daemonCronJobControllerTest) broadcastImage() string {
	return "ghcr.io/berquerant/daemonjob/broadcast:dev"
}

func (daemonCronJobControllerTest) broadcastRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "broadcast-role",
		},
	}
}

func (d daemonCronJobControllerTest) newReconciler() *DaemonCronJobReconciler {
	return &DaemonCronJobReconciler{
		Client:         k8sClient,
		Scheme:         k8sClient.Scheme(),
		BroadcastImage: d.broadcastImage(),
		BroadcastRole:  d.broadcastRole().Name,
	}
}

func (daemonCronJobControllerTest) assertShouldHaveCommonLabels(labels map[string]string, daemonCronJobName string) {
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(DaemonJobLabelDaemonCronJobName, daemonCronJobName))
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(DaemonJobLabelRole, DaemonJobLabelRoleBroadcast))
}

func (daemonCronJobControllerTest) newNormalDaemonCronJob(namespace, name string) *daemonjobv1.DaemonCronJob {
	return &daemonjobv1.DaemonCronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: daemonjobv1.DaemonCronJobSpec{
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
			CronJobTemplate: daemonjobv1.DaemonCronJobTemplateSpec{
				Metadata: daemonjobv1.DaemonCronJobTemplateMeta{
					Annotations: map[string]string{
						"app-name": "tmpl-sample",
					},
					Labels: map[string]string{
						"app-name": "tmpl-sample",
					},
				},
				Spec: batchv1.CronJobSpec{
					Schedule: "* * * * *",
					JobTemplate: batchv1.JobTemplateSpec{
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
			},
		},
	}
}

func (d daemonCronJobControllerTest) expectChilrenCreated(daemonCronJob *daemonjobv1.DaemonCronJob) {
	var (
		namespace    = daemonCronJob.Namespace
		resourceName = daemonCronJob.Name
		// namespacedName = types.NamespacedName{
		// 	Namespace: namespace,
		// 	Name:      resourceName,
		// }
		by = func(msg string) {
			By(fmt.Sprintf("%s for DaemonCronJob %s in namespace %s", msg, resourceName, namespace))
		}
	)

	by("Should successfully create a ServiceAccount")
	saList, err := ListDaemonCronJobBroadcastServiceAccounts(ctx, k8sClient, namespace, resourceName)
	Expect(err).To(Succeed())
	Expect(saList.Items).To(HaveLen(1))
	sa := saList.Items[0]
	d.assertShouldHaveCommonLabels(sa.Labels, resourceName)

	by("Should successfully create a ClusterRoleBinding")
	crbList, err := ListDaemonCronJobBroadcastClusterRoleBindings(ctx, k8sClient, namespace, resourceName)
	Expect(err).To(Succeed())
	Expect(crbList.Items).To(HaveLen(1))
	crb := crbList.Items[0]
	d.assertShouldHaveCommonLabels(crb.Labels, resourceName)
	Expect(crb.RoleRef.Name).To(Equal(d.broadcastRole().Name))
	Expect(crb.Subjects).To(HaveLen(1))
	subject := crb.Subjects[0]
	Expect(subject.Namespace).To(Equal(sa.Namespace))
	Expect(subject.Name).To(Equal(sa.Name))

	by("Should successfully create a broadcast CronJob")
	broadcastCronJobList, err := ListDaemonCronJobBroadcastCronJobs(ctx, k8sClient, namespace, resourceName)
	Expect(err).To(Succeed())
	Expect(broadcastCronJobList.Items).To(HaveLen(1))
	broadcastCronJob := broadcastCronJobList.Items[0]

	Expect(broadcastCronJob.Name).To(Equal(resourceName + "-" + DaemonCronJobResourceSuffix))
	Expect(broadcastCronJob.Annotations).To(Equal(daemonCronJob.Spec.CronJobTemplate.Metadata.Annotations))
	d.assertShouldHaveCommonLabels(broadcastCronJob.Labels, resourceName)
	for k, v := range daemonCronJob.Spec.CronJobTemplate.Metadata.Labels {
		Expect(broadcastCronJob.Labels).To(HaveKeyWithValue(k, v))
	}

	Expect(broadcastCronJob.Spec.JobTemplate.Annotations).To(Equal(daemonCronJob.Spec.CronJobTemplate.Metadata.Annotations))
	d.assertShouldHaveCommonLabels(broadcastCronJob.Spec.JobTemplate.Labels, resourceName)
	for k, v := range daemonCronJob.Spec.CronJobTemplate.Metadata.Labels {
		Expect(broadcastCronJob.Spec.JobTemplate.Labels).To(HaveKeyWithValue(k, v))
	}

	Expect(broadcastCronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName).To(Equal(sa.Name))

	by("The broadcast CronJob should have the appropriate container")
	Expect(broadcastCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers).To(HaveLen(1))
	container := broadcastCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
	Expect(container.Name).To(Equal(DaemonJobBroadcastContainerName))
	Expect(container.Image).To(Equal(d.broadcastImage()))
	Expect(container.Resources).To(Equal(daemonCronJob.Spec.BroadcastJobSpec.Resources))
}
