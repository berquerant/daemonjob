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

package controller_test

import (
	"testing"

	"github.com/berquerant/daemonjob/internal/controller"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func assertConditions(t *testing.T, want, got []metav1.Condition) {
	type Cond struct {
		Type    string
		Status  metav1.ConditionStatus
		Reason  string
		Message string
	}
	var (
		asCond = func(c metav1.Condition) Cond {
			return Cond{
				Type:    c.Type,
				Status:  c.Status,
				Reason:  c.Reason,
				Message: c.Message,
			}
		}
		w = make([]Cond, len(want))
		g = make([]Cond, len(got))
	)
	for i, x := range want {
		w[i] = asCond(x)
	}
	for i, x := range got {
		g[i] = asCond(x)
	}
	assert.Equal(t, w, g)
}

func TestStatus(t *testing.T) {
	const (
		available   = "Available"
		degraded    = "Degraded"
		progressing = "Progressing"
		notFound    = "NotFound"
	)
	type Status struct {
		c []metav1.Condition
		s *controller.Status
	}
	var (
		newStatus = func() *Status {
			var s Status
			s.s = controller.NewStatus(&s.c, available, progressing, degraded, notFound)
			return &s
		}
		newCondition = func(typ string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
			var c metav1.Condition
			c.Type = typ
			c.Status = status
			c.Reason = reason
			c.Message = message
			return c
		}
	)

	t.Run("State", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			conditions []metav1.Condition
			want       string
			unknown    bool
		}{
			{
				name:    "empty conditions",
				unknown: true,
			},
			{
				name: "no true statuses",
				conditions: []metav1.Condition{
					newCondition(available, metav1.ConditionFalse, "", ""),
				},
				unknown: true,
			},
			{
				name: "progressing",
				conditions: []metav1.Condition{
					newCondition(progressing, metav1.ConditionTrue, "", ""),
				},
				want: progressing,
			},
			{
				name: "degraded",
				conditions: []metav1.Condition{
					newCondition(progressing, metav1.ConditionTrue, "", ""),
					newCondition(degraded, metav1.ConditionTrue, "", ""),
				},
				want: degraded,
			},
			{
				name: "available",
				conditions: []metav1.Condition{
					newCondition(available, metav1.ConditionTrue, "", ""),
					newCondition(progressing, metav1.ConditionTrue, "", ""),
					newCondition(degraded, metav1.ConditionFalse, "", ""),
				},
				want: available,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := newStatus()
				s.s.Set(tc.conditions...)
				got, ok := s.s.State()
				if tc.unknown {
					assert.False(t, ok)
					return
				}
				assert.True(t, ok)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("SetProgressing", func(t *testing.T) {
		s := newStatus()
		s.s.SetProgressing("REASON", "MESSAGE")
		assertConditions(t, []metav1.Condition{
			newCondition(progressing, metav1.ConditionTrue, "REASON", "MESSAGE"),
		}, s.c)
	})

	t.Run("SetNotFound", func(t *testing.T) {
		s := newStatus()
		s.s.SetNotFound("NAME")
		assertConditions(t, []metav1.Condition{
			newCondition(available, metav1.ConditionFalse, notFound, ""),
			newCondition(degraded, metav1.ConditionTrue, notFound, "NAME is not found"),
		}, s.c)
	})

	t.Run("SetPair", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			s := newStatus()
			s.s.SetPair(true, "REASON", "MESSAGE")
			assertConditions(t, []metav1.Condition{
				newCondition(available, metav1.ConditionTrue, "REASON", "MESSAGE"),
				newCondition(degraded, metav1.ConditionFalse, "REASON", ""),
			}, s.c)
		})
		t.Run("not ok", func(t *testing.T) {
			s := newStatus()
			s.s.SetPair(false, "REASON", "MESSAGE")
			assertConditions(t, []metav1.Condition{
				newCondition(available, metav1.ConditionFalse, "REASON", ""),
				newCondition(degraded, metav1.ConditionTrue, "REASON", "MESSAGE"),
			}, s.c)
		})
	})

	t.Run("Set", func(t *testing.T) {
		s := newStatus()
		s.s.Set()
		assert.Zero(t, s.c)

		c1 := newCondition(available, metav1.ConditionTrue, "", "")
		s.s.Set(c1)
		assertConditions(t, []metav1.Condition{
			c1,
		}, s.c)

		c2 := newCondition(degraded, metav1.ConditionFalse, "", "")
		s.s.Set(c2)
		assertConditions(t, []metav1.Condition{
			c1,
			c2,
		}, s.c)
	})
}

func TestConditionStatus(t *testing.T) {
	assert.Equal(t, metav1.ConditionTrue, controller.ConditionStatus(true))
	assert.Equal(t, metav1.ConditionFalse, controller.ConditionStatus(false))
}

func TestCheckJobStatus(t *testing.T) {
	for _, tc := range []struct {
		name       string
		conditions []batchv1.JobCondition
		want       *controller.JobStatus
	}{
		{
			name:       "progressing",
			conditions: []batchv1.JobCondition{},
			want: &controller.JobStatus{
				Type: controller.JobStatusProgressing,
			},
		},
		{
			name: "complete",
			conditions: []batchv1.JobCondition{
				{
					Type:    batchv1.JobComplete,
					Status:  corev1.ConditionTrue,
					Reason:  "CompletionsReached",
					Message: "Reached expected number of succeeded pods",
				},
			},
			want: &controller.JobStatus{
				Type:    controller.JobStatusComplete,
				Reason:  "CompletionsReached",
				Message: "Reached expected number of succeeded pods",
			},
		},
		{
			name: "failed",
			conditions: []batchv1.JobCondition{
				{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "BackoffLimitExceeded",
					Message: "Job has reached the specified backoff limit",
				},
			},
			want: &controller.JobStatus{
				Type:    controller.JobStatusFailed,
				Reason:  "BackoffLimitExceeded",
				Message: "Job has reached the specified backoff limit",
			},
		},
		{
			name: "ignore status false",
			conditions: []batchv1.JobCondition{
				{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionFalse,
					Reason:  "BackoffLimitExceeded",
					Message: "Job has reached the specified backoff limit",
				},
			},
			want: &controller.JobStatus{
				Type: controller.JobStatusProgressing,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, controller.CheckJobStatus(tc.conditions))
		})
	}
}

func TestCollectTerminationLogs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status *corev1.PodStatus
		want   controller.TerminationLogs
	}{
		{
			name: "running",
			status: &corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "main",
						Image: "image",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
			want: controller.TerminationLogs(nil),
		},
		{
			name: "waiting",
			status: &corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "main",
						Image: "image",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Message: "MESSAGE",
							},
						},
					},
				},
			},
			want: controller.TerminationLogs(nil),
		},
		{
			name: "terminated but no message",
			status: &corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "main",
						Image: "image",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: 0,
							},
						},
					},
				},
			},
			want: controller.TerminationLogs([]controller.TerminationLog{
				{
					Container: "main",
					ExitCode:  0,
					Kind:      "Container",
				},
			}),
		},
		{
			name: "got log",
			status: &corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "main",
						Image: "image",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: 1,
								Message:  "MESSAGE",
							},
						},
					},
				},
			},
			want: controller.TerminationLogs([]controller.TerminationLog{
				{
					Container: "main",
					Message:   "MESSAGE",
					ExitCode:  1,
					Kind:      "Container",
				},
			}),
		},
		{
			name: "got log from init container",
			status: &corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "main",
						Image: "image",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: 1,
								Message:  "MESSAGE",
							},
						},
					},
				},
			},
			want: controller.TerminationLogs([]controller.TerminationLog{
				{
					Container: "main",
					Message:   "MESSAGE",
					ExitCode:  1,
					Kind:      "InitContainer",
				},
			}),
		},
		{
			name: "got log from ephemeral container",
			status: &corev1.PodStatus{
				EphemeralContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "main",
						Image: "image",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: 1,
								Message:  "MESSAGE",
							},
						},
					},
				},
			},
			want: controller.TerminationLogs([]controller.TerminationLog{
				{
					Container: "main",
					Message:   "MESSAGE",
					ExitCode:  1,
					Kind:      "EphemeralContainer",
				},
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, controller.CollectTerminationLogs(tc.status))
		})
	}
}
