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

package handler

import (
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

// lockModelDownloadIdentity serializes resource variants whose database
// uniqueness still differs by source and revision, including cleanup history.
func lockModelDownloadIdentity(
	c *gin.Context, txQ *query.Query, name string, category model.DownloadCategory,
) error {
	db := txQ.ModelDownload.WithContext(c).UnderlyingDB()
	if db.Name() != "postgres" {
		return nil
	}
	identity := string(category) + ":" + name
	if err := db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", identity).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "lock logical download identity")
	}
	return nil
}

// A soft-deleted Ready record can be restored without downloading again. Mark
// affected history Failed before submitting a Job that may remove its files.
// Keep deleted_at intact, even if Job submission subsequently fails.
func invalidateSoftDeletedDownloadsForCleanup(
	c *gin.Context, db *gorm.DB, download *model.ModelDownload,
) error {
	return db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		txQ := query.Use(tx)
		if err := lockModelDownloadIdentity(c, txQ, download.Name, download.Category); err != nil {
			return err
		}
		q := txQ.ModelDownload
		history, err := q.WithContext(c).Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(q.ID.Neq(download.ID), q.Name.Eq(download.Name), q.Category.Eq(string(download.Category)),
				q.Status.Eq(string(model.ModelDownloadStatusReady)), q.DeletedAt.IsNotNull()).Find()
		if err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "find download history affected by storage cleanup")
		}
		var affectedIDs []uint
		for _, previous := range history {
			if downloadStoragePathsOverlap(download.Path, previous.Path) {
				affectedIDs = append(affectedIDs, previous.ID)
			}
		}
		if len(affectedIDs) == 0 {
			return nil
		}
		_, err = q.WithContext(c).Unscoped().Where(q.ID.In(affectedIDs...)).Updates(map[string]any{
			"status":  model.ModelDownloadStatusFailed,
			"message": "Stored files were invalidated by a new download attempt; download this resource again.",
		})
		if err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "invalidate download history before storage cleanup")
		}
		return nil
	})
}

// Cleanup also invalidates an ancestor resource if it removes a historical
// subdirectory; siblings and names sharing only a string prefix are unaffected.
func downloadStoragePathsOverlap(first, second string) bool {
	if first == "" || second == "" {
		return false
	}
	first = path.Clean(strings.ReplaceAll(first, `\`, "/"))
	second = path.Clean(strings.ReplaceAll(second, `\`, "/"))
	return first == second || strings.HasPrefix(first, second+"/") || strings.HasPrefix(second, first+"/")
}
