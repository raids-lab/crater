package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/model"
	"github.com/raids-lab/crater/pkg/query"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const MySQLDSN = "root:buaak8sportal@2023mysql@tcp(***REMOVED***:30306)/crater?charset=utf8mb4&parseTime=True"

func ConnectDB(dsn string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect db fail: %w", err))
	}
	return db
}

func main() {
	db := ConnectDB(MySQLDSN)
	query.SetDefault(db)

	name := "huangjx"

	u := query.User
	_, err := u.WithContext(context.Background()).Where(u.Name.Eq(name)).First()
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		logutils.Log.Error("user already exists")
		os.Exit(1)
	}

	// create user, project named by user, user_project
	user := model.User{
		Name:      name,
		Nickname:  nil,
		Password:  nil,
		Role:      "admin",
		NameSpace: "crater-jobs",
		Status:    "active",
	}
	err = u.WithContext(context.Background()).Create(&user)
	if err != nil {
		logutils.Log.Error(err)
		os.Exit(1)
	}
	project := model.Project{
		Name:        name,
		Description: "project for " + name,
		NameSpace:   "crater-jobs",
		Status:      "active",
		Quota:       "{}",
	}
	p := query.Project
	err = p.WithContext(context.Background()).Create(&project)
	if err != nil {
		logutils.Log.Error(err)
		os.Exit(1)
	}
	userProject := model.UserProject{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "admin",
		Quota:     "{}",
	}
	up := query.UserProject
	err = up.WithContext(context.Background()).Create(&userProject)
	if err != nil {
		logutils.Log.Error(err)
		os.Exit(1)
	}
}
