package main

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

func servingUsageMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID:       "202610041200",
		Migrate:  func(tx *gorm.DB) error { return createTableIfMissing(tx, &model.ServingUsage{}) },
		Rollback: func(tx *gorm.DB) error { return dropTableIfPresent(tx, &model.ServingUsage{}) },
	}
}
