package routes

import (
	"net/http"

	"github.com/dwikalam/ecommerce-service/internal/app/handlers"
	"github.com/dwikalam/ecommerce-service/internal/app/types/interfaces"
)

func NewHttpHandler(logger interfaces.Logger, testHandler *handlers.TestHandler) http.Handler {
	mux := http.NewServeMux()

	if testHandler != nil {
		mux.Handle("GET /api/v1/test", testHandler.HandleHelloWorldResponse())
		mux.Handle("GET /api/v1/test/timeout", testHandler.HandleTimeoutExceededResponse())
	}

	return mux
}
