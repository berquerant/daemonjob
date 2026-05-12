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

package e2e

import (
	"github.com/berquerant/daemonjob/internal/controller"
	"github.com/berquerant/daemonjob/test/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func daemonCronJobSetSamplesBeforeAddNode() utils.Testcase {
	return utils.Testcase{
		Name:         "DaemonCronJobSet/SamplesBeforeAddNode",
		KustomizeDir: "samples",
		BeforeAddNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemoncronjobset-sample"
			var namespace = a.Namespace

			It("Should successfully handle sample manifest", func() {
				By("Create the resource")
				utils.EventuallyFound(namespace, resourceName, "daemoncronjobset").To(Succeed())

				By("Create a CronJob")
				utils.EventuallyDaemonCronJobSetCronJobNodes(namespace, resourceName, utils.NodeName(0)).To(Succeed())

				By("The resource state should be changed into Available")
				utils.EventuallyDaemonCronJobSetState(namespace, resourceName, "Available", 1, 1).To(Succeed())

				By("Worker Jobs should be created")
				var workerJobNames []string
				Eventually(func(g Gomega) {
					xs, err := utils.GetDaemonCronJobSetJobsByNode(namespace, resourceName, utils.NodeName(0))
					g.Expect(err).To(Succeed())
					g.Expect(xs).NotTo(BeEmpty())
					workerJobNames = xs
				}).To(Succeed())

				By("Worker Jobs should be completed")
				for _, x := range workerJobNames {
					utils.EventuallyJobComplete(namespace, x).To(Succeed())
				}
			})
		},
		AfterAddNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemoncronjobset-sample"
			var namespace = a.Namespace

			It("Should successfully handle sample manifest", func() {
				By("CronJob on worker node 0 should be remained")
				Consistently(func(g Gomega) {
					xs, err := utils.GetNamesByLabelSelector(namespace, "cronjob",
						controller.DaemonJobLabelDaemonCronJobSetName, resourceName,
						controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
						controller.DaemonJobLabelNode, utils.NodeName(0),
					)
					g.Expect(err).To(Succeed())
					g.Expect(xs).To(HaveLen(1))
				}).To(Succeed())

				By("Create a CronJob on worker node 1")
				utils.EventuallyDaemonCronJobSetCronJobNodes(namespace, resourceName, utils.NodeName(0), utils.NodeName(1)).To(Succeed())

				By("The resource state should be changed into Available")
				utils.EventuallyDaemonCronJobSetState(namespace, resourceName, "Available", 2, 2).To(Succeed())

				By("Worker Jobs should be created on worker node 1")
				var workerJobNames []string
				Eventually(func(g Gomega) {
					xs, err := utils.GetDaemonCronJobSetJobsByNode(namespace, resourceName, utils.NodeName(1))
					g.Expect(err).To(Succeed())
					g.Expect(xs).NotTo(BeEmpty())
					workerJobNames = xs
				}).To(Succeed())

				By("Worker Jobs on worker node 1 should be completed")
				for _, x := range workerJobNames {
					utils.EventuallyJobComplete(namespace, x).To(Succeed())
				}
			})
		},
		AfterRemoveNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemoncronjobset-sample"
			var namespace = a.Namespace

			It("Should successfully handle sample manifest", func() {
				By("Cronjob on worker node 1 should be deleted")
				utils.EventuallyNotFoundByLabelSelector(namespace, "cronjob",
					controller.DaemonJobLabelDaemonCronJobSetName, resourceName,
					controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
					controller.DaemonJobLabelNode, utils.NodeName(1),
				).To((Succeed()))

				By("CronJob on worker node 0 should be remained")
				Consistently(func(g Gomega) {
					xs, err := utils.GetNamesByLabelSelector(namespace, "cronjob",
						controller.DaemonJobLabelDaemonCronJobSetName, resourceName,
						controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
						controller.DaemonJobLabelNode, utils.NodeName(0),
					)
					g.Expect(err).To(Succeed())
					g.Expect(xs).To(HaveLen(1))
				}).To(Succeed())

				By("The resoure status should be updated")
				utils.EventuallyDaemonCronJobSetState(namespace, resourceName, "Available", 1, 1).To(Succeed())
			})
		},
	}
}
