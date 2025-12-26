package main

import (
	"context"
	"expense-splitter/groups"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CreateGroupInput struct {
	Name string `json:"name,omitempty" jsonschema_description:"create a group with the given name"`
}

type CreateGroupOutput struct {
	GroupName string `json:"group_name"`
	CreatedAt string `json:"created_at"`
}

type GetGroupInfoInput struct {
	Name string `json:"name,omitempty" jsonschema_description:"get group info or details"`
}

type GetGroupInfoOutput struct {
	GroupName      string             `json:"group_name"`
	CreatedAt      string             `json:"created_at"`
	Names          []string           `json:"names"`
	ExpenseDetails map[string]float64 `json:"expense_details"`
	GraphDOT       string             `json:"graph_dot"`
}

type ListGroupsOutput struct {
	Groups []string `json:"groups"`
}

type ListGroupsInput struct{}

func CreateGroup(ctx context.Context, req *mcp.CallToolRequest, input *CreateGroupInput) (*mcp.CallToolResult, *CreateGroupOutput, error) {
	name := input.Name
	if name == "" {
		// Get the session so we can talk back to the client.
		ss, _ := req.GetSession().(*mcp.ServerSession)

		er, err := ss.Elicit(ctx, &mcp.ElicitParams{
			Mode:    "form",
			Message: "I need group name to create one",
			RequestedSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Group name",
					},
				},
				"required": []any{"name"},
			},
		})
		if err != nil {
			return nil, nil, err
		}

		if er.Action != "accept" {
			// user declined/cancelled
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "No worries — cancelled."},
				},
			}, nil, nil
		}

		if v, ok := er.Content["name"].(string); ok {
			name = v
		}
	}

	group, err := groups.Create(name)
	if err != nil {
		return nil, nil, err
	}
	output := &CreateGroupOutput{
		GroupName: group.Name,
		CreatedAt: fmt.Sprint(group.CreatedAt),
	}

	return nil, output, nil
}

func ListGroups(ctx context.Context, req *mcp.CallToolRequest, input *ListGroupsInput) (*mcp.CallToolResult, *ListGroupsOutput, error) {
	output := &ListGroupsOutput{
		Groups: groups.List(),
	}
	return nil, output, nil
}

func GetGroupInfo(ctx context.Context, req *mcp.CallToolRequest, input *GetGroupInfoInput) (*mcp.CallToolResult, *GetGroupInfoOutput, error) {
	name := input.Name
	if name == "" {
		// Get the session so we can talk back to the client.
		ss, _ := req.GetSession().(*mcp.ServerSession)

		er, err := ss.Elicit(ctx, &mcp.ElicitParams{
			Mode:    "form",
			Message: "I need group name to get the details of the group",
			RequestedSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Group name",
					},
				},
				"required": []any{"name"},
			},
		})
		if err != nil {
			return nil, nil, err
		}

		if er.Action != "accept" {
			// user declined/cancelled
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "No worries — cancelled."},
				},
			}, nil, nil
		}

		if v, ok := er.Content["name"].(string); ok {
			name = v
		}
	}

	group, exists := groups.Get(name)
	if !exists {
		return nil, nil, fmt.Errorf("group(%s) not found; create it with CreateGroup", name)
	}

	output := &GetGroupInfoOutput{
		GroupName:      group.Name,
		CreatedAt:      fmt.Sprint(group.CreatedAt),
		Names:          group.GetPeople(),
		ExpenseDetails: group.GetExpenseDetails(),
		GraphDOT:       group.GetGraphDOT(),
	}

	return nil, output, nil
}
