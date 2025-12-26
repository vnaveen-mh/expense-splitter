package main

import (
	"context"
	"errors"
	"expense-splitter/groups"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AddExpenseInput struct {
	GroupName        *string            `json:"group_name,omitempty" jsonschema:"group where this expense belongs"`
	Amount           *string            `json:"amount,omitempty" jsonschema:"amount in dollars (e.g. \"208\", \"208.50\")"`
	PaidBy           *string            `json:"paid_by,omitempty" jsonschema:"the person who paid for this expense"`
	Description      *string            `json:"description,omitempty" jsonschema:"description of the expense"`
	SplitMethod      *string            `json:"split_method,omitempty" jsonschema:"how to split the expense" jsonschema_enum:"equal,percentage,weights" jsonschema_default:"equal"`
	SplitPercentages map[string]float64 `json:"split_percentages,omitempty" jsonschema:"percent ownership by person, values 0..100"`
	SplitWeights     map[string]float64 `json:"split_weights,omitempty" jsonschema:"Map person->weight (relative shares)"`
}

type AddExpenseOutput struct {
	Msg string `json:"msg" jsonschema_description:"success message"`
}

func AddExpense(ctx context.Context, req *mcp.CallToolRequest, input *AddExpenseInput) (*mcp.CallToolResult, *AddExpenseOutput, error) {
	groupName := input.GroupName
	amountStr := input.Amount
	paidBy := input.PaidBy
	expenseDescription := input.Description
	splitMethod := input.SplitMethod
	percentages := input.SplitPercentages
	weights := input.SplitWeights

	if groupName == nil {
		msg := "What's the group name?"
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"group_name": map[string]any{
					"type":        "string",
					"description": "group name where this expense belong to",
				},
			},
			"required": []any{"group_name"},
		}
		er, err := sendExpenseElicitRequest(ctx, req, msg, schema)
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
		if v, ok := er.Content["group_name"].(string); ok {
			groupName = &v
		}
		if groupName == nil || strings.TrimSpace(*groupName) == "" {
			return nil, nil, errors.New("group_name is required")
		}

		// check if group exists in the app
		_, exists := groups.Get(*groupName)
		if !exists {
			return nil, nil, errors.New("no such group")
		}
	}
	//
	if amountStr == nil {
		msg := "What is the amount in dollars?"
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"amount": map[string]any{
					"type":             "number",
					"description":      "total amount of the expense in dollars",
					"exclusiveMinimum": 0,
				},
			},
			"required": []any{"amount"},
		}
		er, err := sendExpenseElicitRequest(ctx, req, msg, schema)
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
		if v, ok := er.Content["amount"].(string); ok {
			amountStr = &v
		}
		if amountStr == nil {
			return nil, nil, errors.New("amount is required")
		}
	}
	//
	if paidBy == nil {
		group, _ := groups.Get(*groupName)
		people := group.GetPeople()
		enumPeople := make([]any, 0, len(people))
		for _, p := range people {
			enumPeople = append(enumPeople, p)
		}

		msg := fmt.Sprintf("Who paid for the expense in %s? (select from group)", *groupName)
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"paid_by": map[string]any{
					"type":        "string",
					"description": "person who paid for the expense",
					"enum":        enumPeople,
				},
			},
			"required": []any{"paid_by"},
		}
		er, err := sendExpenseElicitRequest(ctx, req, msg, schema)
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
		if v, ok := er.Content["paid_by"].(string); ok {
			paidBy = &v
		}
		if paidBy == nil || strings.TrimSpace(*paidBy) == "" {
			return nil, nil, errors.New("paid_by is required")
		}
	}
	//
	if expenseDescription == nil {
		msg := "What is the expense about? Add a short description"
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"description": map[string]any{
					"type":        "string",
					"description": "a short description about the expense",
					"minLength":   3,
					"maxLength":   100,
				},
			},
			"required": []any{"description"},
		}
		er, err := sendExpenseElicitRequest(ctx, req, msg, schema)
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
		if v, ok := er.Content["description"].(string); ok {
			expenseDescription = &v
		}
		if expenseDescription == nil || strings.TrimSpace(*expenseDescription) == "" {
			return nil, nil, errors.New("description is required")
		}

	}
	//
	if splitMethod == nil || strings.TrimSpace(*splitMethod) == "" {
		v := "equal"
		splitMethod = &v
	}
	if *splitMethod == "percentage" && len(percentages) == 0 {
		if groupName == nil || strings.TrimSpace(*groupName) == "" {
			return nil, nil, errors.New("group_name is required")
		}
		group, _ := groups.Get(*groupName)
		people := group.GetPeople()
		enumPeople := make([]any, 0, len(people))
		for _, p := range people {
			enumPeople = append(enumPeople, p)
		}

		msg := "I need person to percentage map to split the expenses"
		schema := map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"split_percentages": map[string]any{
					"type":          "object",
					"minProperties": 1,
					"propertyNames": map[string]any{
						"type": "string",
						"enum": enumPeople,
					},
					"additionalProperties": map[string]any{
						"type":    "number",
						"minimum": 0,
						"maximum": 100,
					},
					"description": "Map of person->percentage. Must sum to 100.",
				},
			},
			"required": []any{"split_percentages"},
		}
		er, err := sendExpenseElicitRequest(ctx, req, msg, schema)
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

		if m, ok := er.Content["split_percentages"].(map[string]interface{}); ok {
			percentages = make(map[string]float64)
			for name, pct := range m {
				switch x := pct.(type) {
				case float64:
					percentages[name] = x
				case string:
					if f, err := strconv.ParseFloat(x, 64); err == nil {
						percentages[name] = f
					}
				}
			}
		}
	}
	//
	if *splitMethod == "weights" && len(weights) == 0 {
		if groupName == nil || strings.TrimSpace(*groupName) == "" {
			return nil, nil, errors.New("group_name is required")
		}
		group, _ := groups.Get(*groupName)
		people := group.GetPeople()
		enumPeople := make([]any, 0, len(people))
		for _, p := range people {
			enumPeople = append(enumPeople, p)
		}

		msg := "I need person to weights map to split the expenses"
		schema := map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"split_weights": map[string]any{
					"type":          "object",
					"minProperties": 1,
					"propertyNames": map[string]any{
						"type": "string",
						"enum": enumPeople, // build from group members
					},
					"additionalProperties": map[string]any{
						"type":    "number",
						"minimum": 0,
					},
					"description": "Map of person->weight. Weight 0 excludes the person from this expense. At least one weight must be > 0.",
				},
			},
			"required": []any{"split_weights"},
		}
		er, err := sendExpenseElicitRequest(ctx, req, msg, schema)
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
		if m, ok := er.Content["split_weights"].(map[string]interface{}); ok {
			weights = make(map[string]float64)
			for name, w := range m {
				switch x := w.(type) {
				case float64:
					weights[name] = x
				case string:
					if f, err := strconv.ParseFloat(x, 64); err == nil {
						weights[name] = f
					}
				}
			}
		}
	}

	group, exists := groups.Get(*groupName)
	if !exists {
		return nil, nil, errors.New("no such group exists")
	}
	people := group.GetPeople()

	// after ensuring group exists and people list known
	// validate
	totalMicroCents, err := parseDollarsToMicroCents(*amountStr)
	if err != nil {
		return nil, nil, err
	}

	if splitMethod == nil {
		return nil, nil, errors.New("split_method is required")
	}
	if *splitMethod == "percentage" {
		// validate percentages presence, sum, members
		if len(percentages) == 0 {
			return nil, nil, errors.New("split_percentages required for percentage split")
		}
		total := 0.0
		for _, v := range percentages {
			total += v
		}
		if math.Abs(total-100.0) > 0.01 {
			return nil, nil, fmt.Errorf("split_percentages must sum to 100 (got %.2f)", total)
		}
		memberSet := map[string]bool{}
		for _, p := range people {
			memberSet[p] = true
		}
		for name := range percentages {
			if !memberSet[name] {
				return nil, nil, fmt.Errorf("%q is not in group %s", name, *groupName)
			}
		}
	}
	if *splitMethod == "weights" {
		// should I support 0 weights? for example, to exclude a person from an expense
		sumW := 0.0
		for _, w := range weights {
			if w < 0 {
				return nil, nil, fmt.Errorf("weights must be >= 0")
			}
			sumW += w
		}
		if sumW == 0 {
			return nil, nil, fmt.Errorf("sum of weights must be > 0 (atleast one participant is required).")
		}
	}

	// add an expense to the app
	group.AddExpense(&groups.Expense{
		TotalMicroCents:  totalMicroCents,
		PaidBy:           *paidBy,
		Description:      *expenseDescription,
		SplitMethod:      *splitMethod,
		SplitPercentages: percentages,
		SplitWeights:     weights,
	})

	output := &AddExpenseOutput{
		Msg: "success",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Expense added successfully."},
		},
	}, output, nil
}

func sendExpenseElicitRequest(ctx context.Context, req *mcp.CallToolRequest, msg string, schema map[string]any) (*mcp.ElicitResult, error) {
	ss, ok := req.GetSession().(*mcp.ServerSession)
	if !ok || ss == nil {
		return nil, fmt.Errorf("expected *mcp.ServerSession, got %T", ss)
	}

	er, err := ss.Elicit(ctx, &mcp.ElicitParams{
		Mode:            "form",
		Message:         msg,
		RequestedSchema: schema,
	})
	return er, err
}

func parseDollarsToMicroCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("amount is empty")
	}

	parts := strings.SplitN(s, ".", 2)

	// dollars
	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || dollars < 0 {
		return 0, fmt.Errorf("invalid dollar amount: %q", s)
	}

	cents := int64(0)

	if len(parts) == 2 {
		frac := parts[1]
		if len(frac) > 2 {
			return 0, fmt.Errorf("too many decimal places: %q", s)
		}

		if len(frac) == 1 {
			frac += "0"
		}

		c, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid cents: %q", s)
		}
		cents = c
	}

	return (dollars*100 + cents) * 1000, nil
}
