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
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Handler struct {
	gw *GhostWriter
}

// Handles all HTTP requests.
func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	var (
		err  error
		path string
		info os.FileInfo
	)
	h.gw.log.Printf("Path: %q", r.URL.Path)
	path = filepath.Join(h.gw.args.dst, r.URL.Path)
	if info, err = h.gw.fs.Stat(path); err != nil {
		http.NotFound(w, r)
		return
	}
	if info.IsDir() {
		path = filepath.Join(path, "index.html")
		if info, err = h.gw.fs.Stat(path); err != nil {
			http.NotFound(w, r)
			return
		}
	}
	http.ServeFile(w, r, path)
}

// Wraps http requests in a closure so request handlers can access state.
func GetHandler(h *Handler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HandleRequest(w, r)
	}
}

// Serve the given GhostWriter config over HTTP.
func Serve(gw *GhostWriter) (err error) {
	var (
		server  *http.Server
		mux     *http.ServeMux
		handler *Handler
	)
	mux = http.NewServeMux()
	handler = &Handler{gw: gw}
	mux.HandleFunc("/", GetHandler(handler))
	server = &http.Server{
		Addr:           gw.args.addr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return server.ListenAndServe()
}
