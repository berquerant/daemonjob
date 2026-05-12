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
	"sort"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (d daemonCronJobSetControllerTest) reconcileNormalTest() ControllerTestContext {
	const resourceName = "sample"

	return ControllerTestContext{
		Name: "When reconciling a resource",
		BeforeEach: func(namespace string) {
			By("Create worker node 1")
			Expect(d.addNode(1, nil)).To(Succeed())

			daemonCronJobSet := d.newNormalDaemonCronJobSet(namespace, resourceName)
			By("Creating the resource")
			Expect(k8sClient.Create(ctx, daemonCronJobSet)).To(Succeed())
		},
		AfterEach: func(namespace string) {
			By("Cleanup the specific resource instances DaemonCronJobSet")
			Expect(k8sClient.DeleteAllOf(ctx, &daemonjobv1.DaemonCronJobSet{}, &client.DeleteAllOfOptions{
				ListOptions: client.ListOptions{
					Namespace: namespace,
				},
			})).To(Succeed())
			By("Delete worker nodes")
			nodes, err := List[*corev1.NodeList](ctx, k8sClient)
			Expect(err).To(Succeed())
			for _, x := range nodes.Items {
				Expect(k8sClient.Delete(ctx, &x)).To(Succeed())
			}
		},
		Test: func(namespace string) {
			namespacedName := types.NamespacedName{
				Namespace: namespace,
				Name:      resourceName,
			}

			It("Should successfully reconcile the resource", func() {
				controllerReconciler := d.newReconciler()
				reconcile := func(name string) error {
					_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: namespace,
							Name:      name,
						},
					})
					return err
				}

				By("Reconciling the created resource")
				Expect(reconcile(resourceName)).To(Succeed())

				By("Should update the status of the resource")
				daemonCronJobSet, err := Get[*daemonjobv1.DaemonCronJobSet](ctx, k8sClient, namespacedName)
				Expect(err).To(Succeed())
				Expect(daemonCronJobSet.Status.Conditions).NotTo(BeEmpty())

				By("Should create CronJobs")
				cronJobs, err := ListDaemonCronJobSetCronJobs(ctx, k8sClient, namespace, resourceName)
				Expect(err).To(Succeed())
				Expect(cronJobs.Items).To(HaveLen(1))

				cronJob := cronJobs.Items[0]
				Expect(cronJob.Name).To(Equal(resourceName + "-" + d.nodeName(1) + "-" + DaemonCronJobSetResourceSuffix))
				Expect(cronJob.Labels[DaemonJobLabelNode]).To(Equal(d.nodeName(1)))
				Expect(cronJob.Annotations).To(Equal(daemonCronJobSet.Spec.CronJobTemplate.Metadata.Annotations))
				for k, v := range daemonCronJobSet.Spec.CronJobTemplate.Metadata.Labels {
					Expect(cronJob.Labels).To(HaveKeyWithValue(k, v))
				}
				Expect(cronJob.Spec.Schedule).To(Equal(daemonCronJobSet.Spec.CronJobTemplate.Spec.Schedule))

				By("Add worker node 2")
				Expect(d.addNode(2, map[string]string{
					"color": "red",
				})).To(Succeed())

				By("Reconciling again")
				Expect(reconcile(resourceName)).To(Succeed())

				By("Should create a CronJob on worker node 2")
				nodeNames, err := d.listCronJobNodes(namespace, resourceName)
				Expect(err).To(Succeed())
				sort.Strings(nodeNames)
				Expect(nodeNames).To(Equal([]string{d.nodeName(1), d.nodeName(2)}))

				const resourceName2 = "sample-2"
				daemonCronJobSet2 := d.newNormalDaemonCronJobSet(namespace, resourceName2)
				daemonCronJobSet2.Spec.NodeSelector = map[string]string{
					"color": "red",
				}

				By("Create a new DaemonCronJobSet")
				Expect(k8sClient.Create(ctx, daemonCronJobSet2)).To(Succeed())

				By("Reconciling the new DaemonCronJobSet")
				Expect(reconcile(resourceName2)).To(Succeed())

				By("Should only create a CronJob on worker node 2")
				nodeNames, err = d.listCronJobNodes(namespace, resourceName2)
				Expect(err).To(Succeed())
				Expect(nodeNames).To(Equal([]string{d.nodeName(2)}))

				By("Delete worker node 2")
				Expect(d.deleteNode(2)).To(Succeed())

				By("Reconciling resources")
				Expect(reconcile(resourceName)).To(Succeed())
				Expect(reconcile(resourceName2)).To(Succeed())

				By("Cronjobs should be deleted on worker node 2")
				nodeNames, err = d.listCronJobNodes(namespace, resourceName)
				Expect(err).To(Succeed())
				Expect(nodeNames).To(Equal([]string{d.nodeName(1)}))
				nodeNames, err = d.listCronJobNodes(namespace, resourceName2)
				Expect(err).To(Succeed())
				Expect(nodeNames).To(BeEmpty())
			})
		},
	}
}
