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
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func TestSkeleton(t *testing.T) {
	tests := []struct {
		name         string
		skeletonFunc func() client.Object
		expectedKind string
		expectedName string
	}{
		{
			name: "DaemonJob",
			skeletonFunc: func() client.Object {
				return daemonjobctl.DaemonJobSkeleton()
			},
			expectedKind: daemonjobctl.KindDaemonJob,
			expectedName: "my-daemonjob",
		},
		{
			name: "DaemonCronJob",
			skeletonFunc: func() client.Object {
				return daemonjobctl.DaemonCronJobSkeleton()
			},
			expectedKind: daemonjobctl.KindDaemonCronJob,
			expectedName: "my-daemoncronjob",
		},
		{
			name: "DaemonCronJobSet",
			skeletonFunc: func() client.Object {
				return daemonjobctl.DaemonCronJobSetSkeleton()
			},
			expectedKind: daemonjobctl.KindDaemonCronJobSet,
			expectedName: "my-daemoncronjobset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := tt.skeletonFunc()
			assert.Equal(t, tt.expectedName, obj.GetName())
			assert.Equal(t, daemonjobctl.DefaultNamespace, obj.GetNamespace())

			b, err := yaml.Marshal(obj)
			require.NoError(t, err)
			assert.Contains(t, string(b), "kind: "+tt.expectedKind)
			assert.Contains(t, string(b), "name: "+tt.expectedName)
		})
	}
}
