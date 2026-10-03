package storage

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	RegisterQuotaRoutes(r)

	methods := []string{
		"PUT",
		"PROPFIND",
		"PROPPATCH",
	}

	for _, m := range methods {
		r.Handle(m, "/api/ss", WebDav)
		r.Handle(m, "/api/ss/*path", WebDav)
	}
	r.Handle("MKCOL", "/api/ss", CreateDirectory)
	r.Handle("MKCOL", "/api/ss/*path", CreateDirectory)

	webdavGroup := r.Group("api/ss", WebDAVMiddleware())
	RegisterDataset(webdavGroup)
	RegisterFile(webdavGroup)
	webdavGroup.DELETE("/files", RemoveFile)
	webdavGroup.DELETE("/files/*path", RemoveFile)
}
