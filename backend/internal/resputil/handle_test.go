package resputil

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
)

func TestHandleErrorPreservesWrappedBizError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	HandleError(ctx, bizerr.BadRequest.ParameterError.Wrap(errors.New("x"), "bad input"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Code != bizerr.BadRequest.ParameterError {
		t.Fatalf("expected code %d, got %d", bizerr.BadRequest.ParameterError, response.Code)
	}
	if response.Message != "bad input" {
		t.Fatalf("expected message %q, got %q", "bad input", response.Message)
	}
}

func TestSuccessKeepsEmptyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Success(ctx, gin.H{"ok": true})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Message != "" {
		t.Fatalf("expected empty message, got %q", response.Message)
	}
}
