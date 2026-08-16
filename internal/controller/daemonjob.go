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

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	"github.com/berquerant/daemonjob/internal/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	DaemonJobLabelRole = "daemonjob.berquerant.github.io/role"
	// The worker job is bound to a specific node.
	DaemonJobLabelRoleWorker = "worker"
	// The broadcast job creates the worker jobs.
	DaemonJobLabelRoleBroadcast = "broadcast"
	DaemonJobLabelDaemonJobName = "daemonjob.berquerant.github.io/daemonjob-name"
	DaemonJobLabelNode          = "daemonjob.berquerant.github.io/node"
	// Retain the namespace that the cluster-scoped resource depends on.
	DaemonJobLabelNamespace         = "daemonjob.berquerant.github.io/namespace"
	DaemonJobEnvSelfName            = "SELF_NAME"
	DaemonJobEnvNamespace           = "NAMESPACE"
	DaemonJobEnvDaemonJobName       = "DAEMON_JOB_NAME"
	DaemonJobEnvControllerUid       = "CONTROLLER_UID"
	DaemonJobBroadcastContainerName = "broadcast"
	DaemonJobResourceSuffix         = "dj"
)

func NewDaemonJobBroadcastArgs(daemonJob *daemonjobv1.DaemonJob, clusterRoleName, image string) *DaemonJobBroadcastArgs {
	return &DaemonJobBroadcastArgs{
		DaemonJob:       daemonJob,
		ClusterRoleName: clusterRoleName,
		Image:           image,
	}
}

// DaemonJobBroadcastArgs generates the resources required by DaemonJob.
type DaemonJobBroadcastArgs struct {
	DaemonJob       *daemonjobv1.DaemonJob
	ClusterRoleName string
	Image           string
}

func (a DaemonJobBroadcastArgs) ResourceName() string {
	return a.DaemonJob.Name + "-" + DaemonJobResourceSuffix
}

func (a DaemonJobBroadcastArgs) JobName() string {
	return a.ResourceName()
}

func (a DaemonJobBroadcastArgs) serviceAccountName() string {
	return a.ResourceName()
}

func (a DaemonJobBroadcastArgs) namespace() string {
	return a.DaemonJob.Namespace
}

func (a DaemonJobBroadcastArgs) commonLabels() map[string]string {
	x := util.CommonLabels()
	x[DaemonJobLabelDaemonJobName] = a.DaemonJob.Name
	x[DaemonJobLabelRole] = DaemonJobLabelRoleBroadcast
	x["app.kubernetes.io/instance"] = x[DaemonJobLabelDaemonJobName]
	x["app.kubernetes.io/component"] = x[DaemonJobLabelRole]
	return x
}

// ClusterRoleBinding allows a broadcast job to take the necessary actions.
// Like:
//   - List Nodes
//   - Create Jobs
//   - Get DaemonJobs
func (a DaemonJobBroadcastArgs) ClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	crb := new(rbacv1.ClusterRoleBinding)
	crb.Name = a.namespace() + "-" + a.serviceAccountName()
	crb.Labels = a.commonLabels()
	crb.Labels[DaemonJobLabelNamespace] = a.namespace() // Keep namespace.
	crb.Subjects = []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      a.serviceAccountName(),
			Namespace: a.namespace(),
		},
	}
	crb.RoleRef = rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     a.ClusterRoleName,
		APIGroup: rbacv1.GroupName,
	}
	return crb
}

func (a DaemonJobBroadcastArgs) ServiceAccount() *corev1.ServiceAccount {
	sa := new(corev1.ServiceAccount)
	sa.Name = a.serviceAccountName()
	sa.Namespace = a.namespace()
	sa.Labels = a.commonLabels()
	return sa
}

func (a DaemonJobBroadcastArgs) Job() *batchv1.Job {
	var (
		job      = new(batchv1.Job)
		dJob     = a.DaemonJob.DeepCopy()
		dJobTmpl = dJob.Spec.JobTemplate
		bJobSpec = dJob.Spec.BroadcastJobSpec
	)
	//
	// metadata
	//
	meta := &job.ObjectMeta
	meta.Name = a.JobName()
	meta.Namespace = a.namespace()
	meta.Annotations = dJobTmpl.Metadata.Annotations
	meta.Labels = dJobTmpl.Metadata.Labels
	if meta.Labels == nil {
		meta.Labels = map[string]string{}
	}
	maps.Copy(meta.Labels, a.commonLabels())
	//
	// spec
	//
	spec := &job.Spec
	spec.BackoffLimit = new(int32(0))
	//
	// spec.template
	//
	tmpl := &spec.Template
	tmpl.Annotations = meta.Annotations
	tmpl.Labels = meta.Labels
	tmpl.Spec.RestartPolicy = corev1.RestartPolicyNever
	tmpl.Spec.ServiceAccountName = a.serviceAccountName()
	tmpl.Spec.SecurityContext = new(corev1.PodSecurityContext)
	tmpl.Spec.SecurityContext.RunAsNonRoot = new(true)
	tmpl.Spec.SecurityContext.SeccompProfile = new(corev1.SeccompProfile)
	tmpl.Spec.SecurityContext.SeccompProfile.Type = corev1.SeccompProfileTypeRuntimeDefault
	tmpl.Spec.Affinity = bJobSpec.Affinity
	tmpl.Spec.ImagePullSecrets = bJobSpec.ImagePullSecrets
	tmpl.Spec.NodeName = bJobSpec.NodeName
	tmpl.Spec.NodeSelector = bJobSpec.NodeSelector
	tmpl.Spec.PreemptionPolicy = bJobSpec.PreemptionPolicy
	tmpl.Spec.Priority = bJobSpec.Priority
	tmpl.Spec.PriorityClassName = bJobSpec.PriorityClassName
	tmpl.Spec.SchedulerName = bJobSpec.SchedulerName
	tmpl.Spec.Tolerations = bJobSpec.Tolerations
	tmpl.Spec.TopologySpreadConstraints = bJobSpec.TopologySpreadConstraints
	//
	// spec.template.volumes
	//
	var tmpVolume corev1.Volume
	tmpVolume.Name = "tmp"
	tmpVolume.EmptyDir = new(corev1.EmptyDirVolumeSource)
	tmpl.Spec.Volumes = []corev1.Volume{tmpVolume}
	//
	// spec.template.spec.containers
	//
	var container corev1.Container
	var tmpVolumeMount corev1.VolumeMount
	tmpVolumeMount.Name = tmpVolume.Name
	tmpVolumeMount.MountPath = "/tmp"
	container.VolumeMounts = []corev1.VolumeMount{tmpVolumeMount}
	container.Name = DaemonJobBroadcastContainerName
	container.SecurityContext = new(corev1.SecurityContext)
	container.SecurityContext.AllowPrivilegeEscalation = new(false)
	container.SecurityContext.ReadOnlyRootFilesystem = new(true)
	container.SecurityContext.Capabilities = new(corev1.Capabilities)
	container.SecurityContext.Capabilities.Drop = []corev1.Capability{"ALL"}
	container.Image = a.Image
	container.Resources = a.DaemonJob.Spec.BroadcastJobSpec.Resources
	container.Env = []corev1.EnvVar{
		{
			Name: DaemonJobEnvSelfName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.labels['batch.kubernetes.io/job-name']",
				},
			},
		},
		{
			Name: DaemonJobEnvNamespace,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: DaemonJobEnvControllerUid,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.labels['batch.kubernetes.io/controller-uid']",
				},
			},
		},
		{
			Name: DaemonJobEnvDaemonJobName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: fmt.Sprintf(`metadata.labels['%s']`, DaemonJobLabelDaemonJobName),
				},
			},
		},
	}
	tmpl.Spec.Containers = []corev1.Container{container}

	return job
}
