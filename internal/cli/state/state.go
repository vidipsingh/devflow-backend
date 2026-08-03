package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const stateFile = ".devflow/state.json"

type State struct {
	Repo       string   `json:"repo"`
	Branch     string   `json:"branch"`
	Staged     []string `json:"staged"`
	Stash      []string `json:"stash"`
	LastCommit string   `json:"lastCommit"`
}

// StashFiles moves staged files into stash, clears staged
func StashFiles() error {
	s, err := Load()
	if err != nil { return err }
	if len(s.Staged) == 0 { return fmt.Errorf("nothing staged to stash") }
	s.Stash = append(s.Stash, s.Staged...)
	s.Staged = []string{}
	return Save(s)
}

// UnstashFiles moves stashed files back to staged, clears stash
func UnstashFiles() error {
	s, err := Load()
	if err != nil { return err }
	if len(s.Stash) == 0 { return fmt.Errorf("stash is empty") }
	s.Staged = append(s.Staged, s.Stash...)
	s.Stash = []string{}
	return Save(s)
}

func Load() (State, error) {
	data, err := os.ReadFile(stateFile)
	if os.IsNotExist(err) {
		return State{Branch: "main"}, nil
	}
	if err != nil {
		return State{}, err
	}
	var s State
	return s, json.Unmarshal(data, &s)
}

func Save(s State) error {
	if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0644)
}

func Exists() bool {
	_, err := os.Stat(stateFile)
	return err == nil
}

// Stage adds paths to the staged list (deduplicating)
func Stage(paths []string) error {
	s, err := Load()
	if err != nil { return err }
	existing := map[string]bool{}
	for _, p := range s.Staged { existing[p] = true }
	for _, p := range paths {
		if !existing[p] {
			s.Staged = append(s.Staged, p)
			existing[p] = true
		}
	}
	return Save(s)
}

// ClearStaged empties the staged list after a commit
func ClearStaged() error {
	s, err := Load()
	if err != nil { return err }
	s.Staged = []string{}
	return Save(s)
}
