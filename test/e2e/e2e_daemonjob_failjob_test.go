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

func daemonJobFailjobBeforeAddNode() utils.Testcase {
	return utils.Testcase{
		Name:         "DaemonJob/FailjobBeforeAddNode",
		KustomizeDir: "failjob",
		BeforeAddNode: func(a *utils.TestcaseArgs) {
			const resourceName = "daemonjob-sample"
			var namespace = a.Namespace

			It("Should successfully handle failjob manifest", func() {
				By("Create the resource")
				utils.EventuallyFound(namespace, resourceName, "daemonjob").To(Succeed())

				By("Create the broadcast Job")
				broadcastJobName := resourceName + "-" + controller.DaemonJobResourceSuffix
				utils.EventuallyFound(namespace, broadcastJobName, "job").To(Succeed())

				By("Complete the broadcast Job")
				utils.EventuallyJobComplete(namespace, broadcastJobName).To(Succeed())

				By("Some worker Jobs are failed")
				var workerJobNames []string
				Eventually(func(g Gomega) {
					x, err := utils.GetDaemonJobWorkerJobs(namespace, resourceName)
					g.Expect(err).To(Succeed())
					g.Expect(x).NotTo(BeEmpty())
					workerJobNames = x
				}).To(Succeed())
				utils.EventuallyJobState(namespace, workerJobNames[0], "Failed").To(Succeed())

			})
		},
	}
}
