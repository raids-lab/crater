package main

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

func TestServingUsageMigrationLifecycle(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	migration := servingUsageMigration()
	for range 2 {
		if err := migration.Migrate(db); err != nil {
			t.Fatal(err)
		}
	}
	if !db.Migrator().HasTable(&model.ServingUsage{}) {
		t.Fatal("meter table missing")
	}
	for range 2 {
		if err := migration.Rollback(db); err != nil {
			t.Fatal(err)
		}
	}
	if db.Migrator().HasTable(&model.ServingUsage{}) {
		t.Fatal("meter table not removed")
	}
}
