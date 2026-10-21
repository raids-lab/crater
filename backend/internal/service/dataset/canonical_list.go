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
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

// Match strings.TrimSpace when comparing historical downloaded-resource names.
const resourceNameWhitespace = "\t\n\v\f\r \u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005" +
	"\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000"

// CanonicalListScope keeps the lowest visible ID for each ready downloaded
// resource. Names without a matching ready download and different types stay separate.
// Call it on a permission-scoped Dataset query before adding display filters,
// ordering, Count, or Offset/Limit. Subqueries inherit the caller's database and
// context, and never include a lower-ID row outside that permission scope.
func CanonicalListScope(db *gorm.DB) *gorm.DB {
	visibleIDs := db.Session(&gorm.Session{}).Select("datasets.id")
	visibleNames := db.Session(&gorm.Session{}).Select("datasets.name").
		Where("datasets.type IN ?", []model.DataType{model.DataTypeDataset, model.DataTypeModel})

	readyDownload := db.Session(&gorm.Session{NewDB: true}).Model(&model.ModelDownload{}).
		Select("1").
		Where("model_downloads.status = ?", model.ModelDownloadStatusReady).
		Where("model_downloads.category = datasets.type").
		Where("model_downloads.name IN (?)", visibleNames).
		Where("LOWER(TRIM(model_downloads.name, ?)) = LOWER(TRIM(datasets.name, ?))",
			resourceNameWhitespace, resourceNameWhitespace)

	lowerVisibleID := db.Session(&gorm.Session{NewDB: true}).
		Model(&model.Dataset{}).Table("datasets AS canonical_datasets").Select("1").
		Where("canonical_datasets.id IN (?)", visibleIDs).
		Where("canonical_datasets.id < datasets.id AND canonical_datasets.type = datasets.type").
		Where("LOWER(TRIM(canonical_datasets.name, ?)) = LOWER(TRIM(datasets.name, ?))",
			resourceNameWhitespace, resourceNameWhitespace)

	return db.Session(&gorm.Session{}).Where("NOT EXISTS (?) OR NOT EXISTS (?)", readyDownload, lowerVisibleID)
}
