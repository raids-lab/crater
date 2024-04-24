package main

import (
	"webdav/service"

	"webdav/logutils"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	methods := []string{
		"HEAD",
		"DELETE",
		"PUT",
		"MKCOL",
		"LOCK",
		"UNLOCK",
		"PROPFIND",
		"PROPPATCH",
	}

	for _, m := range methods {
		r.Handle(m, "/api/ss", service.WebDav)
		r.Handle(m, "/api/ss/*path", service.WebDav)
	}

	r.Handle("OPTIONS", "/api/ss", service.AlloweOption)
	r.Handle("OPTIONS", "/api/ss/*path", service.AlloweOption)
	r.Handle("GET", "/api/ss/mydir", service.GetMyDir)
	r.Handle("GET", "/api/ss/file", service.GetFile)
	r.Handle("GET", "/api/ss/file/*path", service.GetFile)
	r.Handle("GET", "/api/ss/download/*path", service.Download)
	r.Handle("POST", "/api/ss/checkspace", service.CheckFilesExist)
	err := r.Run(":7320")
	if err != nil {
		logutils.Log.Fatal(err)
	}
}
