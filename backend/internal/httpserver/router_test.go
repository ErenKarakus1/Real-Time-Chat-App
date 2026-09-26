package httpserver

import (
	"testing"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/handlers"
	"github.com/gin-gonic/gin"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	registerRoutes(
		router,
		"test-secret",
		nil,
		handlers.NewAuthHandler(nil, nil),
		handlers.NewConversationHandler(nil),
		handlers.NewMessageHandler(nil, nil),
		handlers.NewWebSocketHandler(nil, nil, nil, "test-secret"),
	)

	routes := make(map[string]bool)
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	expectedRoutes := []string{
		"GET /health",
		"POST /auth/register",
		"POST /auth/login",
		"GET /auth/me",
		"GET /users/search",
		"GET /users/presence",
		"GET /conversations",
		"PATCH /conversations/:conversation_id",
		"DELETE /conversations/:conversation_id",
		"POST /conversations/rooms",
		"POST /conversations/direct",
		"POST /conversations/:conversation_id/read",
		"PATCH /conversations/:conversation_id/owner",
		"GET /conversations/:conversation_id/participants",
		"POST /conversations/:conversation_id/participants",
		"PATCH /conversations/:conversation_id/participants/:user_id/role",
		"DELETE /conversations/:conversation_id/participants/me",
		"DELETE /conversations/:conversation_id/participants/:user_id",
		"GET /conversations/:conversation_id/messages",
		"POST /conversations/:conversation_id/messages",
		"PATCH /conversations/:conversation_id/messages/:message_id",
		"DELETE /conversations/:conversation_id/messages/:message_id",
		"GET /ws/conversations/:conversation_id",
	}

	for _, route := range expectedRoutes {
		if !routes[route] {
			t.Fatalf("missing route %s", route)
		}
	}

	if len(routes) != len(expectedRoutes) {
		t.Fatalf("registered %d routes, want %d", len(routes), len(expectedRoutes))
	}
}
