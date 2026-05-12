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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DaemonJobSpec defines the desired state of DaemonJob.
type DaemonJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// JobTemplate specifies the job that will be created when executing a DaemonJob.
	// +required
	JobTemplate DaemonJobTemplateSpec `json:"jobTemplate"`
	// NodeSelector is a selector which must be true for the job to fit on a node.
	// Selector which must match a node's labels for the job to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// BroadcastJobSpec is a spec of the broadcast job.
	// +optional
	BroadcastJobSpec DaemonJobBroadcastJobSpec `json:"broadcastJobSpec,omitempty"`
}

// DaemonJobBroadcastJobSpec defines the spec of the broadcast Job.
type DaemonJobBroadcastJobSpec struct {
	// If specified, the pod's scheduling constraints
	// Affinity is a group of affinity scheduling rules.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same
	// namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imgePullSecrets,omitempty"`
	// NodeName indicates in which node this pod is scheduled.
	// If empty, this pod is a candidate for scheduling by the scheduler defined in schedulerName.
	// Once this field is set, the kubelet for this node becomes responsible for the lifecycle of this pod.
	// This field should not be used to express a desire  for the pod to be scheduled on a specific node.
	// https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodename
	// +optional
	NodeName string `json:"nodeName,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +mapType=atomic
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// PreemptionPolicy is the Policy for preempting pods with lower priority.
	// One of Never, PreemptLowerPriority.
	// Defaults to PreemptLowerPriority if unset.
	//
	// Possible enum values:
	//  - `"Never"` means that pod never preempts other pods with lower priority.
	//  - `"PreemptLowerPriority"` means that pod can preempt other pods with lower priority.
	PreemptionPolicy *corev1.PreemptionPolicy `json:"preemptionPolicy,omitempty"`
	// The priority value.
	// Various system components use this field to find the priority of the pod.
	// When Priority Admission Controller is enabled, it prevents users from setting this field.
	// The admission controller populates this field from PriorityClassName.
	// The higher the value, the higher the priority.
	// +optional
	Priority *int32 `json:"priority,omitempty"`
	// If specified, indicates the pod's priority.
	// "system-node-critical" and "system-cluster-critical" are two special keywords
	// which indicate the highest priorities with the former being the highest priority.
	// Any other name must be defined by creating a PriorityClass object with that name.
	// If not specified, the pod priority will be default or zero if there is no default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// Resources specifies the compute resources required by the container.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// If specified, the pod will be dispatched by specified scheduler.
	// If not specified, the pod will be dispatched by default scheduler.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty"`
	// If specified, the pod's tolerations.
	// The pod this Toleration is attached to tolerates any taint that matches
	// the triple <key,value,effect> using the matching operator <operator>.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// TopologySpreadConstraints describes how a group of pods ought to spread across topology domains.
	// Scheduler will schedule pods in a way which abides by the constraints.
	// All topologySpreadConstraints are ANDed.
	// TopologySpreadConstraint specifies how to spread matching pods among the given topology.
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

type DaemonJobTemplateSpec struct {
	// Metadata is a standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	Metadata DaemonJobTemplateMeta `json:"metadata,omitempty"`
	// Spec is the specification of the desired behavior of a job.
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +required
	Spec batchv1.JobSpec `json:"spec,omitempty"`
}

type DaemonJobTemplateMeta struct {
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
	// TypeDaemonJobComplete means that the resource is successfully completed.
	TypeDaemonJobComplete = "Complete"
	// TypeDaemonJobProgressing means that the resources is being created or updated.
	TypeDaemonJobProgressing = "Progressing"
	// TypeDaemonJobDegraded means that the resource failed to reach or maintain its desired state.
	TypeDaemonJobDegraded = "Degraded"
	// TypeDaemonJobUnknown means that the resource is unknown state.
	TypeDaemonJobUnknown = "Unknown"
)

const (
	ReasonOK           = "OK"
	ReasonReconciling  = "Reconciling"
	ReasonBroadcasting = "Broadcasting"
	ReasonProgressing  = "Progressing"
	ReasonFailed       = "Failed"
	ReasonUnknown      = "Unknown"
)

// DaemonJobStatus defines the observed state of DaemonJob.
type DaemonJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// Conditions represent the current state of the DaemonJob resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - Complete: the resource is successfully completed
	// - Progressing: the resources is being created or updated
	// - Degraded: the resource failed to reach or maintain its desired state
	// - Unknown: the resource is unknown state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	// State represent the current state of the DaemonJob resource.
	// See Conditions.
	// +optional
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Suspend",type="string",JSONPath=".spec.jobTemplate.spec.suspend"
// +kubebuilder:printcolumn:name="Containers",type="string",JSONPath=".spec.jobTemplate.spec.template.spec.containers[*].name"
// +kubebuilder:printcolumn:name="Images",type="string",JSONPath=".spec.jobTemplate.spec.template.spec.containers[*].image"

// DaemonJob defines a task to be executed once on every node in the cluster.
// The DaemonJob creates a single "broadcast Job" (configured via spec.broadcastJobSpec).
// This broadcast Job then generates individual "worker Jobs" on every node, based on the spec.JobTemplate.
// Worker Jobs automatically apply tolerations to their Pods, equivalent to those managed by a standard DaemonSet,
// ensuring they can run on all designated nodes.
// DaemonJob and its associated resources may be assigned the following labels for identification and tracking:
//
//   - daemonjob.berquerant.github.io/daemonjob-name: The name of the originating DaemonJob.
//   - daemonjob.berquerant.github.io/node: The name of the specific node the resource is assigned to.
//   - daemonjob.berquerant.github.io/role: The role of the Job: either broadcast or worker.
//   - daemonjob.berquerant.github.io/namespace: The namespace of the originating DaemonJob.
type DaemonJob struct {
	metav1.TypeMeta `json:",inline"`

	// Metadata is a standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of DaemonJob.
	// +required
	Spec DaemonJobSpec `json:"spec,omitempty"`

	// Status defines the observed state of DaemonJob.
	// +optional
	Status DaemonJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DaemonJobList contains a list of DaemonJob
type DaemonJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []DaemonJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DaemonJob{}, &DaemonJobList{})
}
