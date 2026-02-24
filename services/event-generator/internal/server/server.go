package server

import (
	"net/http"

	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/handler"
)

func NewMux(eventHandler *handler.EventHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.Healthz)
	mux.HandleFunc("POST /event", eventHandler.PostEvent)
	return mux
}
