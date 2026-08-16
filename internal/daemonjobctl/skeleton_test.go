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
	"testing"

	"github.com/berquerant/daemonjob/internal/daemonjobctl"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestSkeleton(t *testing.T) {
	t.Run("DaemonJob", func(t *testing.T) {
		sk := daemonjobctl.DaemonJobSkeleton()
		assert.Equal(t, "DaemonJob", sk.Kind)
		assert.Equal(t, "my-daemonjob", sk.Name)
		assert.NotEmpty(t, sk.Spec.JobTemplate.Spec.Template.Spec.Containers)

		b, err := yaml.Marshal(sk)
		assert.NoError(t, err)
		assert.Contains(t, string(b), "kind: DaemonJob")
	})

	t.Run("DaemonCronJob", func(t *testing.T) {
		sk := daemonjobctl.DaemonCronJobSkeleton()
		assert.Equal(t, "DaemonCronJob", sk.Kind)
		assert.Equal(t, "my-daemoncronjob", sk.Name)
		assert.NotEmpty(t, sk.Spec.CronJobTemplate.Spec.Schedule)

		b, err := yaml.Marshal(sk)
		assert.NoError(t, err)
		assert.Contains(t, string(b), "kind: DaemonCronJob")
	})

	t.Run("DaemonCronJobSet", func(t *testing.T) {
		sk := daemonjobctl.DaemonCronJobSetSkeleton()
		assert.Equal(t, "DaemonCronJobSet", sk.Kind)
		assert.Equal(t, "my-daemoncronjobset", sk.Name)
		assert.NotEmpty(t, sk.Spec.CronJobTemplate.Spec.Schedule)

		b, err := yaml.Marshal(sk)
		assert.NoError(t, err)
		assert.Contains(t, string(b), "kind: DaemonCronJobSet")
	})
}
