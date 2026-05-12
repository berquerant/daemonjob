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
	"strconv"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = daemonCronJobSetControllerTest{}.run()

type daemonCronJobSetControllerTest struct{}

func (d daemonCronJobSetControllerTest) run() bool {
	return ControllerTest{
		Name:            "DaemonCronJobSet Controller",
		NamespacePrefix: "daemoncronjobset",
		Contexts: []ControllerTestContext{
			d.reconcileNormalTest(),
		},
	}.Run()
}

func (d daemonCronJobSetControllerTest) newReconciler() *DaemonCronJobSetReconciler {
	return &DaemonCronJobSetReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
}

func (daemonCronJobSetControllerTest) newNormalDaemonCronJobSet(namespace, name string) *daemonjobv1.DaemonCronJobSet {
	return &daemonjobv1.DaemonCronJobSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: daemonjobv1.DaemonCronJobSetSpec{
			CronJobTemplate: daemonjobv1.DaemonCronJobSetTemplateSpec{
				Metadata: daemonjobv1.DaemonCronJobSetTemplateMeta{
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

func (daemonCronJobSetControllerTest) nodeName(index int) string {
	return "testworker-" + strconv.Itoa(index)
}

func (d daemonCronJobSetControllerTest) addNode(nodeIndex int, labels map[string]string) error {
	return k8sClient.Create(ctx, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   d.nodeName(nodeIndex),
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Phase: corev1.NodeRunning,
		},
	})
}

func (d daemonCronJobSetControllerTest) deleteNode(nodeIndex int) error {
	return k8sClient.Delete(ctx, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: d.nodeName(nodeIndex),
		},
	})
}

func (d daemonCronJobSetControllerTest) listCronJobNodes(namespace, daemonCronJobSetName string) ([]string, error) {
	cronJobs, err := ListDaemonCronJobSetCronJobs(ctx, k8sClient, namespace, daemonCronJobSetName)
	if err != nil {
		return nil, err
	}
	nodeNames := make([]string, len(cronJobs.Items))
	for i, x := range cronJobs.Items {
		nodeNames[i] = x.Labels[DaemonJobLabelNode]
	}
	return nodeNames, nil
}
