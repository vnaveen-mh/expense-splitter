package main

import (
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[REQUEST] %s | %s | %s %s\n",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path)

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("[RESPONSE] %s | %s | %s %s | Status: %d | Duration: %v",
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration)
	})
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "create_group", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "create_group", Description: "Create a group"}, CreateGroup)
	mcp.AddTool(server, &mcp.Tool{Name: "list_groups", Description: "List groups"}, ListGroups)
	mcp.AddTool(server, &mcp.Tool{Name: "add_people", Description: "Add people to the group"}, AddPeople)
	mcp.AddTool(server, &mcp.Tool{Name: "get_group_info", Description: "Get group info or details"}, GetGroupInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_expense",
		Description: "Add expense to the group paid by a person",
		InputSchema: addExpenseInputSchema,
	},
		AddExpense)

	// Create the streamable HTTP handler
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})

	//handlerWithLogging := loggingMiddleware(handler)
	handlerWithLogging := handler

	log.Printf("Listening on :8080\n")
	log.Fatal(http.ListenAndServe(":8080", handlerWithLogging))
}
