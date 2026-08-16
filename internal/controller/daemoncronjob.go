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
	"fmt"
	"maps"
	"slices"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	DaemonJobLabelDaemonCronJobName = "daemonjob.berquerant.github.io/daemoncronjob-name"
	DaemonJobEnvDaemonCronJobName   = "DAEMON_CRONJOB_NAME"
	DaemonCronJobResourceSuffix     = "dcj"
)

func NewDaemonCronJobBroadcastArgs(daemonCronJob *daemonjobv1.DaemonCronJob, clusterRoleName, image string) *DaemonCronJobBroadcastArgs {
	return &DaemonCronJobBroadcastArgs{
		DaemonCronJob:   daemonCronJob,
		ClusterRoleName: clusterRoleName,
		Image:           image,
	}
}

// DaemonCronJobBroadcastArgs generates the resources required by DaemonCronJob.
type DaemonCronJobBroadcastArgs struct {
	DaemonCronJob   *daemonjobv1.DaemonCronJob
	ClusterRoleName string
	Image           string
}

func (a DaemonCronJobBroadcastArgs) AsDaemonJob() *daemonjobv1.DaemonJob {
	d := a.DaemonCronJob.DeepCopy()
	x := new(daemonjobv1.DaemonJob)

	x.Name = d.Name
	x.Namespace = d.Namespace
	x.Labels = d.Labels
	x.Annotations = d.Annotations
	x.Spec.JobTemplate.Metadata = daemonjobv1.DaemonJobTemplateMeta{
		Labels:      d.Spec.CronJobTemplate.Spec.JobTemplate.Labels,
		Annotations: d.Spec.CronJobTemplate.Spec.JobTemplate.Annotations,
	}
	x.Spec.JobTemplate.Spec = d.Spec.CronJobTemplate.Spec.JobTemplate.Spec
	x.Spec.BroadcastJobSpec = d.Spec.BroadcastJobSpec
	x.Spec.NodeSelector = d.Spec.NodeSelector
	return x
}

func (a DaemonCronJobBroadcastArgs) daemonJobArgs() *DaemonJobBroadcastArgs {
	return NewDaemonJobBroadcastArgs(a.AsDaemonJob(), a.ClusterRoleName, a.Image)
}

func (a DaemonCronJobBroadcastArgs) fixCommonLabels(d map[string]string) {
	delete(d, DaemonJobLabelDaemonJobName)
	d[DaemonJobLabelDaemonCronJobName] = a.DaemonCronJob.Name
	d["app.kubernetes.io/instance"] = d[DaemonJobLabelDaemonCronJobName]
}

func (a DaemonCronJobBroadcastArgs) ResourceName() string {
	return a.DaemonCronJob.Name + "-" + DaemonCronJobResourceSuffix
}

func (a DaemonCronJobBroadcastArgs) JobName() string {
	return a.ResourceName()
}

func (a DaemonCronJobBroadcastArgs) CronJobName() string {
	return a.ResourceName()
}

func (a DaemonCronJobBroadcastArgs) serviceAccountName() string {
	return a.ResourceName()
}

func (a DaemonCronJobBroadcastArgs) ClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	x := a.daemonJobArgs().ClusterRoleBinding()
	a.fixCommonLabels(x.Labels)
	x.Name = a.daemonJobArgs().namespace() + "-" + a.serviceAccountName()
	x.Subjects[0].Name = a.serviceAccountName()
	return x
}

func (a DaemonCronJobBroadcastArgs) ServiceAccount() *corev1.ServiceAccount {
	x := a.daemonJobArgs().ServiceAccount()
	a.fixCommonLabels(x.Labels)
	x.Name = a.serviceAccountName()
	return x
}

func (a DaemonCronJobBroadcastArgs) Job() *batchv1.Job {
	job := a.daemonJobArgs().Job()
	job.Name = a.JobName()
	a.fixCommonLabels(job.Labels)
	a.fixCommonLabels(job.Spec.Template.Labels)
	job.Spec.Template.Spec.ServiceAccountName = a.serviceAccountName()
	job.Spec.Template.Spec.Containers[0].Env = slices.DeleteFunc(job.Spec.Template.Spec.Containers[0].Env,
		func(x corev1.EnvVar) bool {
			return x.Name == DaemonJobEnvDaemonJobName
		})
	job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name: DaemonJobEnvDaemonCronJobName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fmt.Sprintf(`metadata.labels['%s']`, DaemonJobLabelDaemonCronJobName),
			},
		},
	})
	return job
}

func (a DaemonCronJobBroadcastArgs) CronJob() *batchv1.CronJob {
	cronJob := new(batchv1.CronJob)
	dSpec := a.DaemonCronJob.Spec.CronJobTemplate.Spec.DeepCopy()
	job := a.Job()

	cronJob.Name = a.CronJobName()
	cronJob.Namespace = a.DaemonCronJob.Namespace
	cronJob.Annotations = a.DaemonCronJob.Spec.CronJobTemplate.Metadata.Annotations
	cronJob.Labels = a.DaemonCronJob.Spec.CronJobTemplate.Metadata.Labels
	maps.Copy(cronJob.Labels, job.Labels)
	cronJob.Spec.ConcurrencyPolicy = dSpec.ConcurrencyPolicy
	cronJob.Spec.FailedJobsHistoryLimit = dSpec.FailedJobsHistoryLimit
	cronJob.Spec.JobTemplate.Labels = a.DaemonCronJob.Spec.CronJobTemplate.Metadata.Labels
	if cronJob.Spec.JobTemplate.Labels == nil {
		cronJob.Spec.JobTemplate.Labels = map[string]string{}
	}
	maps.Copy(cronJob.Spec.JobTemplate.Labels, a.daemonJobArgs().commonLabels())
	a.fixCommonLabels(cronJob.Spec.JobTemplate.Labels)
	cronJob.Spec.JobTemplate.Annotations = a.DaemonCronJob.Spec.CronJobTemplate.Metadata.Annotations
	cronJob.Spec.JobTemplate.Spec = job.Spec
	cronJob.Spec.Schedule = dSpec.Schedule
	cronJob.Spec.StartingDeadlineSeconds = dSpec.StartingDeadlineSeconds
	cronJob.Spec.SuccessfulJobsHistoryLimit = dSpec.SuccessfulJobsHistoryLimit
	cronJob.Spec.Suspend = dSpec.Suspend
	cronJob.Spec.TimeZone = dSpec.TimeZone
	return cronJob
}
