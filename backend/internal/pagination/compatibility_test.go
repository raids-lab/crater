// Copyright 2026 The Crater Project Team, RAIDS-Lab
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

package pagination

import (
	"net/url"
	"testing"
)

func TestLegacyListPaginationIsExplicit(t *testing.T) {
	for _, test := range []struct {
		query string
		paged bool
	}{
		{"", false},
		{"search=alice&category=model&sort=name", false},
		{"page=1", true},
		{"page_size=20", true},
		{"page=", true},
		{"page_size=invalid", true},
	} {
		t.Run(test.query, func(t *testing.T) {
			values, err := url.ParseQuery(test.query)
			if err != nil {
				t.Fatal(err)
			}
			if Requested(values) != test.paged {
				t.Fatalf("pagination presence changed for %q", test.query)
			}
			offset, limit := Bounds(values, 2, 20)
			if test.paged && (offset != 20 || limit != 20) {
				t.Fatalf("paged bounds = %d, %d", offset, limit)
			}
			if !test.paged && (offset != 0 || limit != -1) {
				t.Fatalf("complete-list bounds must not truncate: %d, %d", offset, limit)
			}
		})
	}
}

func TestLegacyPageSizeAliasesAreValidated(t *testing.T) {
	for _, test := range []struct {
		query string
		alias string
		max   int
		want  int
		bad   bool
	}{
		{"pageSize=25", "pageSize", 200, 25, false},
		{"page_size=25&pageSize=25", "pageSize", 200, 25, false},
		{"page_size=25&pageSize=26", "pageSize", 200, 0, true},
		{"pageSize=", "pageSize", 200, 0, true},
		{"pageSize=0", "pageSize", 200, 0, true},
		{"pageSize=201", "pageSize", 200, 0, true},
		{"pageSize=10&pageSize=20", "pageSize", 200, 0, true},
		{"limit=500", "limit", 0, 500, false},
	} {
		t.Run(test.query, func(t *testing.T) {
			values, err := url.ParseQuery(test.query)
			if err != nil {
				t.Fatal(err)
			}
			got, err := PageSizeAlias(values, 25, test.alias, test.max)
			if (err != nil) != test.bad || (!test.bad && got != test.want) {
				t.Fatalf("alias returned %d, %v; want %d, bad=%v", got, err, test.want, test.bad)
			}
		})
	}
}
