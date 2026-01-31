package main

import (
	"context"
	"expense-splitter/groups"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CreateGroupInput struct {
	Name string `json:"name,omitempty" jsonschema_description:"create a group with the given name. Spaces will be automatically converted to hyphens (e.g., 'vegas trip' becomes 'vegas-trip'). If a group with this name already exists, the tool will return an error."`
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

func CreateGroup(ctx context.Context, req *mcp.CallToolRequest, input *CreateGroupInput) (result *mcp.CallToolResult, output *CreateGroupOutput, err error) {
	name := ""
	if input != nil {
		name = input.Name
	}
	start := logToolEnter("CreateGroup", "name=%q", name)
	defer func() {
		logToolExit("CreateGroup", start, err, result, output)
	}()
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
	output = &CreateGroupOutput{
		GroupName: group.Name,
		CreatedAt: fmt.Sprint(group.CreatedAt),
	}

	return nil, output, nil
}

func ListGroups(ctx context.Context, req *mcp.CallToolRequest, input *ListGroupsInput) (result *mcp.CallToolResult, output *ListGroupsOutput, err error) {
	start := logToolEnter("ListGroups", "")
	defer func() {
		logToolExit("ListGroups", start, err, result, output)
	}()
	output = &ListGroupsOutput{
		Groups: groups.List(),
	}
	return nil, output, nil
}

func GetGroupInfo(ctx context.Context, req *mcp.CallToolRequest, input *GetGroupInfoInput) (result *mcp.CallToolResult, output *GetGroupInfoOutput, err error) {
	name := ""
	if input != nil {
		name = input.Name
	}
	start := logToolEnter("GetGroupInfo", "name=%q", name)
	defer func() {
		logToolExitWithDetail("GetGroupInfo", start, err, result, output, "output=%+v", output)
	}()
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

	output = &GetGroupInfoOutput{
		GroupName:      group.Name,
		CreatedAt:      fmt.Sprint(group.CreatedAt),
		Names:          group.GetPeople(),
		ExpenseDetails: group.GetExpenseDetails(),
		GraphDOT:       group.GetGraphDOT(),
	}

	return nil, output, nil
}
