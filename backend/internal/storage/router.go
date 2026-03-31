package storage

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	methods := []string{
		"PUT",
		"MKCOL",
		"PROPFIND",
		"PROPPATCH",
	}

	for _, m := range methods {
		r.Handle(m, "/api/ss", WebDav)
		r.Handle(m, "/api/ss/*path", WebDav)
	}

	webdavGroup := r.Group("api/ss", WebDAVMiddleware())
	RegisterDataset(webdavGroup)
	RegisterFile(webdavGroup)
}
