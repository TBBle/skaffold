/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package filemon

import (
	"context"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFileMonitor(t *testing.T) {
	var tests = []struct {
		description string
		update      func(folder *testutil.TempDir)
	}{
		{
			description: "file change",
			update: func(folder *testutil.TempDir) {
				folder.Chtimes("file", time.Now().Add(2*time.Second))
			},
		},
		{
			description: "file delete",
			update: func(folder *testutil.TempDir) {
				folder.Remove("file")
			},
		},
		{
			description: "file create",
			update: func(folder *testutil.TempDir) {
				folder.Write("new", "content")
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("file", "content")

			folderChanged := newCallback()

			// Watch folder
			monitor := NewMonitor()
			err := monitor.Register(tmpDir.List, folderChanged.call)
			t.CheckNoError(err)

			// Run the watcher
			ctx, cancel := context.WithCancel(context.Background())
			var stopped sync.WaitGroup
			stopped.Add(1)
			go func() {
				err = monitor.Run(ctx, ioutil.Discard, false)
				stopped.Done()
				t.CheckNoError(err)
			}()

			test.update(tmpDir)

			// Wait for the callbacks
			folderChanged.wait()
			cancel()
			stopped.Wait() // Make sure the watcher is stopped before deleting the tmp folder
		})
	}
}

type callback struct {
	wg *sync.WaitGroup
}

func newCallback() *callback {
	var wg sync.WaitGroup
	wg.Add(1)

	return &callback{
		wg: &wg,
	}
}

func (c *callback) call(e Events) {
	c.wg.Done()
}

func (c *callback) wait() {
	c.wg.Wait()
}