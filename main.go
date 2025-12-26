package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

	log.Printf("Running mcp server...\n")
	// Run the server over stdin/stdout until the client disconnects
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
