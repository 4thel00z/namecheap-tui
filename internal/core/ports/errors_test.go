package ports_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

func TestAPIError(t *testing.T) {
	err := &ports.APIError{Number: 1011147, Message: "Invalid request IP"}
	want := "namecheap api error 1011147: Invalid request IP"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
	wrapped := fmt.Errorf("call failed: %w", err)
	var target *ports.APIError
	if !errors.As(wrapped, &target) || target.Number != 1011147 {
		t.Error("errors.As failed to unwrap APIError")
	}
}
