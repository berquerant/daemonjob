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
	"sync"
)

func NewQuorumExecutor(maxAck int) *QuorumExecutor {
	return &QuorumExecutor{
		cond: &sync.Cond{
			L: new(sync.Mutex),
		},
		ack:    0,
		maxAck: maxAck,
	}
}

type QuorumExecutor struct {
	cond   *sync.Cond
	ack    int
	maxAck int
}

// AckAndWait increments the ack count and wait for maxAck.
func (c *QuorumExecutor) AckAndWait() {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	c.ack++
	c.cond.Broadcast()
	for c.ack > 0 {
		c.cond.Wait()
	}
}

// WaitAndExecute waits for maxAck and then executes f.
func (c *QuorumExecutor) WaitAndExecute(f func()) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	for c.ack < c.maxAck {
		c.cond.Wait()
	}
	f()
	c.ack = 0
	c.cond.Broadcast()
}
