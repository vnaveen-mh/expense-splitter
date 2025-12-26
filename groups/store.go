package groups

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type groupManager struct {
	store map[string]*Group
	mu    sync.Mutex
}

var groupMgr *groupManager

// init initializes the in-memory group store.
func init() {
	groupMgr = &groupManager{
		store: map[string]*Group{},
	}
}

// Create validates the name and creates a new group if it doesn't already exist.
func Create(name string) (*Group, error) {
	displayName := strings.TrimSpace(name)
	key := normalizeName(displayName)

	groupMgr.mu.Lock()
	defer groupMgr.mu.Unlock()

	if existing, exists := groupMgr.store[key]; exists {
		return nil, fmt.Errorf("group(%s) already exists", existing.Name)
	}
	group, err := NewGroup(displayName)
	if err != nil {
		return nil, err
	}
	groupMgr.store[key] = group
	return group, nil
}

// Get returns the group by name and whether it exists.
func Get(name string) (*Group, bool) {
	groupMgr.mu.Lock()
	defer groupMgr.mu.Unlock()

	group, exists := groupMgr.store[normalizeName(name)]
	return group, exists
}

// List returns all group names in sorted order.
func List() []string {
	groupMgr.mu.Lock()
	defer groupMgr.mu.Unlock()

	names := make([]string, 0, len(groupMgr.store))
	for _, group := range groupMgr.store {
		names = append(names, group.Name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	return names
}

// ListGroups returns all groups in name-sorted order.
func ListGroups() []*Group {
	groupMgr.mu.Lock()
	defer groupMgr.mu.Unlock()

	list := make([]*Group, 0, len(groupMgr.store))
	for _, group := range groupMgr.store {
		list = append(list, group)
	}
	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	return list
}

// Delete removes a group by name and reports whether it was deleted.
func Delete(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}

	groupMgr.mu.Lock()
	defer groupMgr.mu.Unlock()

	key := normalizeName(name)
	if _, exists := groupMgr.store[key]; !exists {
		return false
	}
	delete(groupMgr.store, key)
	return true
}
