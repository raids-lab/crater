package utils

import (
	"context"
	"time"

	"github.com/raids-lab/crater/dao/query"
)

func CheckUserEmail(c context.Context, userID uint) bool {
	u := query.User
	user, err := u.WithContext(c).
		Where(u.ID.Eq(userID)).
		First()
	if err != nil {
		return false
	}
	varified, _ := CheckEmailVerified(user.LastEmailVerifiedAt)
	return varified
}

func CheckEmailVerified(lastVerified *time.Time) (varified bool, last *time.Time) {
	// todo: emailValidityDays写到配置文件中
	emailValidityDays := 180

	if lastVerified == nil || lastVerified.IsZero() {
		return false, nil
	}

	curTime := GetLocalTime()
	return lastVerified.AddDate(0, 0, emailValidityDays).After(curTime), lastVerified
}
