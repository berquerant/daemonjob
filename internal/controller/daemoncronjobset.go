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
	"context"
	"maps"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	"github.com/berquerant/daemonjob/internal/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DaemonJobLabelDaemonCronJobSetName = "daemonjob.berquerant.github.io/daemoncronjobset-name"
	DaemonCronJobSetResourceSuffix     = "dcjs"
)

func NewDaemonCronJobSetArgs(daemonCronJobSet *daemonjobv1.DaemonCronJobSet) *DaemonCronJobSetArgs {
	return &DaemonCronJobSetArgs{
		DaemonCronJobSet: daemonCronJobSet,
	}
}

// DaemonCronJobSetArgs generates the resources required by DaemonCronSetJob.
type DaemonCronJobSetArgs struct {
	DaemonCronJobSet *daemonjobv1.DaemonCronJobSet
}

func (a DaemonCronJobSetArgs) namespace() string {
	return a.DaemonCronJobSet.Namespace
}

func (a DaemonCronJobSetArgs) cronJobName(nodeName string) string {
	return a.DaemonCronJobSet.Name + "-" + nodeName + "-" + DaemonCronJobSetResourceSuffix
}

func (a DaemonCronJobSetArgs) commonLabels(nodeName string) map[string]string {
	x := util.CommonLabels()
	x[DaemonJobLabelDaemonCronJobSetName] = a.DaemonCronJobSet.Name
	x[DaemonJobLabelRole] = DaemonJobLabelRoleWorker
	x[DaemonJobLabelNode] = nodeName
	x["app.kubernetes.io/instance"] = x[DaemonJobLabelDaemonCronJobSetName]
	x["app.kubernetes.io/component"] = x[DaemonJobLabelRole]
	return x
}

func (DaemonCronJobSetArgs) defaultTolerations() []corev1.Toleration {
	return []corev1.Toleration{
		{
			Effect:   corev1.TaintEffectNoExecute,
			Key:      "node.kubernetes.io/not-ready",
			Operator: corev1.TolerationOpExists,
		},
		{
			Effect:   corev1.TaintEffectNoExecute,
			Key:      "node.kubernetes.io/unreachable",
			Operator: corev1.TolerationOpExists,
		},
		{
			Effect:   corev1.TaintEffectNoSchedule,
			Key:      "node.kubernetes.io/disk-pressure",
			Operator: corev1.TolerationOpExists,
		},
		{
			Effect:   corev1.TaintEffectNoSchedule,
			Key:      "node.kubernetes.io/memory-pressure",
			Operator: corev1.TolerationOpExists,
		},
		{
			Effect:   corev1.TaintEffectNoSchedule,
			Key:      "node.kubernetes.io/pid-pressure",
			Operator: corev1.TolerationOpExists,
		},
		{
			Effect:   corev1.TaintEffectNoSchedule,
			Key:      "node.kubernetes.io/unschedulable",
			Operator: corev1.TolerationOpExists,
		},
	}
}

func (a DaemonCronJobSetArgs) ListNodes(ctx context.Context, c client.Client) ([]string, error) {
	nodes, err := List[*corev1.NodeList](ctx, c, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(a.DaemonCronJobSet.Spec.NodeSelector),
	})
	if err != nil {
		return nil, err
	}
	names := make([]string, len(nodes.Items))
	for i, x := range nodes.Items {
		names[i] = x.Name
	}
	return names, nil
}

func (a DaemonCronJobSetArgs) NewCronJobForNode(nodeName string) *batchv1.CronJob {
	var (
		cronJob = new(batchv1.CronJob)
		d       = a.DaemonCronJobSet.DeepCopy()
		dSpec   = d.Spec.CronJobTemplate
	)
	//
	// metadata
	//
	cronJob.Name = a.cronJobName(nodeName)
	cronJob.Namespace = a.namespace()
	cronJob.Annotations = dSpec.Metadata.Annotations
	cronJob.Labels = dSpec.Metadata.Labels
	if cronJob.Labels == nil {
		cronJob.Labels = map[string]string{}
	}
	maps.Copy(cronJob.Labels, a.commonLabels(nodeName))
	//
	// spec
	//
	cronJob.Spec = dSpec.Spec
	//
	// spec.jobTemplate
	//
	if cronJob.Spec.JobTemplate.Labels == nil {
		cronJob.Spec.JobTemplate.Labels = map[string]string{}
	}
	maps.Copy(cronJob.Spec.JobTemplate.Labels, a.commonLabels(nodeName))
	//
	// spec.jobTemplate.spec.template
	//
	if cronJob.Spec.JobTemplate.Spec.Template.Labels == nil {
		cronJob.Spec.JobTemplate.Spec.Template.Labels = map[string]string{}
	}
	maps.Copy(cronJob.Spec.JobTemplate.Spec.Template.Labels, a.commonLabels(nodeName))
	//
	// spec.jobTemplate.spec.template.spec.affinity
	//
	affinity := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchFields: []corev1.NodeSelectorRequirement{
							{
								Key:      "metadata.name",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{nodeName},
							},
						},
					},
				},
			},
		},
	}
	cronJob.Spec.JobTemplate.Spec.Template.Spec.Affinity = affinity
	//
	// spec.jobTemplate.spec.template.spec.tolerations
	//
	cronJob.Spec.JobTemplate.Spec.Template.Spec.Tolerations = append(
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Tolerations,
		a.defaultTolerations()...,
	)
	return cronJob
}

func (a DaemonCronJobSetArgs) CronJobs(ctx context.Context, c client.Client) ([]*batchv1.CronJob, error) {
	nodeNames, err := a.ListNodes(ctx, c)
	if err != nil {
		return nil, err
	}
	cronJobs := make([]*batchv1.CronJob, len(nodeNames))
	for i, nodeName := range nodeNames {
		cronJobs[i] = a.NewCronJobForNode(nodeName)
	}
	return cronJobs, nil
}
