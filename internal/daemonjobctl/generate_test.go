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

	t.Run("DaemonJob", func(t *testing.T) {
		raw, err := os.ReadFile("../../config/samples/daemonjob_v1_daemonjob.yaml")
		require.NoError(t, err)

		res, err := daemonjobctl.Generate(raw, nodes, image, role)
		require.NoError(t, err)
		assert.Len(t, res.Direct, 3)    // SA, CRB, Job (broadcast)
		assert.Len(t, res.Simulated, 2) // 2 worker jobs

		var buf bytes.Buffer
		err = daemonjobctl.WriteYAML(&buf, res)
		require.NoError(t, err)
		out := buf.String()

		// Direct checks
		assert.Contains(t, out, "kind: ServiceAccount")
		assert.Contains(t, out, "kind: ClusterRoleBinding")
		assert.Contains(t, out, "kind: Job")
		assert.Contains(t, out, "name: default-daemonjob-sample-dj") // CRB name prefix uses default namespace
		assert.Contains(t, out, "namespace: default")

		// Simulated checks (commented out)
		assert.Contains(t, out, "# Worker Jobs (simulated)")
		assert.Contains(t, out, "# --- node: node-1")
		assert.Contains(t, out, "# --- node: node-2")
		assert.Contains(t, out, "#   name: daemonjob-sample-dj-node-1")
		assert.Contains(t, out, "#   namespace: default")
	})

	t.Run("DaemonCronJob", func(t *testing.T) {
		raw, err := os.ReadFile("../../config/samples/daemonjob_v1_daemoncronjob.yaml")
		require.NoError(t, err)

		res, err := daemonjobctl.Generate(raw, nodes, image, role)
		require.NoError(t, err)
		assert.Len(t, res.Direct, 3)    // SA, CRB, CronJob (broadcast)
		assert.Len(t, res.Simulated, 2) // 2 worker jobs

		var buf bytes.Buffer
		err = daemonjobctl.WriteYAML(&buf, res)
		require.NoError(t, err)
		out := buf.String()

		assert.Contains(t, out, "kind: ServiceAccount")
		assert.Contains(t, out, "kind: ClusterRoleBinding")
		assert.Contains(t, out, "kind: CronJob")
		assert.Contains(t, out, "# Worker Jobs (simulated)")
		assert.Contains(t, out, "# --- node: node-1")
		assert.Contains(t, out, "# --- node: node-2")
	})

	t.Run("DaemonCronJobSet", func(t *testing.T) {
		raw, err := os.ReadFile("../../config/samples/daemonjob_v1_daemoncronjobset.yaml")
		require.NoError(t, err)

		res, err := daemonjobctl.Generate(raw, nodes, image, role)
		require.NoError(t, err)
		assert.Len(t, res.Direct, 2) // 2 CronJobs (one per node)
		assert.Empty(t, res.Simulated)

		var buf bytes.Buffer
		err = daemonjobctl.WriteYAML(&buf, res)
		require.NoError(t, err)
		out := buf.String()

		assert.Contains(t, out, "kind: CronJob")
		assert.Contains(t, out, "daemoncronjobset-sample-node-1-dcjs")
		assert.Contains(t, out, "daemoncronjobset-sample-node-2-dcjs")
		assert.NotContains(t, out, "# Worker Jobs (simulated)")
	})

	t.Run("PassThrough NonCustomResource", func(t *testing.T) {
		raw := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: my-ns
data:
  key: value
`)
		res, err := daemonjobctl.Generate(raw, nodes, image, role)
		require.NoError(t, err)
		assert.Len(t, res.Direct, 1)
		assert.Empty(t, res.Simulated)

		var buf bytes.Buffer
		err = daemonjobctl.WriteYAML(&buf, res)
		require.NoError(t, err)
		out := buf.String()

		assert.Contains(t, out, "kind: ConfigMap")
		assert.Contains(t, out, "name: my-config")
		assert.Contains(t, out, "namespace: my-ns")
		assert.Contains(t, out, "key: value")
		assert.Contains(t, out, "# Pass-through: ConfigMap/my-config")
	})

	t.Run("MultiDocument With CustomResource And PassThrough", func(t *testing.T) {
		rawDJ, err := os.ReadFile("../../config/samples/daemonjob_v1_daemonjob.yaml")
		require.NoError(t, err)

		multiDoc := string(rawDJ) + "\n---\n" + `
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: default
type: Opaque
`
		res, err := daemonjobctl.Generate([]byte(multiDoc), nodes, image, role)
		require.NoError(t, err)
		assert.Len(t, res.Direct, 4)    // SA, CRB, Job, Secret
		assert.Len(t, res.Simulated, 2) // 2 worker jobs

		var buf bytes.Buffer
		err = daemonjobctl.WriteYAML(&buf, res)
		require.NoError(t, err)
		out := buf.String()

		assert.Contains(t, out, "kind: ServiceAccount")
		assert.Contains(t, out, "kind: Secret")
		assert.Contains(t, out, "name: my-secret")
		assert.Contains(t, out, "# Pass-through: Secret/my-secret")
	})
}
