package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

type Server struct {
	engine  *gin.Engine
	handler *Handler
}

func NewServer(handler *Handler) *Server {
	server := Server{
		engine:  gin.New(),
		handler: handler,
	}

	server.engine.ContextWithFallback = true
	server.engine.Use(ginSlogLogger(), gin.Recovery())
	server.setupApi()
	return &server
}

func (server *Server) Run(addr string) error {
	slog.Info("running server", slog.String("address", addr))
	return server.engine.Run(addr)
}
