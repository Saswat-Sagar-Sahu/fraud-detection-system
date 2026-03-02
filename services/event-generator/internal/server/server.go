package server

import (
	"github.com/gin-gonic/gin"

	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/handler"
)

func NewMux(eventHandler *handler.EventHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/healthz", handler.Healthz)
	router.POST("/event", eventHandler.PostEvent)
	return router
}
