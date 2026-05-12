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

package controller

import (
	"fmt"
	"slices"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Status struct {
	conditions      *[]metav1.Condition
	typeAvailable   string
	typeProgressing string
	typeDegraded    string
	reasonNotFound  string
}

// NewStatus returns a new Status that writes the conditions.
//
// - typeAvailable means the resources is fully functional
// - typeProgressing means the resource is being created or update
// - typeDegraded means the resources failed to reach or maintain its desired state
// - reasonNotFound is the reason when the resource is not found
func NewStatus(conditions *[]metav1.Condition, typeAvailable, typeProgressing, typeDegraded, reasonNotFound string) *Status {
	return &Status{
		conditions:      conditions,
		typeAvailable:   typeAvailable,
		typeProgressing: typeProgressing,
		typeDegraded:    typeDegraded,
		reasonNotFound:  reasonNotFound,
	}
}

// State returns the current state.
func (s Status) State() (string, bool) {
	var progressing bool
	for _, c := range *s.conditions {
		if c.Status != metav1.ConditionTrue {
			continue
		}
		switch c.Type {
		case s.typeAvailable, s.typeDegraded:
			return c.Type, true
		case s.typeProgressing:
			progressing = true
		}
	}
	if progressing {
		return s.typeProgressing, true
	}
	return "", false
}

func (s *Status) SetProgressing(reason, message string) {
	s.Set(metav1.Condition{
		Type:    s.typeProgressing,
		Status:  ConditionStatus(true),
		Reason:  reason,
		Message: message,
	})
}

func (s *Status) SetNotFound(name string) {
	s.SetPair(false, s.reasonNotFound, name+" is not found")
}

// SetPair writes Available and Degraded statuses.
// Available with Status=ok, Degraded with Status=not ok.
func (s *Status) SetPair(ok bool, reason, message string) {
	x := metav1.Condition{
		Type:   s.typeAvailable,
		Status: ConditionStatus(ok),
		Reason: reason,
	}
	y := metav1.Condition{
		Type:   s.typeDegraded,
		Status: ConditionStatus(!ok),
		Reason: reason,
	}
	if ok {
		x.Message = message
	} else {
		y.Message = message
	}
	s.Set(x, y)
}

func (s *Status) Set(c ...metav1.Condition) {
	for _, x := range c {
		meta.SetStatusCondition(s.conditions, x)
	}
}

func ConditionStatus(v bool) metav1.ConditionStatus {
	if v {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

type JobStatusType int

const (
	JobStatusProgressing JobStatusType = iota
	JobStatusComplete
	JobStatusFailed
)

type JobStatus struct {
	Type    JobStatusType
	Message string
	Reason  string
}

// CheckJobStatus deterines the job status.
func CheckJobStatus(jobConditions []batchv1.JobCondition) *JobStatus {
	var status JobStatus
	for _, c := range jobConditions {
		if c.Status != corev1.ConditionTrue {
			continue
		}
		// https://github.com/kubernetes/kubernetes/blob/66452049f3d692768c39c797b21b793dce80314e/staging/src/k8s.io/api/batch/v1/types.go#L604
		switch c.Type {
		case batchv1.JobComplete:
			status.Type = JobStatusComplete
		case batchv1.JobFailed:
			status.Type = JobStatusFailed
		default:
			continue
		}
		status.Reason = c.Reason
		status.Message = c.Message
		break
	}
	return &status
}

type TerminationLogs []TerminationLog

func (t TerminationLogs) String() string {
	ss := make([]string, len(t))
	for i, x := range t {
		ss[i] = x.String()
	}
	return strings.Join(ss, "; ")
}

type TerminationLog struct {
	Container string
	Kind      string
	Message   string
	ExitCode  int32
}

func (t TerminationLog) String() string {
	return fmt.Sprintf("%s %s exit with %d, termination message=%s",
		t.Kind, t.Container, t.ExitCode, t.Message,
	)
}

// Collect the termination logs from the terminated containers.
// https://kubernetes.io/docs/tasks/debug/debug-application/determine-reason-pod-failure/
func CollectTerminationLogs(status *corev1.PodStatus) TerminationLogs {
	return TerminationLogs(slices.Concat(
		collectTerminationLogs("Container", status.ContainerStatuses),
		collectTerminationLogs("InitContainer", status.InitContainerStatuses),
		collectTerminationLogs("EphemeralContainer", status.EphemeralContainerStatuses),
	))
}

func collectTerminationLogs(kind string, statuses []corev1.ContainerStatus) []TerminationLog {
	var v []TerminationLog
	for _, s := range statuses {
		if x := s.State.Terminated; x != nil {
			v = append(v, TerminationLog{
				Container: s.Name,
				Message:   x.Message,
				ExitCode:  x.ExitCode,
				Kind:      kind,
			})
		}
	}
	return v
}
