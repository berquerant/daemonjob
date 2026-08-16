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
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/berquerant/daemonjob/internal/daemonjobctl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update .golden.yaml files")

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
		goldenFile        string
		expectedDirect    int
		expectedSimulated int
	}{
		{
			name:              "DaemonJob",
			inputRaw:          rawDaemonJob,
			inputNodes:        nodes,
			goldenFile:        "daemonjob.golden.yaml",
			expectedDirect:    3, // SA, CRB, broadcast Job
			expectedSimulated: 2, // 2 worker jobs
		},
		{
			name:              "DaemonCronJob",
			inputRaw:          rawDaemonCronJob,
			inputNodes:        nodes,
			goldenFile:        "daemoncronjob.golden.yaml",
			expectedDirect:    3, // SA, CRB, broadcast CronJob
			expectedSimulated: 2, // 2 worker jobs
		},
		{
			name:              "DaemonCronJobSet",
			inputRaw:          rawDaemonCronJobSet,
			inputNodes:        nodes,
			goldenFile:        "daemoncronjobset.golden.yaml",
			expectedDirect:    2, // 2 CronJobs
			expectedSimulated: 0,
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
			goldenFile:        "passthrough.golden.yaml",
			expectedDirect:    1,
			expectedSimulated: 0,
		},
		{
			name:              "MultiDocument With CustomResource And PassThrough",
			inputRaw:          []byte(string(rawDaemonJob) + "\n---\n" + "apiVersion: v1\nkind: Secret\nmetadata:\n  name: my-secret\n  namespace: default\ntype: Opaque\n"),
			inputNodes:        nodes,
			goldenFile:        "multidoc.golden.yaml",
			expectedDirect:    4, // SA, CRB, Job, Secret
			expectedSimulated: 2, // 2 worker jobs
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
			actual := buf.String()

			goldenPath := filepath.Join("testdata", tt.goldenFile)
			if *updateGolden {
				err := os.WriteFile(goldenPath, buf.Bytes(), 0644)
				require.NoError(t, err)
			}

			expected, err := os.ReadFile(goldenPath)
			require.NoError(t, err, "golden file not found: %s. Run with -update to generate", goldenPath)
			assert.Equal(t, string(expected), actual)
		})
	}
}
