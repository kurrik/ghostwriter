// Copyright 2012 Arne Roomann-Kurrik
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

package main

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
	"time"
)

// State for fs notify wrapper.
type Watcher struct {
	watcher *fsnotify.Watcher
	root    string
	gw      *GhostWriter
}

// Listen for FS events and signal work when something changes.
// Will send errors over e.
func (w *Watcher) Handle(work chan bool, e chan error) {
	var (
		evt *fsnotify.FileEvent
		err error
	)
	for {
		select {
		case evt = <-w.watcher.Event:
			w.gw.log.Printf("Filesystem changed: %v\n", evt.String())
			isNewDir := evt.IsCreate() && w.gw.isDir(evt.Name)
			if isNewDir || evt.IsDelete() || evt.IsRename() {
				err = w.WatchDirs()
				if err != nil {
					e <- err
					return
				}
			}
			select {
			case work <- true:
				// gw.log.Printf("Work queued\n")
				// Queued work.
			default:
				// gw.log.Printf("Work queue full\n")
				// Work queue full, no worries.
			}
		case err = <-w.watcher.Error:
			e <- err
			return
		}
	}
}

// Sets up filesystem notices for all directories under root, inclusive.
// Can be called multiple times, initializes watcher object each time.
func (w *Watcher) WatchDirs() (err error) {
	var (
		path      string
		i         int
		queue     []string
		src       string
		info      os.FileInfo
		filenames []string
		filename  string
		errors    int
	)
	if w.watcher != nil {
		w.watcher.Close()
	}
	if w.watcher, err = fsnotify.NewWatcher(); err != nil {
		return
	}
	if queue, err = w.gw.readDir(w.root); err != nil {
		return
	}
	w.gw.log.Printf("Watching %v\n", w.root)
	if err = w.watcher.Watch(w.root); err != nil {
		return
	}
	errors = 0
	for len(queue) > 0 {
		path = queue[0]
		src = filepath.Join(w.root, path)
		queue = queue[1:]
		if info, err = w.gw.fs.Stat(src); err != nil {
			return
		}
		if info.IsDir() {
			if filenames, err = w.gw.readDir(src); err != nil {
				queue = append(queue, src)
				w.gw.log.Printf("Error: %v, retrying later\n", err)
				err = nil
				errors += 1
				if errors > 10 {
					err = fmt.Errorf("Too many errors experienced, quitting")
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
			for i, filename = range filenames {
				filenames[i] = filepath.Join(path, filename)
			}
			queue = append(queue, filenames...)
			w.gw.log.Printf("Watching %v\n", src)
			if err = w.watcher.Watch(src); err != nil {
				queue = append(queue, src)
				w.gw.log.Printf("Error: %v, retrying later\n", err)
				err = nil
				errors += 1
				if errors > 10 {
					err = fmt.Errorf("Too many errors experienced, quitting")
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	return
}

// Watches the filesystem for changes and runs gw.Process in response.
func Watch(gw *GhostWriter, root string) (err error) {
	var (
		working bool = true
		timer   *time.Timer
		watcher *Watcher
	)
	var (
		errors = make(chan error, 1)
		work   = make(chan bool, 1)
	)
	watcher = &Watcher{
		root: root,
		gw:   gw,
	}
	if err = watcher.WatchDirs(); err != nil {
		return
	}
	go watcher.Handle(work, errors)
	work <- true // Enqueue one render for startup.
	for working {
		select {
		case <-work:
			// Queue a render in the future.  This is because
			// many notifications are sent for individual changes
			// and rendering in response to each would be a waste.
			if timer != nil {
				timer.Stop()
				timer = nil
			}
			timer = time.AfterFunc(200*time.Millisecond, func() {
				gw.log.Printf("Processing site:\n")
				if err := gw.Process(); err != nil {
					errors <- err
				}
			})
		case err = <-errors:
			working = false
		}
	}
	return
}
