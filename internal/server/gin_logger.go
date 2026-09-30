package server

import (
	"log/slog"
	"time"

	"github.com/MacPiggins/gw-wallet/internal/logging"
	"github.com/gin-gonic/gin"
)

func ginSlogLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		requestContext := logging.AppendCtx(
			ctx.Request.Context(),
			slog.String("http.method", ctx.Request.Method),
			slog.String("http.path", ctx.Request.URL.Path),
		)
		ctx.Request = ctx.Request.WithContext(requestContext)

		ctx.Next()

		status := ctx.Writer.Status()
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		attrs := []slog.Attr{
			slog.Int("http.status", status),
			slog.Duration("http.latency", time.Since(startedAt)),
			slog.String("client.ip", ctx.ClientIP()),
			slog.Int("http.response_bytes", ctx.Writer.Size()),
		}
		if len(ctx.Errors) > 0 {
			attrs = append(attrs, slog.String("error", ctx.Errors.String()))
		}

		slog.LogAttrs(requestContext, level, "HTTP request completed", attrs...)
	}
}
