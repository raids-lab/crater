package utils

import (
	"time"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
)

func GetLocalTime() time.Time {
	timeZone := config.GetConfig().Postgres.TimeZone
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		logutils.Log.Errorf("Failed to load location: %v", err)
		return time.Now()
	}
	return time.Now().In(loc)
}

func GetPermanentTime() time.Time {
	timeZone := config.GetConfig().Postgres.TimeZone
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		logutils.Log.Errorf("Failed to load location: %v", err)
		return time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(9999, 12, 31, 0, 0, 0, 0, loc)
}

func IsPermanentTime(t time.Time) bool {
	return t.Equal(GetPermanentTime())
}
