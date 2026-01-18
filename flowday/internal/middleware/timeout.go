package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware creates a timeout middleware that cancels the request context
// after the specified duration. This helps prevent long-running requests from
// consuming server resources indefinitely.
//
// Handlers should check ctx.Done() or use context-aware operations (like
// database queries with context) to respect the timeout.
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout from the request context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace the request context with the timeout context
		c.Request = c.Request.WithContext(ctx)

		// Monitor for timeout in a separate goroutine
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded && !c.IsAborted() && !c.Writer.Written() {
					c.JSON(http.StatusRequestTimeout, gin.H{
						"error": "Request timeout",
					})
					c.Abort()
				}
			case <-done:
				return
			}
		}()

		// Continue with the request
		c.Next()

		// Signal completion
		close(done)
	}
}