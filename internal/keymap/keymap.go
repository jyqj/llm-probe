package keymap

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Entry represents a downstream-to-upstream key mapping.
type Entry struct {
	UpstreamBase string `json:"upstream_base"`
	UpstreamKey  string `json:"upstream_key"`
	Label        string `json:"label"`
	CreatedAt    string `json:"created_at"`
}

// KeyMap manages downstream→upstream key mappings.
type KeyMap struct {
	mu       sync.RWMutex
	entries  map[string]*Entry
	filePath string
}

// New creates a KeyMap, loading from the given file path.
func New(filePath string) *KeyMap {
	km := &KeyMap{
		entries:  make(map[string]*Entry),
		filePath: filePath,
	}
	km.Load()
	return km
}

// Load reads the keys file from disk.
func (km *KeyMap) Load() {
	km.mu.Lock()
	defer km.mu.Unlock()

	data, err := os.ReadFile(km.filePath)
	if err != nil {
		return
	}
	var entries map[string]*Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return
	}
	km.entries = entries
}

// Save writes the keys to disk atomically (holds lock for entire operation).
func (km *KeyMap) Save() error {
	km.mu.Lock()
	defer km.mu.Unlock()
	data, err := json.MarshalIndent(km.entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := km.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, km.filePath)
}

// Resolve looks up a downstream key and returns the upstream config.
func (km *KeyMap) Resolve(downstreamKey string) *Entry {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.entries[downstreamKey]
}

// Register adds a new key mapping. Returns the downstream key.
func (km *KeyMap) Register(upstreamBase, upstreamKey, label, downstreamKey string) string {
	if downstreamKey == "" {
		downstreamKey = NewDownstreamKey()
	}
	km.mu.Lock()
	km.entries[downstreamKey] = &Entry{
		UpstreamBase: upstreamBase,
		UpstreamKey:  upstreamKey,
		Label:        label,
		CreatedAt:    time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	km.mu.Unlock()
	km.Save()
	return downstreamKey
}

// Delete removes a key mapping.
func (km *KeyMap) Delete(downstreamKey string) bool {
	km.mu.Lock()
	_, existed := km.entries[downstreamKey]
	delete(km.entries, downstreamKey)
	km.mu.Unlock()
	if existed {
		km.Save()
	}
	return existed
}

// List returns all entries (safe copy).
func (km *KeyMap) List() map[string]*Entry {
	km.mu.RLock()
	defer km.mu.RUnlock()
	out := make(map[string]*Entry, len(km.entries))
	for k, v := range km.entries {
		cp := *v
		out[k] = &cp
	}
	return out
}

// Import merges or replaces entries.
func (km *KeyMap) Import(data map[string]*Entry, merge bool) int {
	km.mu.Lock()
	if !merge {
		km.entries = make(map[string]*Entry)
	}
	for k, v := range data {
		km.entries[k] = v
	}
	count := len(data)
	km.mu.Unlock()
	km.Save()
	return count
}

// Count returns the number of registered keys.
func (km *KeyMap) Count() int {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return len(km.entries)
}

// Export returns the raw entries map for JSON export.
func (km *KeyMap) Export() map[string]*Entry {
	return km.List()
}

// NewDownstreamKey generates a new sk-gw-xxx key.
func NewDownstreamKey() string {
	b := make([]byte, 36)
	rand.Read(b)
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > 48 {
		s = s[:48]
	}
	return "sk-gw-" + s
}
