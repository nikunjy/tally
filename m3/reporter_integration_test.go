// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package m3

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
)

// TestIntegrationProcessFlushOnExit tests whether data is correctly flushed
// when the scope is closed for shortly lived programs
func TestIntegrationProcessFlushOnExit(t *testing.T) {
	for i := 0; i < 5; i++ {
		testProcessFlushOnExit(t, i)
	}
}

func testProcessFlushOnExit(t *testing.T, i int) {
	dir, err := ioutil.TempDir("", "foo")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var wg sync.WaitGroup
	server := newFakeM3Server(t, &wg, true, Compact)
	wg.Add(1)
	go server.Serve()
	defer server.Close()

	r, err := NewReporter(Options{
		HostPorts: []string{server.Addr},
		Service:   "test-service",
		Env:       "test",
	})
	require.NoError(t, err)

	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		CachedReporter: r,
	}, 5*time.Second)

	scope.Counter("my-counter").Inc(42)
	scope.Gauge("my-gauge").Update(123)
	scope.Timer("my-timer").Record(456 * time.Millisecond)

	closer.Close()
	wg.Wait()
	require.Equal(t, 1, len(server.Service.getBatches()))
	require.NotNil(t, server.Service.getBatches()[0])
	require.Equal(t, 3, len(server.Service.getBatches()[0].GetMetrics()))
	metrics := server.Service.getBatches()[0].GetMetrics()
	fmt.Printf("Test %d emitted:\n%v\n", i, metrics)
}
