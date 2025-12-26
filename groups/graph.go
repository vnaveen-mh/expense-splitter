package groups

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// graph represents a graph of nodes/people.
type graph struct {
	Name  string
	nodes map[string][]*edge
}

// edge represents a directed edge between two nodes (i.e. between two people from->to) with metadata.
// The directed edge A->B implies A has to pay TotalMicroCents to B
// Example: if TotalMicroCents is 2,000,000, and if this edge exists as A->B, then A has to pay B 2000 cents or 20$
type edge struct {
	To        string    `json:"to"`
	CreatedAt time.Time `json:"created_at"`
	Metadata  any       `json:"metadata,omitempty"` // Opaque app data
}

// newGraph creates a new empty graph.
func newGraph(name string) *graph {
	return &graph{
		Name:  name,
		nodes: make(map[string][]*edge),
	}
}

// addNode adds a node to the graph after validating the name.
// Caller must hold the group lock.
func (g *graph) addNode(node string) error {
	node = strings.TrimSpace(node)
	if node == "" {
		return fmt.Errorf("node name cannot be empty")
	}
	if _, exists := g.nodes[node]; exists {
		slog.Warn("node already exists in the graph, no action taken", "graph", g.Name, "node", node)
		return fmt.Errorf("node(%s) already exists in graph(%s)", node, g.Name)
	}
	// create an empty slice which means no edges connected to this node yet
	g.nodes[node] = []*edge{}
	slog.Debug("New node is added to Graph", "graph", g.Name, "node", node)
	return nil
}

// addEdge adds a directed edge between two nodes with metadata.
// Caller must hold the group lock.
func (g *graph) addEdge(from, to string, metadata any) error {
	edgeSlice, exists := g.nodes[from]
	if !exists {
		slog.Error("From node does not exist in the Graph", "graph", g.Name, "from", from, "to", to)
		return fmt.Errorf("from-node(%s) does not exist in the graph(%s)", from, g.Name)
	}
	if _, exists := g.nodes[to]; !exists {
		slog.Error("To node does not exist in the Graph", "graph", g.Name, "from", from, "to", to)
		return fmt.Errorf("to-node(%s) does not exist in the graph(%s)", to, g.Name)
	}

	newEdge := &edge{
		To:        to,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	}
	edgeSlice = append(edgeSlice, newEdge)
	g.nodes[from] = edgeSlice
	return nil
}

// size returns the number of nodes in the graph.
// Caller must hold the group lock.
func (g *graph) size() int {
	return len(g.nodes)
}
