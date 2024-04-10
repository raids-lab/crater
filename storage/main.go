package main

import (
	"webdav/service"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	methods := []string{
		"OPTIONS",
		"HEAD",
		"DELETE", "GET",
		"PUT",
		"MKCOL",
		"LOCK",
		"UNLOCK",
		"PROPFIND",
		"PROPPATCH",
	}

	for _, m := range methods {
		r.Handle(m, "/files", service.WebDav)
		r.Handle(m, "/files/*path", service.WebDav)
	}
	r.Handle("POST", "/files/shareddir", service.GetSharedProjectDir)
	r.Handle("POST", "/files/mydir", service.GetMyDir)
	r.Handle("POST", "/files/sharedfile", service.GetFile)
	r.Handle("GET", "/testtoken", service.Testtoken)
	err := r.Run(":7320")
	if err != nil {
		log.Fatal(err)
	}
}
