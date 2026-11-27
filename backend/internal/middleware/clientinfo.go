// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/util"
)

// ClientIdentity parses the self-declared client type into the request context.
// It never rejects a request: the value is provenance metadata for display and
// diagnostics only.
func ClientIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		util.SetClientInfo(c, util.ParseClientInfo(c.Request.Header))
		c.Next()
	}
}
