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

package v1

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DaemonCronJobSpec defines the desired state of DaemonCronJob
type DaemonCronJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// CronJobTemplate specifies the cronjob that will be created when executing a DaemonCronJob.
	// +required
	CronJobTemplate DaemonCronJobTemplateSpec `json:"cronJobTemplate"`
	// nodeSelector is a selector which must be true for the job to fit on a node.
	// Selector which must match a node's labels for the job to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// BroadcastJobSpec is a spec of the broadcast job.
	// +optional
	BroadcastJobSpec DaemonJobBroadcastJobSpec `json:"broadcastJobSpec,omitempty"`
}

type DaemonCronJobTemplateSpec struct {
	// Metadata is a standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	Metadata DaemonCronJobTemplateMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of a cron job, including the schedule.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +required
	Spec batchv1.CronJobSpec `json:"spec,omitempty"`
}

type DaemonCronJobTemplateMeta struct {
	// Labels is a map of string keys and values that can be used to organize and categorize (scope and select) objects.
	// May match selectors of replication controllers and services.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations is an unstructured key value map stored with a resource that may be set by
	// external tools to store and retrieve arbitrary metadata.
	// They are not queryable and should be preserved when modifying objects.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

const (
	// TypeDaemonCronJobAvailable means that the resource is fully functional.
	TypeDaemonCronJobAvailable = "Available"
	// TypeDaemonCronJobDegraded means that the resource failed to reach or maintain its desired state.
	TypeDaemonCronJobDegraded = "Degraded"
	// TypeDaemonCronJobUnknown means that the resource is unknown state.
	TypeDaemonCronJobUnknown = "Unknown"
)

// DaemonCronJobStatus defines the observed state of DaemonCronJob.
type DaemonCronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// Conditions represent the current state of the DaemonCronJob resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.cronJobTemplate.spec.schedule"
// +kubebuilder:printcolumn:name="Timezone",type="string",JSONPath=".spec.cronJobTemplate.spec.timeZone"
// +kubebuilder:printcolumn:name="Suspend",type="string",JSONPath=".spec.cronJobTemplate.spec.jobTemplate.suspend"
// +kubebuilder:printcolumn:name="Containers",type="string",JSONPath=".spec.cronJobTemplate..spec.jobTemplate.spec.template.spec.containers[*].name"
// +kubebuilder:printcolumn:name="Images",type="string",JSONPath=".spec.cronJobTemplate.spec.jobTemplate.spec.template.spec.containers[*].image"

// DaemonCronJob defines a task that runs periodically on every node in the cluster.
// It functions by creating a CronJob, which is responsible for triggering a DaemonJob's broadcast Job at scheduled intervals.
// DaemonCronJob and its associated resources may be assigned the following labels for identification and tracking:
//
//   - daemonjob.berquerant.github.io/daemoncronjob-name: The name of the originating DaemonCronJob.
//   - daemonjob.berquerant.github.io/node: The name of the specific node the resource is assigned to.
//   - daemonjob.berquerant.github.io/role: The role of the Job: either broadcast or worker.
//   - daemonjob.berquerant.github.io/namespace: The namespace of the originating DaemonCronJob.
type DaemonCronJob struct {
	metav1.TypeMeta `json:",inline"`

	// Metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of DaemonCronJob
	// +required
	Spec DaemonCronJobSpec `json:"spec"`

	// Status defines the observed state of DaemonCronJob
	// +optional
	Status DaemonCronJobStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// DaemonCronJobList contains a list of DaemonCronJob
type DaemonCronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []DaemonCronJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DaemonCronJob{}, &DaemonCronJobList{})
}
