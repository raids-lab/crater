package main

import (
	"fmt"
	"os"
	"webdav/dao/query"
	"webdav/logutils"
	"webdav/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	r := gin.Default()
	err := query.InitDB()
	if err != nil {
		fmt.Println("err init:", err)
		os.Exit(1)
	}

	query.SetDefault(query.DB)
	if gin.Mode() == gin.DebugMode {
		err = godotenv.Load(".env")
		if err != nil {
			logutils.Log.Info("can't load env,err:", err)
		}
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "7320" // 默认端口
	}

	go service.StartCheckSpace()
	methods := []string{
		"PUT",
		"MKCOL",
		"PROPFIND",
		"PROPPATCH",
	}

	for _, m := range methods {
		r.Handle(m, "/api/ss", service.WebDav)
		r.Handle(m, "/api/ss/*path", service.WebDav)
	}
	webdavGroup := r.Group("api/ss", service.WebDAVMiddleware())
	service.RegisterDataset(webdavGroup)
	service.RegisterFile(webdavGroup)

	err = r.Run(":" + port)
	if err != nil {
		logutils.Log.Fatal(err)
	}
}
