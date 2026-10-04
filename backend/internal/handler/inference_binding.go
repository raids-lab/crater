package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/bizerr"
)

// Deployment requests have one contract. Reject obsolete worker fields and
// unknown parameters instead of silently dropping a caller's configuration.
func bindServingRequest(c *gin.Context, req *CreateKthenaReq) error {
	const maxServingRequestBytes = 1 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxServingRequestBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(req); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return bizerr.BadRequest.InvalidRequest.New("expected one JSON deployment")
	}
	return nil
}
