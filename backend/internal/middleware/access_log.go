// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// FormatAccessLog keeps credentials in URL query parameters out of HTTP access logs.
//
//nolint:gocritic // Gin's LogFormatter receives its parameters by value.
func FormatAccessLog(param gin.LogFormatterParams) string {
	var statusColor, methodColor, resetColor string
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}
	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}
	return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %q\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		accessLogPath(param.Request.URL),
		param.ErrorMessage,
	)
}

func accessLogPath(uri *url.URL) string {
	raw := uri.RawQuery
	if raw == "" {
		return uri.Path
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return uri.Path
	}
	redacted := false
	for key := range values {
		if strings.EqualFold(key, "token") {
			values.Set(key, "REDACTED")
			redacted = true
		}
	}
	if redacted {
		raw = values.Encode()
	}
	return uri.Path + "?" + raw
}
