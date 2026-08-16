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

package daemonjobctl

import (
	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DaemonJobSkeleton returns a minimal but complete skeleton DaemonJob manifest.
func DaemonJobSkeleton() *daemonjobv1.DaemonJob {
	backoffLimit := int32(0)
	return &daemonjobv1.DaemonJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: daemonjobv1.GroupVersion.String(),
			Kind:       KindDaemonJob,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-daemonjob",
			Namespace: DefaultNamespace,
		},
		Spec: daemonjobv1.DaemonJobSpec{
			NodeSelector: map[string]string{},
			JobTemplate: daemonjobv1.DaemonJobTemplateSpec{
				Metadata: daemonjobv1.DaemonJobTemplateMeta{
					Labels:      map[string]string{DefaultLabelName: "my-daemonjob"},
					Annotations: map[string]string{},
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: &backoffLimit,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:    DefaultContainer,
									Image:   DefaultImage,
									Command: []string{DefaultShell, "-c", DefaultCommand},
								},
							},
						},
					},
				},
			},
		},
	}
}

// DaemonCronJobSkeleton returns a minimal but complete skeleton DaemonCronJob manifest.
func DaemonCronJobSkeleton() *daemonjobv1.DaemonCronJob {
	backoffLimit := int32(0)
	return &daemonjobv1.DaemonCronJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: daemonjobv1.GroupVersion.String(),
			Kind:       KindDaemonCronJob,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-daemoncronjob",
			Namespace: DefaultNamespace,
		},
		Spec: daemonjobv1.DaemonCronJobSpec{
			NodeSelector: map[string]string{},
			CronJobTemplate: daemonjobv1.DaemonCronJobTemplateSpec{
				Metadata: daemonjobv1.DaemonCronJobTemplateMeta{
					Labels:      map[string]string{DefaultLabelName: "my-daemoncronjob"},
					Annotations: map[string]string{},
				},
				Spec: batchv1.CronJobSpec{
					Schedule: "*/5 * * * *",
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							BackoffLimit: &backoffLimit,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									RestartPolicy: corev1.RestartPolicyNever,
									Containers: []corev1.Container{
										{
											Name:    DefaultContainer,
											Image:   DefaultImage,
											Command: []string{DefaultShell, "-c", DefaultCommand},
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

// DaemonCronJobSetSkeleton returns a minimal but complete skeleton DaemonCronJobSet manifest.
func DaemonCronJobSetSkeleton() *daemonjobv1.DaemonCronJobSet {
	backoffLimit := int32(0)
	return &daemonjobv1.DaemonCronJobSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: daemonjobv1.GroupVersion.String(),
			Kind:       KindDaemonCronJobSet,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-daemoncronjobset",
			Namespace: DefaultNamespace,
		},
		Spec: daemonjobv1.DaemonCronJobSetSpec{
			NodeSelector: map[string]string{},
			CronJobTemplate: daemonjobv1.DaemonCronJobSetTemplateSpec{
				Metadata: daemonjobv1.DaemonCronJobSetTemplateMeta{
					Labels:      map[string]string{DefaultLabelName: "my-daemoncronjobset"},
					Annotations: map[string]string{},
				},
				Spec: batchv1.CronJobSpec{
					Schedule: "*/5 * * * *",
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							BackoffLimit: &backoffLimit,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									RestartPolicy: corev1.RestartPolicyNever,
									Containers: []corev1.Container{
										{
											Name:    DefaultContainer,
											Image:   DefaultImage,
											Command: []string{DefaultShell, "-c", DefaultCommand},
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
