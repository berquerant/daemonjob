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

package util

import (
	"fmt"
	"hash/fnv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	// k8scontroller "k8s.io/kubernetes/pkg/controller"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

// Converts an object into a secure string by hashing it.
func ComputeHash(v any) string {
	hasher := fnv.New32a()
	hashutil.DeepHashObject(hasher, v)
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
}

func PodTemplateHash(template *corev1.PodTemplateSpec, collisionCount *int32) string {
	return ComputeHash(fmt.Sprintf("%s-%d", ComputeHash(template), *collisionCount))
	// return k8scontroller.ComputeHash(template, collisionCount)
}
