package server

import "github.com/gin-gonic/gin"

func (s *Server) setupApi() {
	api := s.engine.Group("/api")
	s.setupV1Wallet(api)
}

func (s *Server) setupV1Wallet(api *gin.RouterGroup) {
	v1 := api.Group("/v1")
	v1.POST("/register", s.handler.Register)
	v1.POST("/login", s.handler.Login)
	v1.GET("/balance", s.handler.CheckAuth, s.handler.Balance)
	v1.POST("/wallet/deposit", s.handler.CheckAuth, s.handler.Deposit)
	v1.POST("/wallet/withdraw", s.handler.CheckAuth, s.handler.Withdraw)
	v1.GET("/exchange/rates", s.handler.Rates)
	v1.POST("/exchange", s.handler.CheckAuth, s.handler.Exchange)
}
