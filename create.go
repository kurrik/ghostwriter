// Copyright 2013 Arne Roomann-Kurrik
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
	"bufio"
	"fmt"
	"github.com/kurrik/fauxfile"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const TMPL_BODY_MD = `This is the post snippet.

<!--BREAK-->

This is content after the break.`

const TMPL_META_YAML = `date: %s
slug: %s
title: %s
tags:
%s`

// Writes content to a new file at path dst.
func writeFile(gw *GhostWriter, content string, dst string) (err error) {
	var (
		fdst fauxfile.File
	)
	if fdst, err = gw.fs.Create(dst); err != nil {
		return
	}
	defer fdst.Close()
	_, err = fdst.WriteString(content)
	return
}

// Prints the list of parseable posts given the GhostWriter config.
func printPosts(gw *GhostWriter) (err error) {
	var (
		name   = gw.args.posts
		src    = filepath.Join(gw.args.src, name)
		names  []string
		msrc   string
		id     string
		oldlog *log.Logger
	)
	// Supress logging for this function.
	oldlog = gw.log
	gw.log = log.New(ioutil.Discard, "", log.LstdFlags)

	if names, err = gw.readDir(src); err != nil {
		err = fmt.Errorf("Posts directory not found: %v", err)
		gw.log = oldlog
		return
	}
	for _, id = range names {
		if !gw.isDir(filepath.Join(src, id)) {
			continue
		}
		msrc = filepath.Join(name, id, "meta.yaml")
		if _, err = gw.parsePostMeta(msrc); err != nil {
			// Not a post, but don't raise an error.
			continue
		}
		fmt.Printf("  %v\n", id)
	}
	gw.log = oldlog
	return
}

// Create a new post given the GhostWriter configuration.
func Create(gw *GhostWriter) (err error) {
	var (
		path    string
		slug    string
		title   string
		tag     string
		tags    []string
		date    string
		tagfmt  string
		dst     string
		content string
		reader  *bufio.Reader
	)
	if err = gw.parseSiteMeta(); err != nil {
		return
	}
	fmt.Printf("Existing posts:\n")
	fmt.Printf("---------------\n")
	if err = printPosts(gw); err != nil {
		return
	}
	fmt.Println()
	fmt.Printf("Enter the directory name for the new post: ")
	if _, err = fmt.Fscanf(os.Stdin, "%s", &path); err != nil {
		return
	}
	fmt.Printf("Enter the url slug for the post: ")
	if _, err = fmt.Fscanf(os.Stdin, "%s", &slug); err != nil {
		return
	}
	fmt.Printf("Enter the title for the post: ")
	reader = bufio.NewReader(os.Stdin)
	if title, err = reader.ReadString('\n'); err != nil {
		return
	}
	title = strings.TrimSpace(title)
	fmt.Printf("Enter the tags for the post, space separated: ")
	if tag, err = reader.ReadString('\n'); err != nil {
		return
	}
	tags = strings.Split(tag, " ")
	for _, tag = range tags {
		tagfmt += fmt.Sprintf("  - %s\n", tag)
	}
	date = time.Now().Format(gw.site.meta.DateFormat)
	fmt.Printf("Using date: %s", date)
	fmt.Println()

	// Write the actual files.
	dst = filepath.Join(gw.args.src, gw.args.posts, path)
	if err = gw.fs.MkdirAll(dst, 0755); err != nil {
		return
	}
	dst = filepath.Join(gw.args.src, gw.args.posts, path, "meta.yaml")
	content = fmt.Sprintf(TMPL_META_YAML, date, slug, title, tagfmt)
	if err = writeFile(gw, content, dst); err != nil {
		return
	}
	dst = filepath.Join(gw.args.src, gw.args.posts, path, "body.md")
	if err = writeFile(gw, TMPL_BODY_MD, dst); err != nil {
		return
	}

	fmt.Printf("Done.\n")
	return
}
