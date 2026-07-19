package image

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCudaBaseImageWriteRoutesAreAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mgr := &ImagePackMgr{}
	mgr.RegisterProtected(router.Group("/api/v1/images"))
	mgr.RegisterAdmin(router.Group("/api/v1/admin/images"))

	routes := make(map[string]bool)
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	want := []string{
		http.MethodGet + " /api/v1/images/cudabaseimage",
		http.MethodPost + " /api/v1/admin/images/cudabaseimage",
		http.MethodDelete + " /api/v1/admin/images/cudabaseimage/:id",
	}
	for _, route := range want {
		if !routes[route] {
			t.Errorf("missing route %s", route)
		}
	}

	forbidden := []string{
		http.MethodPost + " /api/v1/images/cudabaseimage",
		http.MethodDelete + " /api/v1/images/cudabaseimage/:id",
	}
	for _, route := range forbidden {
		if routes[route] {
			t.Errorf("user route must not expose CUDA mutation: %s", route)
		}
	}
}
