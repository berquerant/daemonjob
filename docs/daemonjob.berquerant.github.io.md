# API Reference

## Packages
- [daemonjob.berquerant.github.io/v1](#daemonjobberquerantgithubiov1)


## daemonjob.berquerant.github.io/v1

Package v1 contains API Schema definitions for the daemonjob v1 API group.

### Resource Types
- [DaemonCronJob](#daemoncronjob)
- [DaemonCronJobList](#daemoncronjoblist)
- [DaemonCronJobSet](#daemoncronjobset)
- [DaemonCronJobSetList](#daemoncronjobsetlist)
- [DaemonJob](#daemonjob)
- [DaemonJobList](#daemonjoblist)



#### DaemonCronJob



DaemonCronJob defines a task that runs periodically on every node in the cluster.
It functions by creating a CronJob, which is responsible for triggering a DaemonJob's broadcast Job at scheduled intervals.
DaemonCronJob and its associated resources may be assigned the following labels for identification and tracking:

  - daemonjob.berquerant.github.io/daemoncronjob-name: The name of the originating DaemonCronJob.
  - daemonjob.berquerant.github.io/node: The name of the specific node the resource is assigned to.
  - daemonjob.berquerant.github.io/role: The role of the Job: either broadcast or worker.
  - daemonjob.berquerant.github.io/namespace: The namespace of the originating DaemonCronJob.



_Appears in:_
- [DaemonCronJobList](#daemoncronjoblist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `daemonjob.berquerant.github.io/v1` | | |
| `kind` _string_ | `DaemonCronJob` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[DaemonCronJobSpec](#daemoncronjobspec)_ | Spec defines the desired state of DaemonCronJob |  | Required: \{\} <br /> |
| `status` _[DaemonCronJobStatus](#daemoncronjobstatus)_ | Status defines the observed state of DaemonCronJob |  | Optional: \{\} <br /> |


#### DaemonCronJobList



DaemonCronJobList contains a list of DaemonCronJob





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `daemonjob.berquerant.github.io/v1` | | |
| `kind` _string_ | `DaemonCronJobList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[DaemonCronJob](#daemoncronjob) array_ |  |  |  |


#### DaemonCronJobSet



DaemonCronJobSet defines a set of CronJobs designed to run across all nodes in a cluster.
It generates one CronJob for each node based on the spec.cronJobTemplate.
Each generated CronJob is responsible for triggering a Job that runs exclusively on its assigned node.
These per-node Jobs are referred to as worker Jobs.
Worker Jobs automatically apply tolerations to their Pods, equivalent to those managed by a DaemonSet,
ensuring they can be scheduled on all target nodes.
DaemonCronJobSet and its associated resources may be assigned the following labels for identification and tracking:

  - daemonjob.berquerant.github.io/daemoncronjobset-name: The name of the originating DaemonCronJobSet.
  - daemonjob.berquerant.github.io/node: The name of the specific node the resource is assigned to.
  - daemonjob.berquerant.github.io/role: The role of the Job: either broadcast or worker.
  - daemonjob.berquerant.github.io/namespace: The namespace of the originating DaemonCronJobSet.



_Appears in:_
- [DaemonCronJobSetList](#daemoncronjobsetlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `daemonjob.berquerant.github.io/v1` | | |
| `kind` _string_ | `DaemonCronJobSet` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[DaemonCronJobSetSpec](#daemoncronjobsetspec)_ | Spec defines the desired state of DaemonCronJobSet |  | Required: \{\} <br /> |
| `status` _[DaemonCronJobSetStatus](#daemoncronjobsetstatus)_ | Status defines the observed state of DaemonCronJobSet |  | Optional: \{\} <br /> |


#### DaemonCronJobSetList



DaemonCronJobSetList contains a list of DaemonCronJobSet.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `daemonjob.berquerant.github.io/v1` | | |
| `kind` _string_ | `DaemonCronJobSetList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[DaemonCronJobSet](#daemoncronjobset) array_ |  |  |  |


#### DaemonCronJobSetSpec



DaemonCronJobSetSpec defines the desired state of DaemonCronJobSet.



_Appears in:_
- [DaemonCronJobSet](#daemoncronjobset)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `cronJobTemplate` _[DaemonCronJobSetTemplateSpec](#daemoncronjobsettemplatespec)_ | CronJobTemplate specifies the cronjob that will be created when executing a DaemonCronJob. |  | Required: \{\} <br /> |
| `nodeSelector` _object (keys:string, values:string)_ | nodeSelector is a selector which must be true for the job to fit on a node.<br />Selector which must match a node's labels for the job to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node |  | Optional: \{\} <br /> |


#### DaemonCronJobSetStatus



DaemonCronJobSetStatus defines the observed state of DaemonCronJobSet.



_Appears in:_
- [DaemonCronJobSet](#daemoncronjobset)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#condition-v1-meta) array_ | conditions represent the current state of the DaemonCronJobSet resource.<br />Each condition has a unique type and reflects the status of a specific aspect of the resource.<br />Standard condition types include:<br />- "Available": the resource is fully functional<br />- "Degraded": the resource failed to reach or maintain its desired state<br />The status of each condition is one of True, False, or Unknown. |  | Optional: \{\} <br /> |
| `numberDesired` _integer_ | The number of nodes that should be created the daemon cronjobs. |  | Required: \{\} <br /> |
| `numberAvailable` _integer_ | The number of nodes that should be created the daemon cronjobs and have one or more of the daemon cronjobs available. |  | Optional: \{\} <br /> |


#### DaemonCronJobSetTemplateMeta







_Appears in:_
- [DaemonCronJobSetTemplateSpec](#daemoncronjobsettemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `labels` _object (keys:string, values:string)_ | Labels is a map of string keys and values that can be used to organize and categorize (scope and select) objects.<br />May match selectors of replication controllers and services.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels |  | Optional: \{\} <br /> |
| `annotations` _object (keys:string, values:string)_ | Annotations is an unstructured key value map stored with a resource that may be set by<br />external tools to store and retrieve arbitrary metadata.<br />They are not queryable and should be preserved when modifying objects.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations |  | Optional: \{\} <br /> |


#### DaemonCronJobSetTemplateSpec







_Appears in:_
- [DaemonCronJobSetSpec](#daemoncronjobsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[DaemonCronJobSetTemplateMeta](#daemoncronjobsettemplatemeta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[CronJobSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#cronjobspec-v1-batch)_ | Specification of the desired behavior of a cron job, including the schedule.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  | Required: \{\} <br /> |


#### DaemonCronJobSpec



DaemonCronJobSpec defines the desired state of DaemonCronJob



_Appears in:_
- [DaemonCronJob](#daemoncronjob)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `cronJobTemplate` _[DaemonCronJobTemplateSpec](#daemoncronjobtemplatespec)_ | CronJobTemplate specifies the cronjob that will be created when executing a DaemonCronJob. |  | Required: \{\} <br /> |
| `nodeSelector` _object (keys:string, values:string)_ | nodeSelector is a selector which must be true for the job to fit on a node.<br />Selector which must match a node's labels for the job to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node |  | Optional: \{\} <br /> |
| `broadcastJobSpec` _[DaemonJobBroadcastJobSpec](#daemonjobbroadcastjobspec)_ | BroadcastJobSpec is a spec of the broadcast job. |  | Optional: \{\} <br /> |


#### DaemonCronJobStatus



DaemonCronJobStatus defines the observed state of DaemonCronJob.



_Appears in:_
- [DaemonCronJob](#daemoncronjob)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#condition-v1-meta) array_ | Conditions represent the current state of the DaemonCronJob resource.<br />Each condition has a unique type and reflects the status of a specific aspect of the resource.<br />Standard condition types include:<br />- "Available": the resource is fully functional<br />- "Degraded": the resource failed to reach or maintain its desired state<br />The status of each condition is one of True, False, or Unknown. |  | Optional: \{\} <br /> |


#### DaemonCronJobTemplateMeta







_Appears in:_
- [DaemonCronJobTemplateSpec](#daemoncronjobtemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `labels` _object (keys:string, values:string)_ | Labels is a map of string keys and values that can be used to organize and categorize (scope and select) objects.<br />May match selectors of replication controllers and services.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels |  | Optional: \{\} <br /> |
| `annotations` _object (keys:string, values:string)_ | Annotations is an unstructured key value map stored with a resource that may be set by<br />external tools to store and retrieve arbitrary metadata.<br />They are not queryable and should be preserved when modifying objects.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations |  | Optional: \{\} <br /> |


#### DaemonCronJobTemplateSpec







_Appears in:_
- [DaemonCronJobSpec](#daemoncronjobspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[DaemonCronJobTemplateMeta](#daemoncronjobtemplatemeta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[CronJobSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#cronjobspec-v1-batch)_ | Specification of the desired behavior of a cron job, including the schedule.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  | Required: \{\} <br /> |


#### DaemonJob



DaemonJob defines a task to be executed once on every node in the cluster.
The DaemonJob creates a single "broadcast Job" (configured via spec.broadcastJobSpec).
This broadcast Job then generates individual "worker Jobs" on every node, based on the spec.JobTemplate.
Worker Jobs automatically apply tolerations to their Pods, equivalent to those managed by a standard DaemonSet,
ensuring they can run on all designated nodes.
DaemonJob and its associated resources may be assigned the following labels for identification and tracking:

  - daemonjob.berquerant.github.io/daemonjob-name: The name of the originating DaemonJob.
  - daemonjob.berquerant.github.io/node: The name of the specific node the resource is assigned to.
  - daemonjob.berquerant.github.io/role: The role of the Job: either broadcast or worker.
  - daemonjob.berquerant.github.io/namespace: The namespace of the originating DaemonJob.



_Appears in:_
- [DaemonJobList](#daemonjoblist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `daemonjob.berquerant.github.io/v1` | | |
| `kind` _string_ | `DaemonJob` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[DaemonJobSpec](#daemonjobspec)_ | Spec defines the desired state of DaemonJob. |  | Required: \{\} <br /> |
| `status` _[DaemonJobStatus](#daemonjobstatus)_ | Status defines the observed state of DaemonJob. |  | Optional: \{\} <br /> |


#### DaemonJobBroadcastJobSpec



DaemonJobBroadcastJobSpec defines the spec of the broadcast Job.



_Appears in:_
- [DaemonCronJobSpec](#daemoncronjobspec)
- [DaemonJobSpec](#daemonjobspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#affinity-v1-core)_ | If specified, the pod's scheduling constraints<br />Affinity is a group of affinity scheduling rules. |  | Optional: \{\} <br /> |
| `imgePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#localobjectreference-v1-core) array_ | ImagePullSecrets is an optional list of references to secrets in the same<br />namespace to use for pulling any of the images used by this PodSpec.<br />If specified, these secrets will be passed to individual puller implementations for them to use.<br />More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod |  | Optional: \{\} <br /> |
| `nodeName` _string_ | NodeName indicates in which node this pod is scheduled.<br />If empty, this pod is a candidate for scheduling by the scheduler defined in schedulerName.<br />Once this field is set, the kubelet for this node becomes responsible for the lifecycle of this pod.<br />This field should not be used to express a desire  for the pod to be scheduled on a specific node.<br />https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodename |  | Optional: \{\} <br /> |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must be true for the pod to fit on a node.<br />Selector which must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ |  | Optional: \{\} <br /> |
| `preemptionPolicy` _[PreemptionPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#preemptionpolicy-v1-core)_ | PreemptionPolicy is the Policy for preempting pods with lower priority.<br />One of Never, PreemptLowerPriority.<br />Defaults to PreemptLowerPriority if unset.<br />Possible enum values:<br /> - `"Never"` means that pod never preempts other pods with lower priority.<br /> - `"PreemptLowerPriority"` means that pod can preempt other pods with lower priority. |  |  |
| `priority` _integer_ | The priority value.<br />Various system components use this field to find the priority of the pod.<br />When Priority Admission Controller is enabled, it prevents users from setting this field.<br />The admission controller populates this field from PriorityClassName.<br />The higher the value, the higher the priority. |  | Optional: \{\} <br /> |
| `priorityClassName` _string_ | If specified, indicates the pod's priority.<br />"system-node-critical" and "system-cluster-critical" are two special keywords<br />which indicate the highest priorities with the former being the highest priority.<br />Any other name must be defined by creating a PriorityClass object with that name.<br />If not specified, the pod priority will be default or zero if there is no default. |  | Optional: \{\} <br /> |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#resourcerequirements-v1-core)_ | Resources specifies the compute resources required by the container.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |  | Optional: \{\} <br /> |
| `schedulerName` _string_ | If specified, the pod will be dispatched by specified scheduler.<br />If not specified, the pod will be dispatched by default scheduler. |  | Optional: \{\} <br /> |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#toleration-v1-core) array_ | If specified, the pod's tolerations.<br />The pod this Toleration is attached to tolerates any taint that matches<br />the triple <key,value,effect> using the matching operator <operator>. |  | Optional: \{\} <br /> |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#topologyspreadconstraint-v1-core) array_ | TopologySpreadConstraints describes how a group of pods ought to spread across topology domains.<br />Scheduler will schedule pods in a way which abides by the constraints.<br />All topologySpreadConstraints are ANDed.<br />TopologySpreadConstraint specifies how to spread matching pods among the given topology. |  | Optional: \{\} <br /> |


#### DaemonJobList



DaemonJobList contains a list of DaemonJob





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `daemonjob.berquerant.github.io/v1` | | |
| `kind` _string_ | `DaemonJobList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[DaemonJob](#daemonjob) array_ |  |  |  |


#### DaemonJobSpec



DaemonJobSpec defines the desired state of DaemonJob.



_Appears in:_
- [DaemonJob](#daemonjob)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `jobTemplate` _[DaemonJobTemplateSpec](#daemonjobtemplatespec)_ | JobTemplate specifies the job that will be created when executing a DaemonJob. |  | Required: \{\} <br /> |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must be true for the job to fit on a node.<br />Selector which must match a node's labels for the job to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node |  | Optional: \{\} <br /> |
| `broadcastJobSpec` _[DaemonJobBroadcastJobSpec](#daemonjobbroadcastjobspec)_ | BroadcastJobSpec is a spec of the broadcast job. |  | Optional: \{\} <br /> |


#### DaemonJobStatus



DaemonJobStatus defines the observed state of DaemonJob.



_Appears in:_
- [DaemonJob](#daemonjob)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#condition-v1-meta) array_ | Conditions represent the current state of the DaemonJob resource.<br />Each condition has a unique type and reflects the status of a specific aspect of the resource.<br />Standard condition types include:<br />- Complete: the resource is successfully completed<br />- Progressing: the resources is being created or updated<br />- Degraded: the resource failed to reach or maintain its desired state<br />- Unknown: the resource is unknown state<br />The status of each condition is one of True, False, or Unknown. |  | Optional: \{\} <br /> |
| `state` _string_ | State represent the current state of the DaemonJob resource.<br />See Conditions. |  | Optional: \{\} <br /> |


#### DaemonJobTemplateMeta







_Appears in:_
- [DaemonJobTemplateSpec](#daemonjobtemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `labels` _object (keys:string, values:string)_ | Labels is a map of string keys and values that can be used to organize and categorize (scope and select) objects.<br />May match selectors of replication controllers and services.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels |  | Optional: \{\} <br /> |
| `annotations` _object (keys:string, values:string)_ | Annotations is an unstructured key value map stored with a resource that may be set by<br />external tools to store and retrieve arbitrary metadata.<br />They are not queryable and should be preserved when modifying objects.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations |  | Optional: \{\} <br /> |


#### DaemonJobTemplateSpec







_Appears in:_
- [DaemonJobSpec](#daemonjobspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[DaemonJobTemplateMeta](#daemonjobtemplatemeta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[JobSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/vv0.24.1/#jobspec-v1-batch)_ | Spec is the specification of the desired behavior of a job.<br />https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  | Required: \{\} <br /> |


