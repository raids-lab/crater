package main

import (
	"fmt"
	"os"
	"webdav/dao/query"
	"webdav/service"

	"webdav/logutils"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	err := query.InitDB()
	if err != nil {
		fmt.Println("err init:", err)
		os.Exit(1)
	}

	query.SetDefault(query.DB)

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
	err = r.Run(":7320")
	if err != nil {
		logutils.Log.Fatal(err)
	}
}
