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

package broadcast_test

import (
	"testing"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	"github.com/berquerant/daemonjob/internal/broadcast"
	"github.com/berquerant/daemonjob/internal/controller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Run("valid env for daemonjob", func(t *testing.T) {
		t.Setenv(controller.DaemonJobEnvSelfName, "broadcast-1")
		t.Setenv(controller.DaemonJobEnvNamespace, "test-ns")
		t.Setenv(controller.DaemonJobEnvDaemonJobName, "dj-sample")
		t.Setenv(controller.DaemonJobEnvControllerUid, "uid-111")
		t.Setenv(controller.DaemonJobEnvDaemonCronJobName, "")

		cfg, err := broadcast.LoadConfigFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "broadcast-1", cfg.SelfName)
		assert.Equal(t, "test-ns", cfg.Namespace)
		assert.Equal(t, "dj-sample", cfg.DaemonJobName)
		assert.Empty(t, cfg.DaemonCronJobName)
		assert.Equal(t, types.UID("uid-111"), cfg.ControllerUID)
	})

	t.Run("valid env for daemoncronjob", func(t *testing.T) {
		t.Setenv(controller.DaemonJobEnvSelfName, "broadcast-2")
		t.Setenv(controller.DaemonJobEnvNamespace, "test-ns")
		t.Setenv(controller.DaemonJobEnvDaemonJobName, "")
		t.Setenv(controller.DaemonJobEnvDaemonCronJobName, "dcj-sample")
		t.Setenv(controller.DaemonJobEnvControllerUid, "uid-222")

		cfg, err := broadcast.LoadConfigFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "dcj-sample", cfg.DaemonCronJobName)
		assert.Empty(t, cfg.DaemonJobName)
	})

	t.Run("missing namespace", func(t *testing.T) {
		t.Setenv(controller.DaemonJobEnvSelfName, "broadcast-1")
		t.Setenv(controller.DaemonJobEnvNamespace, "")
		t.Setenv(controller.DaemonJobEnvDaemonJobName, "dj-sample")

		_, err := broadcast.LoadConfigFromEnv()
		assert.Error(t, err)
	})

	t.Run("missing target job and cronjob names", func(t *testing.T) {
		t.Setenv(controller.DaemonJobEnvSelfName, "broadcast-1")
		t.Setenv(controller.DaemonJobEnvNamespace, "test-ns")
		t.Setenv(controller.DaemonJobEnvDaemonJobName, "")
		t.Setenv(controller.DaemonJobEnvDaemonCronJobName, "")

		_, err := broadcast.LoadConfigFromEnv()
		assert.Error(t, err)
	})
}

func TestBuildWorkerJob(t *testing.T) {
	t.Run("DaemonJob worker build with label cleaning", func(t *testing.T) {
		cfg := &broadcast.Config{
			SelfName:      "broadcast-job-1",
			Namespace:     "default",
			DaemonJobName: "my-daemonjob",
			ControllerUID: types.UID("uid-1234"),
		}

		runner := &broadcast.Runner{
			Config: cfg,
		}

		spec := &daemonjobv1.DaemonJobSpec{
			JobTemplate: daemonjobv1.DaemonJobTemplateSpec{
				Metadata: daemonjobv1.DaemonJobTemplateMeta{
					Labels: map[string]string{
						"app":                                "test",
						"controller-uid":                     "parent-uid",
						"job-name":                           "parent-job",
						"batch.kubernetes.io/controller-uid": "parent-uid",
						"batch.kubernetes.io/job-name":       "parent-job",
					},
					Annotations: map[string]string{
						"note": "sample",
					},
				},
				Spec: batchv1.JobSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"controller-uid": "parent-uid",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "worker",
									Image: "alpine",
								},
							},
							Tolerations: []corev1.Toleration{
								{
									Key:      "custom-key",
									Operator: corev1.TolerationOpExists,
								},
							},
						},
					},
				},
			},
		}

		job := runner.BuildWorkerJob("node-1", spec)

		assert.Equal(t, "broadcast-job-1-node-1", job.Name)
		assert.Equal(t, "default", job.Namespace)
		assert.Nil(t, job.Spec.Selector) // Selector must be nil for k8s job controller
		assert.Equal(t, "node-1", job.Labels[controller.DaemonJobLabelNode])
		assert.Equal(t, controller.DaemonJobLabelRoleWorker, job.Labels[controller.DaemonJobLabelRole])
		assert.Equal(t, "my-daemonjob", job.Labels[controller.DaemonJobLabelDaemonJobName])
		assert.Equal(t, "test", job.Labels["app"])
		assert.Equal(t, "sample", job.Annotations["note"])

		// Ensure auto-generated job labels are removed from ObjectMeta and PodTemplate
		for _, l := range []string{"controller-uid", "job-name", "batch.kubernetes.io/controller-uid", "batch.kubernetes.io/job-name"} {
			assert.NotContains(t, job.Labels, l)
			assert.NotContains(t, job.Spec.Template.Labels, l)
		}

		// Check OwnerReferences
		assert.Len(t, job.OwnerReferences, 1)
		assert.Equal(t, "broadcast-job-1", job.OwnerReferences[0].Name)
		assert.Equal(t, types.UID("uid-1234"), job.OwnerReferences[0].UID)

		// Check NodeAffinity
		affinity := job.Spec.Template.Spec.Affinity
		require.NotNil(t, affinity)
		require.NotNil(t, affinity.NodeAffinity)
		terms := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
		require.Len(t, terms, 1)
		assert.Equal(t, metav1.ObjectNameField, terms[0].MatchFields[0].Key)
		assert.Equal(t, []string{"node-1"}, terms[0].MatchFields[0].Values)

		// Check Tolerations (6 default + 1 custom)
		tolerations := job.Spec.Template.Spec.Tolerations
		assert.Len(t, tolerations, 7)
		assert.Equal(t, "custom-key", tolerations[6].Key)
	})

	t.Run("DaemonCronJob worker build", func(t *testing.T) {
		cfg := &broadcast.Config{
			SelfName:          "broadcast-cron-1",
			Namespace:         "prod",
			DaemonCronJobName: "my-cronjob",
			ControllerUID:     types.UID("uid-5678"),
		}

		runner := &broadcast.Runner{
			Config: cfg,
		}

		spec := &daemonjobv1.DaemonJobSpec{
			JobTemplate: daemonjobv1.DaemonJobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "cron-worker",
									Image: "busybox",
								},
							},
						},
					},
				},
			},
		}

		job := runner.BuildWorkerJob("node-2", spec)

		assert.Equal(t, "broadcast-cron-1-node-2", job.Name)
		assert.Equal(t, "prod", job.Namespace)
		assert.Equal(t, "node-2", job.Labels[controller.DaemonJobLabelNode])
		assert.Equal(t, controller.DaemonJobLabelRoleWorker, job.Labels[controller.DaemonJobLabelRole])
		assert.Equal(t, "my-cronjob", job.Labels[controller.DaemonJobLabelDaemonCronJobName])
		assert.Empty(t, job.Labels[controller.DaemonJobLabelDaemonJobName])
	})
}
