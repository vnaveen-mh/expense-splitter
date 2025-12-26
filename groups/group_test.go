package groups

import "testing"

func TestExpenseSplitByPercentage(t *testing.T) {
	groupName := "sf-trip"
	group, err := Create(groupName)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Alice", "Bob", "Charlie"} {
		if err := group.AddPerson(name); err != nil {
			t.Fatal(err)
		}
	}
	group.AddExpense(&Expense{
		PaidBy:          "Alice",
		TotalMicroCents: 100 * 100 * 1000,
		Description:     "show tickets",
		SplitMethod:     "percentage",
		SplitPercentages: map[string]float64{
			"Alice":   20,
			"Bob":     40,
			"Charlie": 40,
		},
	})

	t.Log(group.GetExpenseDetails())
}

func TestExpenseSplitByWeights(t *testing.T) {
	groupName := "napa-trip"
	group, err := Create(groupName)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Alice", "Bob", "Charlie"} {
		if err := group.AddPerson(name); err != nil {
			t.Fatal(err)
		}
	}
	group.AddExpense(&Expense{
		PaidBy:          "Alice",
		TotalMicroCents: 100 * 100 * 1000,
		Description:     "show tickets",
		SplitMethod:     "weights",
		SplitWeights: map[string]float64{
			"Alice":   0,
			"Bob":     4,
			"Charlie": 4,
		},
	})

	t.Log(group.GetExpenseDetails())
}
