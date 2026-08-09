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

package broadcast

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	"github.com/berquerant/daemonjob/internal/controller"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var log = ctrl.Log.WithName("broadcast")

// Config holds the configuration populated from environment variables for the broadcast job.
type Config struct {
	SelfName          string
	Namespace         string
	DaemonJobName     string
	DaemonCronJobName string
	ControllerUID     types.UID
}

// LoadConfigFromEnv loads configuration options from standard environment variables.
func LoadConfigFromEnv() (*Config, error) {
	cfg := &Config{
		SelfName:          os.Getenv(controller.DaemonJobEnvSelfName),
		Namespace:         os.Getenv(controller.DaemonJobEnvNamespace),
		DaemonJobName:     os.Getenv(controller.DaemonJobEnvDaemonJobName),
		DaemonCronJobName: os.Getenv(controller.DaemonJobEnvDaemonCronJobName),
		ControllerUID:     types.UID(os.Getenv(controller.DaemonJobEnvControllerUid)),
	}

	if cfg.Namespace == "" {
		return nil, fmt.Errorf("missing required environment variable: %s", controller.DaemonJobEnvNamespace)
	}
	if cfg.SelfName == "" {
		return nil, fmt.Errorf("missing required environment variable: %s", controller.DaemonJobEnvSelfName)
	}
	if cfg.DaemonJobName == "" && cfg.DaemonCronJobName == "" {
		return nil, fmt.Errorf("either %s or %s must be specified", controller.DaemonJobEnvDaemonJobName, controller.DaemonJobEnvDaemonCronJobName)
	}

	return cfg, nil
}

// Runner handles finding nodes and building/applying worker jobs.
type Runner struct {
	Client client.Client
	Config *Config
}

// NewRunner initializes a new Runner instance with Kubernetes client.
func NewRunner(cfg *Config) (*Runner, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(daemonjobv1.AddToScheme(scheme))

	kubeConfig, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kube config: %w", err)
	}

	k8sClient, err := client.New(kubeConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &Runner{
		Client: k8sClient,
		Config: cfg,
	}, nil
}

// Run executes the broadcast workflow.
func (r *Runner) Run(ctx context.Context) error {
	var (
		nodeSelector map[string]string
		jobSpec      daemonjobv1.DaemonJobSpec
	)

	if r.Config.DaemonJobName != "" {
		var daemonJob daemonjobv1.DaemonJob
		key := client.ObjectKey{Namespace: r.Config.Namespace, Name: r.Config.DaemonJobName}
		if err := r.Client.Get(ctx, key, &daemonJob); err != nil {
			return fmt.Errorf("failed to get DaemonJob %s/%s: %w", r.Config.Namespace, r.Config.DaemonJobName, err)
		}
		nodeSelector = daemonJob.Spec.NodeSelector
		jobSpec = daemonJob.Spec
	} else {
		var daemonCronJob daemonjobv1.DaemonCronJob
		key := client.ObjectKey{Namespace: r.Config.Namespace, Name: r.Config.DaemonCronJobName}
		if err := r.Client.Get(ctx, key, &daemonCronJob); err != nil {
			return fmt.Errorf("failed to get DaemonCronJob %s/%s: %w", r.Config.Namespace, r.Config.DaemonCronJobName, err)
		}
		nodeSelector = daemonCronJob.Spec.NodeSelector
		jobSpec = daemonjobv1.DaemonJobSpec{
			JobTemplate: daemonjobv1.DaemonJobTemplateSpec{
				Metadata: daemonjobv1.DaemonJobTemplateMeta{
					Labels:      daemonCronJob.Spec.CronJobTemplate.Spec.JobTemplate.Labels,
					Annotations: daemonCronJob.Spec.CronJobTemplate.Spec.JobTemplate.Annotations,
				},
				Spec: daemonCronJob.Spec.CronJobTemplate.Spec.JobTemplate.Spec,
			},
			BroadcastJobSpec: daemonCronJob.Spec.BroadcastJobSpec,
			NodeSelector:     daemonCronJob.Spec.NodeSelector,
		}
	}

	// List nodes with matching node selector
	var nodeList corev1.NodeList
	listOpts := []client.ListOption{}
	if len(nodeSelector) > 0 {
		listOpts = append(listOpts, client.MatchingLabels(nodeSelector))
	}

	if err := r.Client.List(ctx, &nodeList, listOpts...); err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodeList.Items) == 0 {
		log.Info("No nodes found to process")
		return nil
	}

	workerJobs := make([]*batchv1.Job, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		workerJob := r.BuildWorkerJob(node.Name, &jobSpec)
		workerJobs = append(workerJobs, workerJob)
	}

	// Validate worker jobs with Server-Side Dry-Run
	log.Info("Validating worker jobs manifest with server-side dry-run", "count", len(workerJobs))
	for _, job := range workerJobs {
		dryRunJob := job.DeepCopy()
		if err := r.Client.Create(ctx, dryRunJob, client.DryRunAll); err != nil {
			return fmt.Errorf("dry-run validation failed for worker job %s: %w", job.Name, err)
		}
	}

	// Apply worker jobs
	log.Info("Applying worker jobs for node(s) atomically", "count", len(workerJobs))
	for _, job := range workerJobs {
		if err := r.Client.Create(ctx, job); err != nil {
			return fmt.Errorf("failed to create worker job %s: %w", job.Name, err)
		}
	}

	log.Info("Applied worker jobs for all nodes successfully")
	return nil
}

// BuildWorkerJob constructs a worker batchv1.Job for a specific target node.
func (r *Runner) BuildWorkerJob(nodeName string, daemonJobSpec *daemonjobv1.DaemonJobSpec) *batchv1.Job {
	jobName := fmt.Sprintf("%s-%s", r.Config.SelfName, nodeName)

	dJobTmpl := daemonJobSpec.JobTemplate
	workerJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   r.Config.Namespace,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         batchv1.SchemeGroupVersion.String(),
					Kind:               "Job",
					Name:               r.Config.SelfName,
					UID:                r.Config.ControllerUID,
					Controller:         new(true),
					BlockOwnerDeletion: new(true),
				},
			},
		},
	}

	// Copy template labels & annotations
	maps.Copy(workerJob.Labels, dJobTmpl.Metadata.Labels)
	maps.Copy(workerJob.Annotations, dJobTmpl.Metadata.Annotations)

	// Standard labels
	workerJob.Labels[controller.DaemonJobLabelNode] = nodeName
	workerJob.Labels[controller.DaemonJobLabelRole] = controller.DaemonJobLabelRoleWorker
	if r.Config.DaemonJobName != "" {
		workerJob.Labels[controller.DaemonJobLabelDaemonJobName] = r.Config.DaemonJobName
	} else {
		workerJob.Labels[controller.DaemonJobLabelDaemonCronJobName] = r.Config.DaemonCronJobName
	}

	// Remove k8s auto-generated job labels
	for _, l := range []string{"controller-uid", "job-name", "batch.kubernetes.io/controller-uid", "batch.kubernetes.io/job-name"} {
		delete(workerJob.Labels, l)
	}

	// Job Spec & Pod Template
	workerJob.Spec = *dJobTmpl.Spec.DeepCopy()
	workerJob.Spec.Selector = nil // Let k8s Job controller auto-generate selector

	// Ensure Template Metadata labels/annotations match Job ObjectMeta
	workerJob.Spec.Template.Labels = make(map[string]string)
	if dJobTmpl.Spec.Template.Labels != nil {
		maps.Copy(workerJob.Spec.Template.Labels, dJobTmpl.Spec.Template.Labels)
	}
	maps.Copy(workerJob.Spec.Template.Labels, workerJob.Labels)
	for _, l := range []string{"controller-uid", "job-name", "batch.kubernetes.io/controller-uid", "batch.kubernetes.io/job-name"} {
		delete(workerJob.Spec.Template.Labels, l)
	}

	workerJob.Spec.Template.Annotations = make(map[string]string)
	if dJobTmpl.Spec.Template.Annotations != nil {
		maps.Copy(workerJob.Spec.Template.Annotations, dJobTmpl.Spec.Template.Annotations)
	}
	maps.Copy(workerJob.Spec.Template.Annotations, workerJob.Annotations)

	podSpec := &workerJob.Spec.Template.Spec

	// NodeAffinity to target node
	if podSpec.Affinity == nil {
		podSpec.Affinity = &corev1.Affinity{}
	}
	if podSpec.Affinity.NodeAffinity == nil {
		podSpec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}

	podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
		podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
		corev1.NodeSelectorTerm{
			MatchFields: []corev1.NodeSelectorRequirement{
				{
					Key:      metav1.ObjectNameField,
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{nodeName},
				},
			},
		},
	)

	// Node Tolerations (Default + Base Job Tolerations)
	defaultTolerations := make([]corev1.Toleration, 0, 6+len(podSpec.Tolerations))
	defaultTolerations = append(defaultTolerations,
		corev1.Toleration{
			Key:      "node.kubernetes.io/not-ready",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoExecute,
		},
		corev1.Toleration{
			Key:      "node.kubernetes.io/unreachable",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoExecute,
		},
		corev1.Toleration{
			Key:      "node.kubernetes.io/disk-pressure",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
		corev1.Toleration{
			Key:      "node.kubernetes.io/memory-pressure",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
		corev1.Toleration{
			Key:      "node.kubernetes.io/pid-pressure",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
		corev1.Toleration{
			Key:      "node.kubernetes.io/unschedulable",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	)

	podSpec.Tolerations = append(defaultTolerations, podSpec.Tolerations...)

	return workerJob
}

// WriteTerminationLog writes a failure message to /dev/termination-log if available, or logs via controller-runtime logger.
func WriteTerminationLog(msg string) {
	log.Error(fmt.Errorf("%s", msg), "Broadcast execution failed")
	f, err := os.OpenFile("/dev/termination-log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err == nil {
		defer func() {
			_ = f.Close()
		}()
		_, _ = fmt.Fprintln(f, strings.TrimSpace(msg))
	}
}
