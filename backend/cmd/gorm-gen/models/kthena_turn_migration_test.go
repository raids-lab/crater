package main

import (
	"testing"

	"github.com/raids-lab/crater/dao/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestKthenaTurnLeaseMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"CREATE TABLE kthena_chat_sessions (id integer primary key)", "CREATE TABLE kthena_chat_messages (id integer primary key)"} {
		if err = db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	migration := kthenaTurnLeaseMigration()
	for range 2 {
		if err = migration.Migrate(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, field := range []string{"TurnLease", "TurnExpiresAt"} {
		if !db.Migrator().HasColumn(&model.KthenaChatSession{}, field) {
			t.Fatalf("missing %s", field)
		}
	}
	if !db.Migrator().HasColumn(&model.KthenaChatMessage{}, "RequestHash") {
		t.Fatal("missing request hash")
	}
	for range 2 {
		if err = migration.Rollback(db); err != nil {
			t.Fatal(err)
		}
	}
	if db.Migrator().HasColumn(&model.KthenaChatSession{}, "TurnLease") {
		t.Fatal("lease remains after rollback")
	}
}
