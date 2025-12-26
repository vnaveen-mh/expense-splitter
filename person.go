package main

import (
	"context"
	"errors"
	"expense-splitter/groups"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AddPeopleInput struct {
	Names     []string `json:"names,omitempty" jsonschema_description:"names of the people"`
	GroupName string   `json:"group_name,omitempty" jsonschema_description:"group name to which the person will be added to"`
}

type AddPeopleOutput struct {
	Msg string `json:"msg" jsonschema_description:"success message"`
}

func parseNames(value any) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case []any:
		names := make([]string, 0, len(v))
		for _, item := range v {
			name, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("names must be an array of strings")
			}
			names = append(names, name)
		}
		return names, nil
	default:
		return nil, fmt.Errorf("names must be an array of strings")
	}
}

func AddPeople(ctx context.Context, req *mcp.CallToolRequest, input *AddPeopleInput) (*mcp.CallToolResult, *AddPeopleOutput, error) {
	groupName := input.GroupName
	names := input.Names
	if len(names) == 0 || groupName == "" {
		// Get the session so we can talk back to the client.
		ss, _ := req.GetSession().(*mcp.ServerSession)

		er, err := ss.Elicit(ctx, &mcp.ElicitParams{
			Mode:    "form",
			Message: "I need group name and person name(s)",
			RequestedSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"names": map[string]any{
						"type":        "array",
						"description": "one or more person names",
						"items": map[string]any{
							"type": "string",
						},
						"minItems": 1,
					},
					"group_name": map[string]any{
						"type":        "string",
						"description": "group name",
					},
				},
				"required": []any{"names", "group_name"},
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

		if v, ok := er.Content["names"]; ok {
			parsed, err := parseNames(v)
			if err != nil {
				return nil, nil, err
			}
			names = parsed
		}
		if v, ok := er.Content["group_name"].(string); ok {
			groupName = v
		}
	}
	if len(names) == 0 || groupName == "" {
		return nil, nil, errors.New("group_name and names are required; provide a group name and at least one person name")
	}

	group, exists := groups.Get(groupName)
	if !exists {
		return nil, nil, fmt.Errorf("group(%s) not found; create it with CreateGroup", groupName)
	}
	for _, name := range names {
		if err := group.AddPerson(name); err != nil {
			return nil, nil, err
		}
	}

	output := &AddPeopleOutput{
		Msg: "success",
	}

	return nil, output, nil
}
