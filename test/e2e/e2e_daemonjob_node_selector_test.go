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

func daemonJobNodeSelectorBeforeAddNode() utils.Testcase {
	return utils.Testcase{
		Name:         "DaemonJob/NodeSelectorBeforeAddNode",
		KustomizeDir: "node_selector",
		BeforeAddNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemonjob-sample"
			var namespace = a.Namespace

			It("Should successfully handle node_selector manifest", func() {
				By("Create the resource")
				utils.EventuallyFound(namespace, resourceName, "daemonjob").To(Succeed())

				By("Create the broadcast Job")
				broadcastJobName := resourceName + "-" + controller.DaemonJobResourceSuffix
				utils.EventuallyFound(namespace, broadcastJobName, "job").To(Succeed())

				By("Complete the broadcast Job")
				utils.EventuallyJobComplete(namespace, broadcastJobName).To(Succeed())

				By("The resource state should be changed into Complete")
				utils.EventuallyDaemonJobState(namespace, resourceName, "Complete").To(Succeed())

				By("No worker jobs should be created")
				Consistently(func(g Gomega) {
					workerJobNames, err := utils.GetDaemonJobWorkerJobs(namespace, resourceName)
					g.Expect(err).To(Succeed())
					g.Expect(workerJobNames).To(BeEmpty())
				}).To(Succeed())
			})
		},
	}
}

func daemonJobNodeSelectorAfterAddNode() utils.Testcase {
	return utils.Testcase{
		Name:         "DaemonJob/NodeSelectorAfterAddNode",
		KustomizeDir: "node_selector",
		AfterAddNodeInit: func(a *utils.TestcaseArgs) {
			It("Reapply manifest", func() {
				Expect(a.Reapply()).To(Succeed())
			})
		},
		AfterAddNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemonjob-sample"
			var namespace = a.Namespace

			It("Should successfully handle node_selector manifest", func() {
				By("Create the broadcast Job")
				broadcastJobName := resourceName + "-" + controller.DaemonJobResourceSuffix
				utils.EventuallyFound(namespace, broadcastJobName, "job").To(Succeed())

				By("Complete the broadcast Job")
				utils.EventuallyJobComplete(namespace, broadcastJobName).To(Succeed())

				By("A worker Job should be created")
				var workerJobName string
				Eventually(func(g Gomega) {
					out, err := utils.GetDaemonJobWorkerJobs(namespace, resourceName)
					g.Expect(err).To(Succeed())
					g.Expect(out).To(HaveLen(1))
					workerJobName = out[0]
				}).To(Succeed())

				By("The worker Job should be created in the worker node 1")
				Eventually(func(g Gomega) {
					pods, err := utils.GetJobPodNames(namespace, workerJobName)
					g.Expect(err).To(Succeed())
					g.Expect(pods).To(HaveLen(1))
					nodeName, err := utils.Run(utils.KubectlCmd("-n", namespace,
						"get", "pod", pods[0], "-o=jsonpath={.spec.nodeName}"))
					g.Expect(err).To(Succeed())
					g.Expect(nodeName).To(Equal(utils.NodeName(1)))
				}).To(Succeed())

				By("The worker Job should be completed")
				utils.EventuallyJobComplete(namespace, workerJobName).To(Succeed())

			})
		},
	}
}
