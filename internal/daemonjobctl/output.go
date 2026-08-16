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
	"fmt"
	"io"
	"strings"

	daemonjobv1 "github.com/berquerant/daemonjob/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// placeholderUID is used as ownerReference.uid for simulated worker Jobs.
// The actual UID is assigned by Kubernetes at runtime when the broadcast Job is created.
const placeholderUID = "00000000-0000-0000-0000-000000000000"

// AnnotatedResource pairs a Kubernetes object with an optional human-readable comment.
type AnnotatedResource struct {
	// Comment is rendered as a YAML comment line immediately before the resource document.
	Comment string
	Object  client.Object
}

// WriteResult holds the resources produced by a generate operation.
type WriteResult struct {
	// Header is written as YAML comment lines at the top of the output.
	Header string
	// Direct resources are written as regular YAML documents.
	Direct []AnnotatedResource
	// Simulated resources are written as a commented-out YAML block.
	// For DaemonJob/DaemonCronJob these are the worker Jobs that the broadcast
	// container creates at runtime; they are simulated offline by daemonjobctl.
	Simulated []AnnotatedResource
}

// WriteYAML serializes a WriteResult to w.
// Direct resources each become a separate YAML document (separated by "---").
// Simulated resources are rendered as a single comment block whose YAML lines
// are all prefixed with "# ", so the file remains valid YAML.
func WriteYAML(w io.Writer, r *WriteResult) error {
	if r.Header != "" {
		for line := range strings.SplitSeq(r.Header, "\n") {
			if _, err := fmt.Fprintf(w, "# %s\n", line); err != nil {
				return err
			}
		}
	}

	for _, ar := range r.Direct {
		setTypeMeta(ar.Object)
		b, err := yaml.Marshal(ar.Object)
		if err != nil {
			return fmt.Errorf("marshal %T: %w", ar.Object, err)
		}
		if _, err := fmt.Fprintln(w, "---"); err != nil {
			return err
		}
		if ar.Comment != "" {
			if _, err := fmt.Fprintf(w, "# %s\n", ar.Comment); err != nil {
				return err
			}
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}

	if len(r.Simulated) == 0 {
		return nil
	}

	// Open the commented block.
	preamble := []string{
		"---",
		"# Worker Jobs (simulated)",
		"# These Jobs are created at runtime by the broadcast container, not directly by the controller.",
		fmt.Sprintf("# NOTE: ownerReference.uid is a placeholder (%s); the actual UID is assigned at runtime.", placeholderUID),
		"#",
	}
	for _, l := range preamble {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return err
		}
	}

	for _, ar := range r.Simulated {
		setTypeMeta(ar.Object)
		b, err := yaml.Marshal(ar.Object)
		if err != nil {
			return fmt.Errorf("marshal simulated %T: %w", ar.Object, err)
		}

		// Per-resource separator with node annotation.
		separator := "#"
		if ar.Comment != "" {
			separator = fmt.Sprintf("# --- %s", ar.Comment)
		}
		if _, err := fmt.Fprintln(w, separator); err != nil {
			return err
		}

		// Each YAML line is prefixed with "# ".
		for line := range strings.SplitSeq(strings.TrimRight(string(b), "\n"), "\n") {
			if _, err := fmt.Fprintf(w, "# %s\n", line); err != nil {
				return err
			}
		}
	}

	return nil
}

// setTypeMeta injects the canonical APIVersion and Kind into an object's TypeMeta.
// The builder functions in internal/controller and internal/broadcast do not set
// TypeMeta, so we must set it before marshalling to YAML.
func setTypeMeta(obj client.Object) {
	switch v := obj.(type) {
	case *corev1.ServiceAccount:
		v.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"}
	case *rbacv1.ClusterRoleBinding:
		v.TypeMeta = metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRoleBinding"}
	case *batchv1.Job:
		v.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}
	case *batchv1.CronJob:
		v.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"}
	case *daemonjobv1.DaemonJob:
		v.TypeMeta = metav1.TypeMeta{APIVersion: daemonjobv1.GroupVersion.String(), Kind: KindDaemonJob}
	case *daemonjobv1.DaemonCronJob:
		v.TypeMeta = metav1.TypeMeta{APIVersion: daemonjobv1.GroupVersion.String(), Kind: KindDaemonCronJob}
	case *daemonjobv1.DaemonCronJobSet:
		v.TypeMeta = metav1.TypeMeta{APIVersion: daemonjobv1.GroupVersion.String(), Kind: KindDaemonCronJobSet}
	}
}
