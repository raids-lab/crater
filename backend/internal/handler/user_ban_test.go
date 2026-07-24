package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
)

func TestVisibleUserBanStatusReturnsEmptyStateForNeverBannedUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:visible_user_ban_empty?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserBanRecord{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{
		Name:            "zhouyh25",
		Nickname:        "周同学",
		Role:            model.RoleUser,
		Status:          model.StatusActive,
		Space:           "/space/zhouyh25",
		BanRestrictions: datatypes.NewJSONType(model.UserBanRestrictions{}),
		Attributes: datatypes.NewJSONType(model.UserAttribute{
			Name:     "zhouyh25",
			Nickname: "周同学",
		}),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	mgr := &UserMgr{userBanService: service.NewUserBanService(query.Use(db))}
	router := gin.New()
	mgr.RegisterProtected(router.Group("/v1/users"))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/zhouyh25/ban", http.NoBody)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET user ban status returned HTTP %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data VisibleUserBanStatusResp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Banned || response.Data.PermanentBanned {
		t.Fatalf("never-banned user returned active ban state: %+v", response.Data)
	}
	if response.Data.BannedTimestamp != nil || response.Data.BanRestrictions.Any() || response.Data.Reason != "" {
		t.Fatalf("never-banned user returned non-empty ban details: %+v", response.Data)
	}
	if response.Data.Records == nil || len(response.Data.Records) != 0 {
		t.Fatalf("never-banned user records = %#v, want non-nil empty list", response.Data.Records)
	}
}
