package bizerr

import (
	"errors"
	"testing"
)

func TestWrapPreservesBizErrorAndCause(t *testing.T) {
	cause := errors.New("invalid email")
	err := BadRequest.ParameterError.Wrap(cause, "bad input")

	bErr, ok := FromError(err)
	if !ok {
		t.Fatal("expected wrapped error to be recognized as BizError")
	}
	if bErr.Code != BadRequest.ParameterError {
		t.Fatalf("expected code %d, got %d", BadRequest.ParameterError, bErr.Code)
	}
	if bErr.Message != "bad input" {
		t.Fatalf("expected message %q, got %q", "bad input", bErr.Message)
	}
	if !errors.Is(err, BadRequest.Base) {
		t.Fatal("expected wrapped error to match bad request group")
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected wrapped error to preserve the original cause")
	}
}
