package agent

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func recordAgentMutationOperationLog(
	_ *gin.Context,
	_ string,
	_ json.RawMessage,
	_ any,
	_ error,
	_ string,
	_ string,
) {
}
