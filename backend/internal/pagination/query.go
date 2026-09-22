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
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/raids-lab/crater/internal/bizerr"
)

const (
	MaxSortFields   = 3
	MaxSearchRunes  = 128
	MaxFilterValues = 200
)

func ValidateSort(sortValue string, fields map[string]string) error {
	parts := strings.Split(sortValue, ",")
	if len(parts) > MaxSortFields {
		return bizerr.BadRequest.ParameterError.New("sort accepts at most 3 fields")
	}

	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		name := strings.TrimPrefix(part, "-")
		if _, ok := fields[name]; !ok {
			return bizerr.BadRequest.ParameterError.New(fmt.Sprintf("unsupported sort field %q", name))
		}
		if _, ok := seen[name]; ok {
			return bizerr.BadRequest.ParameterError.New(fmt.Sprintf("duplicate sort field %q", name))
		}
		seen[name] = struct{}{}
	}
	return nil
}

func SortClauses(sortValue string, fields map[string]string, stableClause string) []string {
	clauses := make([]string, 0, MaxSortFields+1)
	for _, part := range strings.Split(sortValue, ",") {
		name := strings.TrimPrefix(part, "-")
		direction := "ASC"
		if strings.HasPrefix(part, "-") {
			direction = "DESC"
		}
		clauses = append(clauses, fields[name]+" "+direction)
	}
	return append(clauses, stableClause)
}

func NormalizeSearch(search string) (string, error) {
	search = strings.TrimSpace(search)
	if utf8.RuneCountInString(search) > MaxSearchRunes {
		return "", bizerr.BadRequest.ParameterError.New("search accepts at most 128 characters")
	}
	return search, nil
}

func ValidatePageBounds(page, pageSize int) error {
	if page < 1 || pageSize < 1 {
		return bizerr.BadRequest.ParameterError.New("page and page_size must be positive")
	}
	if page > int(^uint(0)>>1)/pageSize {
		return bizerr.BadRequest.ParameterError.New("page is too large for page_size")
	}
	return nil
}

func NormalizePageQuery(search string, page, pageSize int) (string, error) {
	normalized, err := NormalizeSearch(search)
	if err != nil {
		return "", err
	}
	if err := ValidatePageBounds(page, pageSize); err != nil {
		return "", err
	}
	return normalized, nil
}

func DefaultAndValidateSort(sortValue, defaultSort string, fields map[string]string) (string, error) {
	if sortValue == "" {
		sortValue = defaultSort
	}
	if err := ValidateSort(sortValue, fields); err != nil {
		return "", err
	}
	return sortValue, nil
}

func PrepareBoundPageQuery(
	bindErr error,
	bindMessage string,
	search *string,
	page, pageSize int,
	sortValue *string,
	defaultSort string,
	fields map[string]string,
) error {
	if bindErr != nil {
		return bizerr.BadRequest.ParameterError.Wrap(bindErr, bindMessage)
	}

	normalized, err := NormalizePageQuery(*search, page, pageSize)
	if err != nil {
		return err
	}
	validatedSort, err := DefaultAndValidateSort(*sortValue, defaultSort, fields)
	if err != nil {
		return err
	}
	*search = normalized
	*sortValue = validatedSort
	return nil
}

func ParsePositiveUintValues(rawValues []string, key string) ([]uint, error) {
	if len(rawValues) > MaxFilterValues {
		return nil, bizerr.BadRequest.ParameterError.New(fmt.Sprintf("%s accepts at most 200 values", key))
	}

	values := make([]uint, 0, len(rawValues))
	seen := make(map[uint]struct{}, len(rawValues))
	for _, raw := range rawValues {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || value == 0 {
			return nil, bizerr.BadRequest.ParameterError.New(fmt.Sprintf("invalid %s %q", key, raw))
		}
		converted := uint(value)
		if _, ok := seen[converted]; ok {
			continue
		}
		seen[converted] = struct{}{}
		values = append(values, converted)
	}
	return values, nil
}

func ValidateAllowedValues(values []string, allowed map[string]struct{}, label string) error {
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return bizerr.BadRequest.ParameterError.New(fmt.Sprintf("unsupported %s %q", label, value))
		}
	}
	return nil
}
