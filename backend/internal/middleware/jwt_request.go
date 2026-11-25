// Copyright 2026 The Crater Project Team, RAIDS-Lab

package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

// protectedRequestToken supports existing browser query tokens and native headers.
func protectedRequestToken(c *gin.Context) (string, error) {
	header := c.Request.Header.Get("Authorization")
	if strings.EqualFold(c.Request.Header.Get("Upgrade"), "websocket") && header == "" {
		return c.Query("token"), nil
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid bearer authorization")
	}
	return parts[1], nil
}
