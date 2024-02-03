// SPDX-License-Identifier: MIT
//
// Copyright (c) 2023 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package queues_test

import (
	"sync"
	"testing"
	"time"

	"github.com/itsdevbear/bolaris/async/dispatch/queues"
)

func TestSerialDispatchQueue(t *testing.T) {
	q := queues.NewSerialDispatchQueue()

	// Test Async
	wg := &sync.WaitGroup{}
	wg.Add(1)
	q.Async(func() {
		wg.Done()
	})
	wg.Wait()

	// Test AsyncAfter
	wg.Add(1)
	q.AsyncAfter(time.Millisecond*100, func() {
		wg.Done()
	})
	wg.Wait()

	// Test Sync
	syncDone := false
	q.Sync(func() {
		syncDone = true
	})
	if !syncDone {
		t.Errorf("Sync function did not execute")
	}

	// Test AsyncAndWait
	asyncAndWaitDone := false
	q.AsyncAndWait(func() {
		asyncAndWaitDone = true
	})
	if !asyncAndWaitDone {
		t.Errorf("AsyncAndWait function did not execute")
	}

	// Test Stop
	q.Stop()
}

func TestSerialDispatchQueue_Stop(t *testing.T) {
	q := queues.NewSerialDispatchQueue()

	// Add some items to the queue
	for i := 0; i < 10; i++ {
		q.Async(func() {
			time.Sleep(time.Millisecond * 100)
		})
	}

	// Stop the queue
	q.Stop()

	// Try to add another item to the queue, it should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Async function did not panic after Stop")
		}
	}()

	q.Async(func() {
		// This code should never be executed
		t.Errorf("Async function executed after Stop")
	})

	q.Stop()
}
