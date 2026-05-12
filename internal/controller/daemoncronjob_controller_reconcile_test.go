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
	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (d daemonCronJobControllerTest) reconcileNormalTest() ControllerTestContext {
	const resourceName = "sample"

	return ControllerTestContext{
		Name: "When reconciling a resource",
		BeforeEach: func(namespace string) {
			daemonCronJob := d.newNormalDaemonCronJob(namespace, resourceName)
			By("Creating the resource")
			Expect(k8sClient.Create(ctx, daemonCronJob)).To(Succeed())
		},
		AfterEach: func(namespace string) {
			daemonCronJob, err := Get[*daemonjobv1.DaemonCronJob](ctx, k8sClient, types.NamespacedName{
				Namespace: namespace,
				Name:      resourceName,
			})
			Expect(err).To(Succeed())
			By("Cleanup the specific resource instance DaemonCronJob")
			Expect(k8sClient.Delete(ctx, daemonCronJob)).To(Succeed())
		},
		Test: func(namespace string) {
			namespacedName := types.NamespacedName{
				Namespace: namespace,
				Name:      resourceName,
			}

			It("Should successfully reconcile the resource", func() {
				controllerReconciler := d.newReconciler()

				By("Reconciling the created resource")
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: namespacedName,
				})
				Expect(err).To(Succeed())

				By("Should update the status of the resource")
				daemonCronJob, err := Get[*daemonjobv1.DaemonCronJob](ctx, k8sClient, namespacedName)
				Expect(err).To(Succeed())
				Expect(daemonCronJob.Status.Conditions).NotTo(BeEmpty())

				By("Should have appropriate schedule")
				broadcastCronJob, err := Get[*batchv1.CronJob](ctx, k8sClient, types.NamespacedName{
					Namespace: namespace,
					Name:      resourceName + "-" + DaemonCronJobResourceSuffix,
				})
				Expect(err).To(Succeed())
				Expect(broadcastCronJob.Spec.Schedule).To(Equal(daemonCronJob.Spec.CronJobTemplate.Spec.Schedule))

				d.expectChilrenCreated(daemonCronJob)
			})
		},
	}
}
