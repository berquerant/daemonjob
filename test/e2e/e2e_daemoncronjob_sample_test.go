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

func daemonCronJobSamplesBeforeAddNode() utils.Testcase {
	return utils.Testcase{
		Name:         "DaemonCronJob/SamplesBeforeAddNode",
		KustomizeDir: "samples",
		BeforeAddNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemoncronjob-sample"
			var namespace = a.Namespace

			It("Should successfully handle sample manifest", func() {
				By("Create the resource")
				utils.EventuallyFound(namespace, resourceName, "daemoncronjob").To(Succeed())

				By("Create the broadcast CronJob")
				broadcastCronJobName := resourceName + "-" + controller.DaemonCronJobResourceSuffix
				utils.EventuallyFound(namespace, broadcastCronJobName, "cronjob").To(Succeed())

				By("The resource state should be changed into Available")
				utils.EventualyDaemonCronJobState(namespace, resourceName, "Available").To(Succeed())

				By("A broadcast Job should be created and completed")
				Eventually(func(g Gomega) {
					broadcastJobNames, err := utils.GetDaemonCronJobBroadcastJobs(namespace, resourceName)
					g.Expect(err).To(Succeed())
					g.Expect(broadcastJobNames).NotTo(BeEmpty())
					for _, x := range broadcastJobNames {
						ok, err := utils.GetJobState(namespace, x, "Complete")
						g.Expect(err).To(Succeed())
						g.Expect(ok).To(BeTrue())
					}
				}).To(Succeed())

				By("A worker Job should be created")
				var workerJobName string
				Eventually(func(g Gomega) {
					workerJobNames, err := utils.GetDaemonCronJobWorkerJobs(namespace, resourceName)
					g.Expect(err).To(Succeed())
					g.Expect(workerJobNames).To(HaveLen(1))
					workerJobName = workerJobNames[0]
				}).To(Succeed())

				By("A worker Job should be completed")
				utils.EventuallyJobComplete(namespace, workerJobName).To(Succeed())

				By("Delete the resource")
				_, err := utils.Run(utils.KubectlCmd("-n", namespace, "delete", "daemoncronjob", resourceName))
				Expect(err).To(Succeed())

				By("ServiceAccount should be deleted")
				utils.EventuallyNotFoundByLabelSelector(namespace, "sa",
					controller.DaemonJobLabelDaemonCronJobName, resourceName,
					controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleBroadcast,
				).To(Succeed())

				By("ClusterRoleBinding should be deleted")
				utils.EventuallyNotFoundByLabelSelector("", "clusterrolebinding",
					controller.DaemonJobLabelDaemonCronJobName, resourceName,
					controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleBroadcast,
					controller.DaemonJobLabelNamespace, namespace,
				).To(Succeed())

				By("Broadcast CronJob should be deleted")
				utils.EventuallyNotFoundByLabelSelector(namespace, "cronjob",
					controller.DaemonJobLabelDaemonCronJobName, resourceName,
					controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleBroadcast,
				).To(Succeed())

				By("Worker Jobs should be deleted")
				utils.EventuallyNotFoundByLabelSelector(namespace, "job",
					controller.DaemonJobLabelDaemonCronJobName, resourceName,
					controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
				).To(Succeed())

			})
		},
	}
}

func daemonCronJobSamplesAfterAddNode() utils.Testcase {
	return utils.Testcase{
		Name:         "DaemonCronJob/SamplesAfterAddNode",
		KustomizeDir: "samples",
		AfterAddNodeInit: func(a *utils.TestcaseArgs) {
			It("Reapply manifest", func() {
				Expect(a.Reapply()).To(Succeed())
			})
		},
		AfterAddNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemoncronjob-sample"
			var namespace = a.Namespace

			It("Should successfully handle sample manifest", func() {
				By("Worker Jobs should be created")
				var workerJobNames []string
				Eventually(func(g Gomega) {
					out, err := utils.GetDaemonCronJobWorkerJobs(namespace, resourceName)
					g.Expect(err).To(Succeed())
					g.Expect(out).To(HaveLen(2))
					workerJobNames = out
				}).To(Succeed())

				By("Worker Jobs should be separate nodes")
				workerJobNodes := make([]string, len(workerJobNames))
				Eventually(func(g Gomega) {
					for i, name := range workerJobNames {
						pods, err := utils.GetJobPodNames(namespace, name)
						g.Expect(err).To(Succeed())
						g.Expect(pods).To(HaveLen(1))
						nodeName, err := utils.Run(utils.KubectlCmd("-n", namespace,
							"get", "pod", pods[0], "-o=jsonpath={.spec.nodeName}"))
						g.Expect(err).To(Succeed())
						workerJobNodes[i] = nodeName
					}
				}).To(Succeed())
				Expect(workerJobNames[0]).NotTo(Equal(workerJobNames[1]))

				By("Worker Jobs should be completed")
				for _, name := range workerJobNames {
					utils.EventuallyJobComplete(namespace, name).To(Succeed())
				}
			})
		},
		AfterRemoveNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemoncronjob-sample"
			var namespace = a.Namespace

			It("Should delete the job on the deleted node", func() {
				utils.EventuallyNotFoundByLabelSelector(namespace, "job",
					controller.DaemonJobLabelDaemonCronJobName, resourceName,
					controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
					controller.DaemonJobLabelNode, utils.NodeName(1),
				).To(Succeed())
			})
		},
	}
}
