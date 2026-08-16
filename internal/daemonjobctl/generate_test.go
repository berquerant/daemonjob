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

package daemonjobctl_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/berquerant/daemonjob/internal/daemonjobctl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	const (
		image = "test-image:latest"
		role  = "test-role"
	)
	nodes := []string{"node-1", "node-2"}

	rawDaemonJob, err := os.ReadFile("../../config/samples/daemonjob_v1_daemonjob.yaml")
	require.NoError(t, err)

	rawDaemonCronJob, err := os.ReadFile("../../config/samples/daemonjob_v1_daemoncronjob.yaml")
	require.NoError(t, err)

	rawDaemonCronJobSet, err := os.ReadFile("../../config/samples/daemonjob_v1_daemoncronjobset.yaml")
	require.NoError(t, err)

	tests := []struct {
		name              string
		inputRaw          []byte
		inputNodes        []string
		expectedDirect    int
		expectedSimulated int
		expectedContains  []string
		expectedOmits     []string
	}{
		{
			name:              "DaemonJob",
			inputRaw:          rawDaemonJob,
			inputNodes:        nodes,
			expectedDirect:    3, // SA, CRB, broadcast Job
			expectedSimulated: 2, // 2 worker jobs
			expectedContains: []string{
				"kind: ServiceAccount",
				"kind: ClusterRoleBinding",
				"kind: Job",
				"name: default-daemonjob-sample-dj",
				"namespace: default",
				"# Worker Jobs (simulated)",
				"# --- node: node-1",
				"# --- node: node-2",
				"#   name: daemonjob-sample-dj-node-1",
				"#   namespace: default",
			},
		},
		{
			name:              "DaemonCronJob",
			inputRaw:          rawDaemonCronJob,
			inputNodes:        nodes,
			expectedDirect:    3, // SA, CRB, broadcast CronJob
			expectedSimulated: 2, // 2 worker jobs
			expectedContains: []string{
				"kind: ServiceAccount",
				"kind: ClusterRoleBinding",
				"kind: CronJob",
				"name: default-daemoncronjob-sample-dcj",
				"namespace: default",
				"# Worker Jobs (simulated)",
				"# --- node: node-1",
				"# --- node: node-2",
			},
		},
		{
			name:              "DaemonCronJobSet",
			inputRaw:          rawDaemonCronJobSet,
			inputNodes:        nodes,
			expectedDirect:    2, // 2 CronJobs
			expectedSimulated: 0,
			expectedContains: []string{
				"kind: CronJob",
				"daemoncronjobset-sample-node-1-dcjs",
				"daemoncronjobset-sample-node-2-dcjs",
			},
			expectedOmits: []string{
				"# Worker Jobs (simulated)",
			},
		},
		{
			name: "PassThrough NonCustomResource",
			inputRaw: []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: my-ns
data:
  key: value
`),
			inputNodes:        nodes,
			expectedDirect:    1,
			expectedSimulated: 0,
			expectedContains: []string{
				"kind: ConfigMap",
				"name: my-config",
				"namespace: my-ns",
				"key: value",
				"# Pass-through: ConfigMap/my-config",
			},
			expectedOmits: []string{
				"# Worker Jobs (simulated)",
			},
		},
		{
			name:              "MultiDocument With CustomResource And PassThrough",
			inputRaw:          []byte(string(rawDaemonJob) + "\n---\n" + "apiVersion: v1\nkind: Secret\nmetadata:\n  name: my-secret\n  namespace: default\ntype: Opaque\n"),
			inputNodes:        nodes,
			expectedDirect:    4, // SA, CRB, Job, Secret
			expectedSimulated: 2, // 2 worker jobs
			expectedContains: []string{
				"kind: ServiceAccount",
				"kind: Secret",
				"name: my-secret",
				"# Pass-through: Secret/my-secret",
				"# Worker Jobs (simulated)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := daemonjobctl.Generate(tt.inputRaw, tt.inputNodes, image, role)
			require.NoError(t, err)
			assert.Len(t, res.Direct, tt.expectedDirect)
			assert.Len(t, res.Simulated, tt.expectedSimulated)

			var buf bytes.Buffer
			err = daemonjobctl.WriteYAML(&buf, res)
			require.NoError(t, err)
			out := buf.String()

			for _, exp := range tt.expectedContains {
				assert.Contains(t, out, exp)
			}
			for _, omit := range tt.expectedOmits {
				assert.NotContains(t, out, omit)
			}
		})
	}
}
