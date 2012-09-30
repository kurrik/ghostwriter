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
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"github.com/kurrik/fauxfile"
	"os"
	"path/filepath"
	"time"
)

// Arguments, passed to the main executable.
type Args struct {
	src          string
	dst          string
	posts        string
	templates    string
	static       string
	config       string
	postTemplate string
	tagsTemplate string
}

// Sensible defaults, for a sensible time.
func DefaultArgs() *Args {
	return &Args{
		src:          "src",
		dst:          "dst",
		posts:        "posts",
		templates:    "templates",
		static:       "static",
		config:       "config.yaml",
		postTemplate: "post.tmpl",
		tagsTemplate: "tags.tmpl",
	}
}

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
	for len(queue) > 0 {
		path = queue[0]
		src = filepath.Join(w.root, path)
		queue = queue[1:]
		if info, err = w.gw.fs.Stat(src); err != nil {
			return
		}
		if info.IsDir() {
			if filenames, err = w.gw.readDir(src); err != nil {
				return
			}
			for i, filename = range filenames {
				filenames[i] = filepath.Join(path, filename)
			}
			queue = append(queue, filenames...)
			w.gw.log.Printf("Watching %v\n", src)
			if err = w.watcher.Watch(src); err != nil {
				return
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
	go watcher.Handle(work, errors)
	if err = watcher.WatchDirs(); err != nil {
		return
	}
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

// Main routine.
func main() {
	var (
		watch bool
		gw    *GhostWriter
		err   error
	)
	a := DefaultArgs()
	flag.StringVar(&a.src, "src", "src", "Path to src files.")
	flag.StringVar(&a.dst, "dst", "dst", "Build output directory.")
	flag.BoolVar(&watch, "watch", false, "Keep watching the source dir?")
	flag.Parse()
	gw = NewGhostWriter(&fauxfile.RealFilesystem{}, a)
	if watch {
		err = Watch(gw, a.src)
	} else {
		err = gw.Process()
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
