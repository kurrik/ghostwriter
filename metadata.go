// Copyright 2017 Arne Roomann-Kurrik
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

// Serializable models used to configure sites, posts, etc.

type SiteMeta struct {
	Title       string
	Root        string
	Author      string
	Email       string
	PathFormat  string
	DateFormat  string
	TagsFormat  string
	RecentCount int
	Metadata    map[string]string
}

type PostMeta struct {
	Tags     []string
	Title    string
	Date     string
	Slug     string
	Scripts  []string
	Styles   []string
	Images   map[string]ImageMeta
	Metadata map[string]string
}

type ImageVariantMeta struct {
	Src    *string
}

type ImageMeta struct {
	Src      string
	Variants map[string]ImageVariantMeta
	Metadata map[string]string
}
