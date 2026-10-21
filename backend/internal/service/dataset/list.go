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

package dataset

import (
	"context"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
)

// ListOptions contains the already validated filters for an accessible dataset query.
type ListOptions struct {
	AccessibleIDs []uint
	Offset        int
	Limit         int
	Search        string
	Owner         string
	Sort          string
	Type          model.DataType
	UserID        uint
}

// List canonicalizes permission-scoped resources before display filters and pagination.
// Sort expressions are selected by the handler's whitelist and always include ID
// as a tie-breaker so adjacent pages remain stable when timestamps are equal.
func List(ctx context.Context, options *ListOptions) ([]*model.Dataset, int64, error) {
	if len(options.AccessibleIDs) == 0 {
		return []*model.Dataset{}, 0, nil
	}

	datasetsQuery := CanonicalListScope(
		query.Dataset.WithContext(ctx).UnderlyingDB().Model(&model.Dataset{}).
			Where("id IN ?", options.AccessibleIDs),
	)

	if options.Type != "" {
		datasetsQuery = datasetsQuery.Where("type = ?", options.Type)
	}

	if options.Search != "" {
		pattern := util.ContainsPattern(options.Search)
		datasetsQuery = datasetsQuery.Where(
			"(LOWER(name) LIKE ? ESCAPE '\\' OR LOWER(describe) LIKE ? ESCAPE '\\')",
			pattern,
			pattern,
		)
	}

	switch options.Owner {
	case "mine":
		datasetsQuery = datasetsQuery.Where("user_id = ?", options.UserID)
	case "others":
		datasetsQuery = datasetsQuery.Where("user_id <> ?", options.UserID)
	}

	switch options.Sort {
	case "createdAt":
		datasetsQuery = datasetsQuery.Order("created_at ASC, id ASC")
	case "updatedAt":
		datasetsQuery = datasetsQuery.Order("updated_at ASC, id ASC")
	case "mountCount":
		datasetsQuery = datasetsQuery.Order("mount_count ASC, id ASC")
	case "-updatedAt":
		datasetsQuery = datasetsQuery.Order("updated_at DESC, id DESC")
	case "-mountCount":
		datasetsQuery = datasetsQuery.Order("mount_count DESC, id DESC")
	default:
		datasetsQuery = datasetsQuery.Order("created_at DESC, id DESC")
	}

	var total int64
	if err := datasetsQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*model.Dataset, 0)
	if err := datasetsQuery.Preload("User").Offset(options.Offset).Limit(options.Limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
