// Package favorites stores starred clips in the user's config folder (never on
// the card).
//
// Clip ids are "<category>/<stem>" (e.g. Normal/NO20261109-191423-003968). The
// stem embeds date, time and a sequence number, so ids stay stable across card
// re-mounts and drive letters.
package favorites

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store is the persisted set of starred clip ids.
type Store struct {
	Path  string
	mu    sync.Mutex
	items map[string]float64 // id -> starred-at epoch seconds
}

type file struct {
	Version int                `json:"version"`
	Items   map[string]float64 `json:"items"`
}

// Open loads the store at path (a missing or broken file means "no favorites").
func Open(path string) *Store {
	s := &Store{Path: path, items: map[string]float64{}}
	if data, err := os.ReadFile(path); err == nil {
		var f file
		if json.Unmarshal(data, &f) == nil && f.Items != nil {
			s.items = f.Items
		}
	}
	return s
}

// IDs returns the starred ids, sorted.
func (s *Store) IDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sortedLocked()
}

func (s *Store) sortedLocked() []string {
	out := make([]string, 0, len(s.items))
	for id := range s.items {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Set stars (on) or un-stars ids, saves, and returns the new list.
func (s *Store) Set(ids []string, on bool) ([]string, error) {
	now := float64(time.Now().UnixMilli()) / 1000
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		if id == "" || len(id) > 300 {
			continue
		}
		if !on {
			delete(s.items, id)
		} else if _, ok := s.items[id]; !ok {
			s.items[id] = now
		}
	}
	if err := s.save(); err != nil {
		return nil, err
	}
	return s.sortedLocked(), nil
}

// save writes atomically: a temp file in the same folder, then rename.
func (s *Store) save() error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(file{Version: 1, Items: s.items}, "", " ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".favorites-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.Path)
}
