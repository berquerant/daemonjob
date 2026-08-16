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
	"bytes"
	"errors"
	"fmt"
	"io"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	"github.com/berquerant/daemonjob/internal/broadcast"
	"github.com/berquerant/daemonjob/internal/controller"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

// Generate parses raw YAML (which may contain one or more YAML documents) and produces a WriteResult
// containing the Kubernetes resources. For custom resources (DaemonJob, DaemonCronJob, DaemonCronJobSet),
// it generates the resources that daemonjob would create. For non-custom resources, it passes them through unchanged.
//
// nodes is a slice of Kubernetes node names against which worker Jobs are simulated.
// For DaemonJob and DaemonCronJob, the broadcast Job / CronJob are included as Direct
// resources, and the per-node worker Jobs are included as Simulated (commented-out) resources.
// For DaemonCronJobSet, all per-node CronJobs are Direct (no broadcast indirection).
//
// image and clusterRole are required for DaemonJob / DaemonCronJob broadcast resources;
// they are ignored for DaemonCronJobSet and pass-through resources.
func Generate(raw []byte, nodes []string, image, clusterRole string) (*WriteResult, error) {
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096)
	combinedResult := &WriteResult{}

	for {
		var rawDoc map[string]any
		if err := decoder.Decode(&rawDoc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode yaml: %w", err)
		}
		if len(rawDoc) == 0 {
			continue
		}

		docBytes, err := yaml.Marshal(rawDoc)
		if err != nil {
			return nil, fmt.Errorf("marshal doc: %w", err)
		}

		obj, err := parseManifest(docBytes)
		if err != nil {
			return nil, err
		}

		var res *WriteResult
		switch v := obj.(type) {
		case *daemonjobv1.DaemonJob:
			res = generateDaemonJob(v, nodes, image, clusterRole)
		case *daemonjobv1.DaemonCronJob:
			res = generateDaemonCronJob(v, nodes, image, clusterRole)
		case *daemonjobv1.DaemonCronJobSet:
			res = generateDaemonCronJobSet(v, nodes)
		case *unstructured.Unstructured:
			res = &WriteResult{
				Direct: []AnnotatedResource{
					{
						Comment: fmt.Sprintf("Pass-through: %s/%s", v.GetKind(), v.GetName()),
						Object:  v,
					},
				},
			}
		default:
			return nil, fmt.Errorf("unsupported object type: %T", obj)
		}

		combinedResult.Direct = append(combinedResult.Direct, res.Direct...)
		combinedResult.Simulated = append(combinedResult.Simulated, res.Simulated...)
	}

	return combinedResult, nil
}

// parseManifest decodes a single raw YAML document into a typed CRD object, or an Unstructured object if not a custom resource.
func parseManifest(raw []byte) (any, error) {
	var meta struct {
		Kind string `json:"kind"`
	}
	if err := yaml.Unmarshal(raw, &meta); err != nil {
		return nil, fmt.Errorf("parse manifest kind: %w", err)
	}

	switch meta.Kind {
	case KindDaemonJob:
		var obj daemonjobv1.DaemonJob
		if err := yaml.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("parse DaemonJob: %w", err)
		}
		if obj.Namespace == "" {
			obj.Namespace = DefaultNamespace
		}
		return &obj, nil
	case KindDaemonCronJob:
		var obj daemonjobv1.DaemonCronJob
		if err := yaml.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("parse DaemonCronJob: %w", err)
		}
		if obj.Namespace == "" {
			obj.Namespace = DefaultNamespace
		}
		return &obj, nil
	case KindDaemonCronJobSet:
		var obj daemonjobv1.DaemonCronJobSet
		if err := yaml.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("parse DaemonCronJobSet: %w", err)
		}
		if obj.Namespace == "" {
			obj.Namespace = DefaultNamespace
		}
		return &obj, nil
	default:
		// Not a daemonjob custom resource; pass-through as Unstructured.
		var unstr unstructured.Unstructured
		if err := yaml.Unmarshal(raw, &unstr.Object); err != nil {
			return nil, fmt.Errorf("parse unstructured object: %w", err)
		}
		return &unstr, nil
	}
}

// generateDaemonJob produces the broadcast resources (SA, CRB, broadcast Job) as Direct,
// and the per-node worker Jobs as Simulated.
func generateDaemonJob(dj *daemonjobv1.DaemonJob, nodes []string, image, clusterRole string) *WriteResult {
	args := controller.NewDaemonJobBroadcastArgs(dj, clusterRole, image)

	direct := []AnnotatedResource{
		{Comment: fmt.Sprintf("Source Custom Resource: DaemonJob/%s", dj.Name), Object: dj},
		{Comment: "ServiceAccount for the broadcast Job", Object: args.ServiceAccount()},
		{Comment: "ClusterRoleBinding for the broadcast Job", Object: args.ClusterRoleBinding()},
		{Comment: "Broadcast Job: spawns one worker Job per node at runtime", Object: args.Job()},
	}

	// Worker Job names follow the pattern: {broadcastJobName}-{nodeName}
	broadcastJobName := args.JobName()
	cfg := &broadcast.Config{
		SelfName:      broadcastJobName,
		Namespace:     dj.Namespace,
		DaemonJobName: dj.Name,
		ControllerUID: types.UID(placeholderUID),
	}
	// Client is not used by BuildWorkerJob; set to nil.
	runner := &broadcast.Runner{Config: cfg}

	simulated := make([]AnnotatedResource, 0, len(nodes))
	for _, node := range nodes {
		job := runner.BuildWorkerJob(node, &dj.Spec)
		simulated = append(simulated, AnnotatedResource{
			Comment: fmt.Sprintf("node: %s", node),
			Object:  job,
		})
	}

	return &WriteResult{
		Header:    fmt.Sprintf("Input: DaemonJob/%s (namespace: %s)", dj.Name, dj.Namespace),
		Direct:    direct,
		Simulated: simulated,
	}
}

// generateDaemonCronJob produces the broadcast resources (SA, CRB, broadcast CronJob) as Direct,
// and the per-node worker Jobs that would be created on each cron trigger as Simulated.
func generateDaemonCronJob(dcj *daemonjobv1.DaemonCronJob, nodes []string, image, clusterRole string) *WriteResult {
	args := controller.NewDaemonCronJobBroadcastArgs(dcj, clusterRole, image)

	direct := []AnnotatedResource{
		{Comment: fmt.Sprintf("Source Custom Resource: DaemonCronJob/%s", dcj.Name), Object: dcj},
		{Comment: "ServiceAccount for the broadcast CronJob", Object: args.ServiceAccount()},
		{Comment: "ClusterRoleBinding for the broadcast CronJob", Object: args.ClusterRoleBinding()},
		{Comment: "Broadcast CronJob: on each trigger spawns a broadcast Job which creates per-node worker Jobs", Object: args.CronJob()},
	}

	// Worker Job names follow the pattern: {broadcastJobName}-{nodeName}
	// where broadcastJobName is the name of the Job spawned by the CronJob on each trigger.
	broadcastJobName := args.JobName()
	daemonJobSpec := args.AsDaemonJob().Spec
	cfg := &broadcast.Config{
		SelfName:          broadcastJobName,
		Namespace:         dcj.Namespace,
		DaemonCronJobName: dcj.Name,
		ControllerUID:     types.UID(placeholderUID),
	}
	// Client is not used by BuildWorkerJob; set to nil.
	runner := &broadcast.Runner{Config: cfg}

	simulated := make([]AnnotatedResource, 0, len(nodes))
	for _, node := range nodes {
		job := runner.BuildWorkerJob(node, &daemonJobSpec)
		simulated = append(simulated, AnnotatedResource{
			Comment: fmt.Sprintf("node: %s", node),
			Object:  job,
		})
	}

	return &WriteResult{
		Header:    fmt.Sprintf("Input: DaemonCronJob/%s (namespace: %s)", dcj.Name, dcj.Namespace),
		Direct:    direct,
		Simulated: simulated,
	}
}

// generateDaemonCronJobSet produces one CronJob per node as Direct resources.
// DaemonCronJobSet does not use the broadcast pattern, so all resources are direct.
func generateDaemonCronJobSet(dcjs *daemonjobv1.DaemonCronJobSet, nodes []string) *WriteResult {
	args := controller.NewDaemonCronJobSetArgs(dcjs)

	direct := make([]AnnotatedResource, 0, len(nodes)+1)
	direct = append(direct, AnnotatedResource{
		Comment: fmt.Sprintf("Source Custom Resource: DaemonCronJobSet/%s", dcjs.Name),
		Object:  dcjs,
	})
	for _, node := range nodes {
		cronJob := args.NewCronJobForNode(node)
		direct = append(direct, AnnotatedResource{
			Comment: fmt.Sprintf("CronJob for node: %s", node),
			Object:  cronJob,
		})
	}

	return &WriteResult{
		Header: fmt.Sprintf("Input: DaemonCronJobSet/%s (namespace: %s)", dcjs.Name, dcjs.Namespace),
		Direct: direct,
	}
}
