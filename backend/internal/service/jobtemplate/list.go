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

package jobtemplate

import (
	"context"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
)

// ListOptions contains the validated filters for a paginated template query.
type ListOptions struct {
	Offset int
	Limit  int
	Search string
	Owner  string
	Sort   string
	UserID uint
}

// List applies business filters before returning one stably sorted page.
func List(ctx context.Context, options ListOptions) ([]*model.Jobtemplate, int64, error) {
	j := query.Jobtemplate
	templatesQuery := j.WithContext(ctx).Preload(j.User).Where(j.ID.IsNotNull())

	if options.Search != "" {
		templatesQuery = templatesQuery.Where(j.Name.Lower().Like(util.ContainsPattern(options.Search)))
	}

	switch options.Owner {
	case "mine":
		templatesQuery = templatesQuery.Where(j.UserID.Eq(options.UserID))
	case "others":
		templatesQuery = templatesQuery.Where(j.UserID.Neq(options.UserID))
	}

	if options.Sort == "createdAt" {
		templatesQuery = templatesQuery.Order(j.CreatedAt.Asc(), j.ID.Asc())
	} else {
		templatesQuery = templatesQuery.Order(j.CreatedAt.Desc(), j.ID.Desc())
	}

	return templatesQuery.FindByPage(options.Offset, options.Limit)
}
