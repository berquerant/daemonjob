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

// DaemonCronJobSetSpec defines the desired state of DaemonCronJobSet.
type DaemonCronJobSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// CronJobTemplate specifies the cronjob that will be created when executing a DaemonCronJob.
	// +required
	CronJobTemplate DaemonCronJobSetTemplateSpec `json:"cronJobTemplate"`
	// nodeSelector is a selector which must be true for the job to fit on a node.
	// Selector which must match a node's labels for the job to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

type DaemonCronJobSetTemplateSpec struct {
	// Metadata is a standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	Metadata DaemonCronJobSetTemplateMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of a cron job, including the schedule.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +required
	Spec batchv1.CronJobSpec `json:"spec,omitempty"`
}

type DaemonCronJobSetTemplateMeta struct {
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
	// TypeDaemonCronJobSetAvailable means that the resource is fully functional.
	TypeDaemonCronJobSetAvailable = "Available"
	// TypeDaemonCronJobSetDegraded means that the resource failed to reach or maintain its desired state.
	TypeDaemonCronJobSetDegraded = "Degraded"
	// TypeDaemonCronJobSetUnknown means that the resource is unknown state.
	TypeDaemonCronJobSetUnknown = "Unknown"
)

// DaemonCronJobSetStatus defines the observed state of DaemonCronJobSet.
type DaemonCronJobSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the DaemonCronJobSet resource.
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
	// The number of nodes that should be created the daemon cronjobs.
	// +required
	Desired int `json:"numberDesired"`
	// The number of nodes that should be created the daemon cronjobs and have one or more of the daemon cronjobs available.
	// +optional
	Available int `json:"numberAvailable"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.cronJobTemplate.spec.schedule"
// +kubebuilder:printcolumn:name="Timezone",type="string",JSONPath=".spec.cronJobTemplate.spec.timeZone"
// +kubebuilder:printcolumn:name="Suspend",type="string",JSONPath=".spec.cronJobTemplate.spec.jobTemplate.suspend"
// +kubebuilder:printcolumn:name="Containers",type="string",JSONPath=".spec.cronJobTemplate..spec.jobTemplate.spec.template.spec.containers[*].name"
// +kubebuilder:printcolumn:name="Images",type="string",JSONPath=".spec.cronJobTemplate.spec.jobTemplate.spec.template.spec.containers[*].image"
// +kubebuilder:printcolumn:name="Desired",type="string",JSONPath=".status.numberDesired"
// +kubebuilder:printcolumn:name="Available",type="string",JSONPath=".status.numberAvailable"

// DaemonCronJobSet defines a set of CronJobs designed to run across all nodes in a cluster.
// It generates one CronJob for each node based on the spec.cronJobTemplate.
// Each generated CronJob is responsible for triggering a Job that runs exclusively on its assigned node.
// These per-node Jobs are referred to as worker Jobs.
// Worker Jobs automatically apply tolerations to their Pods, equivalent to those managed by a DaemonSet,
// ensuring they can be scheduled on all target nodes.
// DaemonCronJobSet and its associated resources may be assigned the following labels for identification and tracking:
//
//   - daemonjob.berquerant.github.io/daemoncronjobset-name: The name of the originating DaemonCronJobSet.
//   - daemonjob.berquerant.github.io/node: The name of the specific node the resource is assigned to.
//   - daemonjob.berquerant.github.io/role: The role of the Job: either broadcast or worker.
//   - daemonjob.berquerant.github.io/namespace: The namespace of the originating DaemonCronJobSet.
type DaemonCronJobSet struct {
	metav1.TypeMeta `json:",inline"`

	// Metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of DaemonCronJobSet
	// +required
	Spec DaemonCronJobSetSpec `json:"spec"`

	// Status defines the observed state of DaemonCronJobSet
	// +optional
	Status DaemonCronJobSetStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// DaemonCronJobSetList contains a list of DaemonCronJobSet.
type DaemonCronJobSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []DaemonCronJobSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DaemonCronJobSet{}, &DaemonCronJobSetList{})
}
