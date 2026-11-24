package handler

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
)

// Opt-in integration test against the disposable local PostgreSQL instance only.
func newLocalProfilePostgres(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("CRATER_TEST_LOCAL_POSTGRES") != "1" {
		t.Skip("local PostgreSQL integration is opt-in")
	}
	db, err := gorm.Open(postgres.Open("host=127.0.0.1 port=55463 user=crater_test dbname=postgres sslmode=disable"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	schema := fmt.Sprintf("review583_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	old := query.User
	q := query.Use(tx)
	query.User = &q.User
	t.Cleanup(func() { query.User = old })
	return tx
}

func TestUserFieldsPostgresAtomicMerge(t *testing.T) {
	tx := newLocalProfilePostgres(t)
	user := model.User{Name: "synthetic-review", Space: "synthetic-review", Nickname: "old", Attributes: datatypes.NewJSONType(model.UserAttribute{Phone: ptr.To("old"), Identity: ptr.To("intern")})}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.PatchUserProfile(t.Context(), user.Name, map[string]any{"phone": "new"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := service.PatchUserProfile(t.Context(), user.Name, map[string]any{"identity": "PhD", "nickname": "canonical"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	data := user.Attributes.Data()
	if user.Nickname != "canonical" || data.Nickname != "canonical" || *data.Phone != "new" || *data.Identity != "PhD" {
		t.Fatal("PostgreSQL merge lost fields")
	}
	if err := service.PatchUserProfile(t.Context(), user.Name, map[string]any{"phone": nil}, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.Attributes.Data().Phone != nil || *user.Attributes.Data().Identity != "PhD" {
		t.Fatal("PostgreSQL clearing lost other keys")
	}
}

func TestUserFieldsPostgresVerifiedEmailProtection(t *testing.T) {
	tx := newLocalProfilePostgres(t)
	user := model.User{Name: "verified-email", Space: "verified-email", Attributes: datatypes.NewJSONType(model.UserAttribute{Email: ptr.To("old@example.invalid")})}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.PatchUserProfile(t.Context(), user.Name, map[string]any{"email": "verified@example.invalid"}, map[string]any{"last_email_verified_at": time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := service.PatchLDAPUserProfile(t.Context(), user.Name, map[string]any{"email": "directory@example.invalid", "phone": "new"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if *user.Attributes.Data().Email != "verified@example.invalid" || *user.Attributes.Data().Phone != "new" {
		t.Fatal("LDAP failed to preserve concurrently verified email")
	}
}
