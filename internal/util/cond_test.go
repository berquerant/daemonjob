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

package util_test

import (
	"sync"
	"testing"
	"time"

	"github.com/berquerant/daemonjob/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestQuorumExecutor(t *testing.T) {
	var (
		q                    = util.NewQuorumExecutor(2)
		executed, ack1, ack2 bool
		wg                   sync.WaitGroup
	)

	t.Log("WaitAndExecute")
	wg.Add(1)
	go q.WaitAndExecute(func() {
		executed = true
		wg.Done()
	})
	time.Sleep(500 * time.Millisecond)
	assert.False(t, executed, "should not be executed because we do not get quorum yet")

	t.Log("Ack1")
	wg.Go(func() {
		q.AckAndWait()
		ack1 = true
	})
	time.Sleep(500 * time.Millisecond)
	assert.False(t, ack1, "should wait for remaining ack")
	assert.False(t, executed, "should not be executed because we do not get quorum yet")

	t.Log("Ack2")
	wg.Go(func() {
		q.AckAndWait()
		ack2 = true
	})

	wg.Wait()
	assert.True(t, ack1, "should end waiting because we got quorum")
	assert.True(t, ack2, "should end waiting because we got quorum")
	assert.True(t, executed, "should be executed because we got quorum")
}
