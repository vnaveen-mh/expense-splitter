package groups

import (
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Concurrency: Group's mutex is the single lock that protects both Group state and the internal graph.
var groupNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z_-]{0,31}$`)
var personNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z_ -]{0,31}$`)

// Person represents a node in the graph
// It has to be a unique name within the group
type Person struct {
	Name string
	// Email, phone
}

type Group struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`

	graph            *graph `json:"-"`
	people           map[string]*Person
	expenses         map[int]*Expense
	expenseIdCounter int
	mu               sync.Mutex
}

// ID is unique only within the graph
// they take on values such as 1, 2, 3 etc.
type Expense struct {
	ID               int                `json:"id"`
	TotalMicroCents  int64              `json:"total_micro_cents" binding:"required"`
	PaidBy           string             `json:"paid_by" binding:"required"`
	Description      string             `json:"description" binding:"required"`
	SplitMethod      string             `json:"split_type" binding:"required"`
	SplitPercentages map[string]float64 `json:"split_percentages"`
	SplitWeights     map[string]float64 `json:"split_weights"`
}

type EdgeMetadata struct {
	AmountInMicroCents int64
	ExpenseID          int
}

// NewGroup creates a new group and returns it
// It initializes an interanl graph data struct
func NewGroup(name string) (*Group, error) {
	// validate name
	name = strings.TrimSpace(name)
	if !groupNamePattern.MatchString(name) {
		return nil, fmt.Errorf("group name must start with a letter, match %q, and be [1, 32] chars long", groupNamePattern.String())
	}

	group := &Group{
		Name:      name,
		CreatedAt: time.Now(),
		graph:     newGraph(name),
		people:    make(map[string]*Person),
		expenses:  make(map[int]*Expense),
	}
	return group, nil
}

// AddPerson adds a person to the group
func (g *Group) AddPerson(name string) error {
	// validate name
	displayName := strings.TrimSpace(name)
	if !personNamePattern.MatchString(displayName) {
		return fmt.Errorf("person name must start with a letter, match %q, and be [1, 32] chars long", personNamePattern.String())
	}
	key := normalizeName(displayName)

	p := &Person{
		Name: displayName,
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// validate if person already exists
	if existing, exists := g.people[key]; exists {
		slog.Error("person already in the group", "person", existing.Name, "group", g.Name)
		return fmt.Errorf("person(%s) already exists in group(%s)", existing.Name, g.Name)
	}

	if err := g.graph.addNode(key); err != nil {
		return err
	}
	g.people[key] = p
	return nil
}

// Size returns the number of people in the group
func (g *Group) Size() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return len(g.people)
}

// AddExpense adds an expense to the group.
// It may result in creating several edges between the nodes of an internal graph
func (g *Group) AddExpense(e *Expense) error {
	// validate fields that dont' require lock
	if e.TotalMicroCents <= 0 {
		slog.Error("expense TotalMicroCents cannot be negative", "total_micro_cents", e.TotalMicroCents)
		return fmt.Errorf("expense TotalMicroCents(%d) cannot be 0 or negative", e.TotalMicroCents)
	}
	e.Description = strings.TrimSpace(e.Description)
	if e.Description == "" {
		slog.Error("expense description cannot be empty")
		return fmt.Errorf("expense description cannot be empty")
	}
	if err := validateSplitMethod(e.SplitMethod); err != nil {
		slog.Error("split method validation failed", "split_method", e.SplitMethod)
		return err
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// validate fields that require lock
	if len(g.people) <= 1 {
		slog.Error("group must contain atleast 2 people to add an expense", "group", g.Name, "size", len(g.people))
		return fmt.Errorf("group(%s) must contain atleast 2 people to add an expense, current size=%d", g.Name, len(g.people))
	}
	paidByKey := normalizeName(e.PaidBy)
	to, exists := g.people[paidByKey]
	if !exists {
		slog.Error("expense PaidBy person not in the group", "paid_by", e.PaidBy, "group", g.Name)
		return fmt.Errorf("expense PaidBy person(%s) must be in the group(%s)", e.PaidBy, g.Name)
	}

	normalizedPercentages, err := normalizeSplitMap(e.SplitPercentages)
	if err != nil {
		return err
	}
	for name := range normalizedPercentages {
		if _, exists := g.people[name]; !exists {
			slog.Error("expense split_percentages validation failed, name not in the group", "name", name, "group", g.Name)
			return fmt.Errorf("expense split_percentages validation failed, name(%s) not in the group(%s)", name, g.Name)
		}
	}

	normalizedWeights, err := normalizeSplitMap(e.SplitWeights)
	if err != nil {
		return err
	}
	for name := range normalizedWeights {
		if _, exists := g.people[name]; !exists {
			slog.Error("expense split_weights validation failed, name not in the group", "name", name, "group", g.Name)
			return fmt.Errorf("expense split_weights validation failed, name(%s) not in the group(%s)", name, g.Name)
		}
	}

	// names can be formed using graph or g.people
	names := []string{}
	for key := range g.people {
		names = append(names, key)
	}

	e.PaidBy = to.Name
	e.SplitPercentages = normalizedPercentages
	e.SplitWeights = normalizedWeights

	var shares map[string]int64
	switch e.SplitMethod {
	case "equal":
		var err error
		shares, err = splitEqual(e.TotalMicroCents, names)
		if err != nil {
			slog.Error("error while splitting equally", "group", g.Name, "error", err.Error())
			return err
		}
	case "percentage":
		var err error
		shares, err = splitByPercent(e.TotalMicroCents, e.SplitPercentages)
		if err != nil {
			slog.Error("error while splitting by percent", "group", g.Name, slog.Any("split_percentages", e.SplitPercentages),
				"error", err.Error())
			return err
		}
	case "weights":
		var err error
		shares, err = splitByWeights(e.TotalMicroCents, e.SplitWeights)
		if err != nil {
			slog.Error("error while splitting by weights", "group", g.Name, slog.Any("split_weignts", e.SplitWeights),
				"error", err.Error())
			return err
		}
	}

	if len(g.people) != len(g.graph.nodes) {
		return fmt.Errorf("group(%s) graph/people out of sync", g.Name)
	}
	for name := range g.people {
		if _, ok := g.graph.nodes[name]; !ok {
			return fmt.Errorf("person(%s) missing from graph(%s)", name, g.Name)
		}
	}
	for name := range g.graph.nodes {
		if _, ok := g.people[name]; !ok {
			return fmt.Errorf("graph has extra node(%s) in group(%s)", name, g.Name)
		}
	}

	g.expenseIdCounter++
	e.ID = g.expenseIdCounter
	g.expenses[e.ID] = e

	// add edges
	for fromKey, from := range g.people {
		if fromKey == paidByKey {
			// skip this
			continue
		}
		if owed, exists := shares[fromKey]; exists {
			slog.Debug("AddExpense", "split_method", e.SplitMethod, "from", from.Name, "to", to.Name, "owed_in_micro_cents", owed)
			metadata := EdgeMetadata{
				AmountInMicroCents: owed,
				ExpenseID:          e.ID,
			}
			if err := g.graph.addEdge(fromKey, paidByKey, metadata); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Group) GetExpenseDetails() map[string]float64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	people := []string{}
	for p := range g.graph.nodes {
		people = append(people, p)
	}
	result := map[string]float64{}

	for _, from := range people {
		for _, to := range people {
			if from == to {
				continue
			}
			amount := g.getMoneyTobePaid(from, to)
			if amount > 0 {
				key := fmt.Sprintf("%s to pay %s", g.displayName(from), g.displayName(to))
				result[key] = amount
			}
		}
	}
	return result
}

func (g *Group) GetPeople() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	people := make([]string, 0, len(g.people))
	for _, person := range g.people {
		people = append(people, person.Name)
	}
	sort.Slice(people, func(i, j int) bool {
		return strings.ToLower(people[i]) < strings.ToLower(people[j])
	})
	return people
}

// GetGraphDOT returns a DOT graph representation of the group's expense edges.
// The caller does not need to handle locking; this method locks internally.
func (g *Group) GetGraphDOT() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	names := make([]string, 0, len(g.people))
	for name := range g.people {
		names = append(names, name)
	}
	sort.Strings(names)

	type edgeKey struct {
		from string
		to   string
	}

	edgeSums := make(map[edgeKey]int64)
	for from, edges := range g.graph.nodes {
		for _, edge := range edges {
			edgeInfo := edge.Metadata.(EdgeMetadata)
			edgeSums[edgeKey{from: from, to: edge.To}] += edgeInfo.AmountInMicroCents
		}
	}

	keys := make([]edgeKey, 0, len(edgeSums))
	for k := range edgeSums {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].from == keys[j].from {
			return keys[i].to < keys[j].to
		}
		return keys[i].from < keys[j].from
	})

	var b strings.Builder
	fmt.Fprintf(&b, "digraph %q {\n", g.Name)
	for _, key := range names {
		fmt.Fprintf(&b, "  %q [label=%q];\n", key, g.displayName(key))
	}
	for _, k := range keys {
		micro := edgeSums[k]
		if micro <= 0 {
			continue
		}
		label := formatMicroCentsAsDollars(micro)
		fmt.Fprintf(&b, "  %q -> %q [label=%q];\n", k.from, k.to, label)
	}
	b.WriteString("}\n")
	return b.String()
}

func (g *Group) displayName(key string) string {
	if p, ok := g.people[key]; ok {
		return p.Name
	}
	return key
}

func normalizeSplitMap(input map[string]float64) (map[string]float64, error) {
	if len(input) == 0 {
		return map[string]float64{}, nil
	}
	out := make(map[string]float64, len(input))
	for name, value := range input {
		key := normalizeName(name)
		if key == "" {
			return nil, fmt.Errorf("split map contains empty name")
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("duplicate name in split map after normalization: %q", name)
		}
		out[key] = value
	}
	return out, nil
}

func formatMicroCentsAsDollars(micro int64) string {
	roundedCents := (micro + 500) / 1000
	return fmt.Sprintf("$%.2f", float64(roundedCents)/100.0)
}

func splitEqual(totalMicroCents int64, names []string) (map[string]int64, error) {
	// returns map of each person's share
	n := int64(len(names))
	if n <= 1 {
		return nil, fmt.Errorf("length of the people must be atleast 2, current size=%d", len(names))
	}

	base := totalMicroCents / n
	rem := totalMicroCents % n

	// deterministic ordering for remainder distribution
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)

	shares := map[string]int64{}
	for i, p := range sorted {
		share := base
		if int64(i) < rem {
			share++ // distribute extra pennies
		}
		shares[p] = share
	}
	return shares, nil
}

func splitByPercent(totalMicroCents int64, perc map[string]float64) (map[string]int64, error) {
	// Validate sum ~ 100
	sum := 0.0
	for _, v := range perc {
		sum += v
	}
	if math.Abs(sum-100.0) > 0.01 {
		return nil, fmt.Errorf("percentages must sum to 100 (got %.4f)", sum)
	}

	// Compute raw shares in cents using floor, then distribute remaining by largest fractional remainder
	type item struct {
		name string
		raw  float64
		base int64
		frac float64
	}

	items := make([]item, 0, len(perc))
	used := int64(0)
	for name, p := range perc {
		raw := (p / 100.0) * float64(totalMicroCents)
		base := int64(math.Floor(raw))
		items = append(items, item{name: name, raw: raw, base: base, frac: raw - float64(base)})
		used += base
	}

	rem := totalMicroCents - used
	sort.Slice(items, func(i, j int) bool {
		if items[i].frac == items[j].frac {
			return items[i].name < items[j].name
		}
		return items[i].frac > items[j].frac
	})

	shares := map[string]int64{}
	for _, it := range items {
		shares[it.name] = it.base
	}
	for i := int64(0); i < rem; i++ {
		shares[items[i%int64(len(items))].name]++
	}

	// Optional: ensure all group members exist in shares; you can decide policy.
	// Often you want only provided keys to participate.

	return shares, nil
}

func splitByWeights(totalMicroCents int64, w map[string]float64) (map[string]int64, error) {
	sumW := 0.0
	for _, v := range w {
		if v < 0 {
			return nil, fmt.Errorf("weights must be >= 0")
		}
		sumW += v
	}
	if sumW <= 0 {
		return nil, fmt.Errorf("sum of weights must be > 0")
	}

	type item struct {
		name string
		raw  float64
		base int64
		frac float64
	}

	items := make([]item, 0, len(w))
	used := int64(0)
	for name, weight := range w {
		if weight == 0 {
			continue
		}
		raw := (weight / sumW) * float64(totalMicroCents)
		base := int64(math.Floor(raw))
		items = append(items, item{name: name, raw: raw, base: base, frac: raw - float64(base)})
		used += base
	}

	rem := totalMicroCents - used
	sort.Slice(items, func(i, j int) bool {
		if items[i].frac == items[j].frac {
			return items[i].name < items[j].name
		}
		return items[i].frac > items[j].frac
	})

	shares := map[string]int64{}
	for _, it := range items {
		shares[it.name] = it.base
	}
	for i := int64(0); i < rem; i++ {
		shares[items[i%int64(len(items))].name]++
	}

	return shares, nil
}

// getMoneyToBePaid returns money to be paid by "from" to "to" in dollars
// The function does not do locking. The callers must ensure to lock group level mutex.
func (g *Group) getMoneyTobePaid(from, to string) float64 {
	// get total by processing all edges of the form: from->to
	sum := int64(0)
	for _, edge := range g.graph.nodes[from] {
		if edge.To == to {
			edgeInfo := edge.Metadata.(EdgeMetadata)
			slog.Debug("getMoneyTobePaid sum1:", "from", from, "to", to, slog.Any("edgeMetadata", edgeInfo))
			sum += edgeInfo.AmountInMicroCents
		}
	}

	// get total by processing all edges of the form: to->from
	sum2 := int64(0)
	for _, edge := range g.graph.nodes[to] {
		if edge.To == from {
			edgeInfo := edge.Metadata.(EdgeMetadata)
			slog.Debug("getMoneyTobePaid sum2:", "from", from, "to", to, slog.Any("expense", edgeInfo))
			sum2 += edgeInfo.AmountInMicroCents
		}
	}
	// return the amount in dollars
	cents := float64(sum-sum2) / 1000.0
	if cents < 1 {
		cents = 0
	}
	return cents / 100.0
}

func validateSplitMethod(splitMethod string) error {
	validValues := []string{"equal", "percentage", "weights"}
	for _, v := range validValues {
		if v == splitMethod {
			return nil
		}
	}
	return fmt.Errorf("split method must be one of equal|percentage|weights")
}
