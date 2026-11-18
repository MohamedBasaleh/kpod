package state

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/MohamedBasaleh/kpod/internal/atomicfile"
	"github.com/MohamedBasaleh/kpod/internal/config"
	"gopkg.in/yaml.v3"
)

const filePerm os.FileMode = 0o600

var ErrNotExist = errors.New("state not found")

var fileLocks sync.Map // state file path -> *sync.Mutex

func lockFor(path string) *sync.Mutex {
	v, _ := fileLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// Repository loads and saves state.yaml.
type Repository struct {
	paths config.Paths
}

func NewRepository(paths config.Paths) Repository {
	return Repository{paths: paths}
}

func (r Repository) Load() (State, error) {
	mu := lockFor(r.paths.StateFile())
	mu.Lock()
	defer mu.Unlock()
	return r.loadUnlocked()
}

func (r Repository) Save(s State) error {
	mu := lockFor(r.paths.StateFile())
	mu.Lock()
	defer mu.Unlock()
	return r.saveUnlocked(s)
}

// LoadOrDefault returns Default() when the file is missing.
func (r Repository) LoadOrDefault() (State, error) {
	s, err := r.Load()
	if errors.Is(err, ErrNotExist) {
		return Default(), nil
	}
	return s, err
}

// Mutate loads state (or Default), applies fn, and saves atomically.
func (r Repository) Mutate(fn func(*State) error) error {
	if fn == nil {
		return nil
	}
	mu := lockFor(r.paths.StateFile())
	mu.Lock()
	defer mu.Unlock()
	s, err := r.loadUnlocked()
	if errors.Is(err, ErrNotExist) {
		s = Default()
	} else if err != nil {
		return err
	}
	if err := fn(&s); err != nil {
		return err
	}
	return r.saveUnlocked(s)
}

func (r Repository) loadUnlocked() (State, error) {
	data, err := os.ReadFile(r.paths.StateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, ErrNotExist
		}
		return State{}, fmt.Errorf("read state: %w", err)
	}
	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return State{}, fmt.Errorf("malformed YAML in %s: %w", r.paths.StateFile(), err)
	}
	normalize(&s)
	if err := Validate(s); err != nil {
		return State{}, err
	}
	return s, nil
}

func (r Repository) saveUnlocked(s State) error {
	normalize(&s)
	if err := Validate(s); err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("serialize state: %w", err)
	}
	return atomicfile.Write(r.paths.StateFile(), data, filePerm)
}

func Validate(s State) error {
	if s.Version != Version {
		return fmt.Errorf("unsupported state version %d (supported: %d)", s.Version, Version)
	}
	return nil
}

func normalize(s *State) {
	if s.SeenContexts == nil {
		s.SeenContexts = []string{}
	}
	if s.SeenNamespaces == nil {
		s.SeenNamespaces = map[string][]string{}
	}
}
