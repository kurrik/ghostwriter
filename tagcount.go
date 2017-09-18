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

// Represents a tag and the number of posts with that tag.
type TagCount struct {
	Tag   string
	Count int
}

// A list of TagCounts
type TagCounts []*TagCount

// Compares two posts.
func (tc TagCounts) Less(i int, j int) bool {
	ci := tc[i].Count
	cj := tc[j].Count
	return ci > cj
}

// Returns the length of a set of tag counts.
func (tc TagCounts) Len() int {
	return len(tc)
}

// Swaps two tag counts in the given positions.
func (tc TagCounts) Swap(i int, j int) {
	tc[i], tc[j] = tc[j], tc[i]
}
