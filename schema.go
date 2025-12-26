package main

var addExpenseInputSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"properties": map[string]any{
		"group_name": map[string]any{
			"type":        "string",
			"description": "group name where this expense belong to",
		},
		"amount": map[string]any{
			"type":        "string",
			"description": "Total amount in dollars (e.g. \"208\" or \"208.50\")",
			"pattern":     `^\d+(\.\d{1,2})?$`,
		},
		"paid_by": map[string]any{
			"type":        "string",
			"description": "person who paid for the expense (must be a member of the group)",
		},
		"description": map[string]any{
			"type":        "string",
			"description": "a short description about the expense",
			"minLength":   3,
			"maxLength":   100,
		},
		"split_method": map[string]any{
			"type":        "string",
			"enum":        []any{"equal", "percentage", "weights"},
			"default":     "equal",
			"description": "How to split. If omitted, defaults to 'equal'.",
		},
		"split_percentages": map[string]any{
			"type":          "object",
			"minProperties": 1,
			"additionalProperties": map[string]any{
				"type":    "number",
				"minimum": 0,
				"maximum": 100,
			},
			"description": "Map of person -> percentage (0..100). Used only when split_method='percentage'.",
		},
		"split_weights": map[string]any{
			"type":          "object",
			"minProperties": 1,
			"additionalProperties": map[string]any{
				"type":    "number",
				"minimum": 0,
			},
			"description": "Map of person->weight. Weight 0 excludes the person from this expense. At least one weight must be > 0.",
		},
	},
	"required": []any{"group_name", "amount", "paid_by", "description"},

	// percentage => require split_percentages, forbid split_weights
	"allOf": []any{
		map[string]any{
			"if": map[string]any{
				"properties": map[string]any{
					"split_method": map[string]any{"const": "percentage"},
				},
				"required": []any{"split_method"},
			},
			"then": map[string]any{
				"required": []any{"split_percentages"},
				"not":      map[string]any{"required": []any{"split_weights"}},
			},
		},
		map[string]any{
			"if": map[string]any{
				"properties": map[string]any{
					"split_method": map[string]any{"const": "weights"},
				},
				"required": []any{"split_method"},
			},
			"then": map[string]any{
				"required": []any{"split_weights"},
				"not":      map[string]any{"required": []any{"split_percentages"}},
			},
		},
		map[string]any{
			"if": map[string]any{
				"properties": map[string]any{
					"split_method": map[string]any{"const": "equal"},
				},
				"required": []any{"split_method"},
			},
			"then": map[string]any{
				"not": map[string]any{
					"anyOf": []any{
						map[string]any{"required": []any{"split_percentages"}},
						map[string]any{"required": []any{"split_weights"}},
					},
				},
			},
		},
	},
}
