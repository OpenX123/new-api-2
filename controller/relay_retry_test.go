package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newRetryTestContext(ctx context.Context) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil).WithContext(ctx)
	return c
}

func TestShouldRetryStopsWhenRequestContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := newRetryTestContext(ctx)
	err := types.NewError(errors.New("upstream failed"), types.ErrorCodeDoRequestFailed)

	assert.False(t, shouldRetry(c, err, 1))
}

func TestShouldRetryStopsForCancellationErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "canceled", err: context.Canceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRetryTestContext(context.Background())
			err := types.NewError(fmt.Errorf("do request failed: %w", tt.err), types.ErrorCodeDoRequestFailed)

			assert.False(t, shouldRetry(c, err, 1))
		})
	}
}

func TestShouldRetryKeepsNormalServerErrorsRetryable(t *testing.T) {
	c := newRetryTestContext(context.Background())
	err := types.NewError(errors.New("upstream failed"), types.ErrorCodeDoRequestFailed)

	assert.True(t, shouldRetry(c, err, 1))
}
