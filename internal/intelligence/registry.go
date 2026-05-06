package intelligence

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds all available intelligence datasets.
type Registry struct {
	mu       sync.RWMutex
	datasets map[string]Dataset
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{datasets: make(map[string]Dataset)}
}

// Register adds a dataset to the registry.
func (r *Registry) Register(ds Dataset) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.datasets[ds.Name()] = ds
}

// Get returns a dataset by name.
func (r *Registry) Get(name string) (Dataset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ds, ok := r.datasets[name]
	return ds, ok
}

// Remove removes a dataset from the registry.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.datasets, name)
}

// List returns all registered dataset names sorted.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.datasets))
	for name := range r.datasets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListStats returns Stats for all registered datasets.
func (r *Registry) ListStats() []Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Stats, 0, len(r.datasets))
	for _, ds := range r.datasets {
		out = append(out, ds.Stats())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// MustGet returns a dataset or panics.
func (r *Registry) MustGet(name string) Dataset {
	ds, ok := r.Get(name)
	if !ok {
		panic(fmt.Sprintf("intelligence dataset %q not registered", name))
	}
	return ds
}
