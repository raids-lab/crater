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
	"errors"
	"reflect"
	"strings"
	"testing"
)

var testSortFields = map[string]string{
	"name":      "name",
	"createdAt": "created_at",
}

func TestPrepareBoundPageQuery(t *testing.T) {
	search := "  template  "
	sortValue := ""
	if err := PrepareBoundPageQuery(nil, "bind query", &search, 2, 20, &sortValue, "-createdAt", testSortFields); err != nil {
		t.Fatalf("PrepareBoundPageQuery returned error: %v", err)
	}
	if search != "template" || sortValue != "-createdAt" {
		t.Fatalf("unexpected normalized query: search=%q sort=%q", search, sortValue)
	}
}

func TestPrepareBoundPageQueryWrapsBindError(t *testing.T) {
	search, sortValue := "", ""
	if err := PrepareBoundPageQuery(errors.New("bad query"), "bind query", &search, 1, 10, &sortValue, "name", testSortFields); err == nil {
		t.Fatal("expected a bind error")
	}
}

func TestNormalizePageQueryValidation(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	testCases := []struct {
		name     string
		search   string
		page     int
		pageSize int
	}{
		{name: "long search", search: strings.Repeat("界", MaxSearchRunes+1), page: 1, pageSize: 10},
		{name: "page", page: 0, pageSize: 10},
		{name: "page size", page: 1, pageSize: 0},
		{name: "offset overflow", page: maxInt, pageSize: 2},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := NormalizePageQuery(testCase.search, testCase.page, testCase.pageSize); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

func TestSortValidationAndClauses(t *testing.T) {
	if err := ValidateSort("-createdAt,name", testSortFields); err != nil {
		t.Fatalf("ValidateSort returned error: %v", err)
	}
	want := []string{"created_at DESC", "name ASC", "id ASC"}
	if got := SortClauses("-createdAt,name", testSortFields, "id ASC"); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected clauses: got %v, want %v", got, want)
	}
	for _, sortValue := range []string{"unknown", "name,-name", "name,createdAt,-name,createdAt"} {
		if err := ValidateSort(sortValue, testSortFields); err == nil {
			t.Fatalf("expected %q to be rejected", sortValue)
		}
	}
}

func TestParsePositiveUintValues(t *testing.T) {
	got, err := ParsePositiveUintValues([]string{"3", "1", "3"}, "id")
	if err != nil {
		t.Fatalf("ParsePositiveUintValues returned error: %v", err)
	}
	if want := []uint{3, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected values: got %v, want %v", got, want)
	}
	for _, values := range [][]string{{"0"}, {"x"}, make([]string, MaxFilterValues+1)} {
		if _, err := ParsePositiveUintValues(values, "id"); err == nil {
			t.Fatalf("expected values to be rejected (count=%d)", len(values))
		}
	}
}

func TestValidateAllowedValues(t *testing.T) {
	allowed := map[string]struct{}{"pending": {}, "running": {}}
	if err := ValidateAllowedValues([]string{"pending", "running"}, allowed, "status"); err != nil {
		t.Fatalf("ValidateAllowedValues returned error: %v", err)
	}
	if err := ValidateAllowedValues([]string{"unknown"}, allowed, "status"); err == nil {
		t.Fatal("expected an unsupported value error")
	}
}
