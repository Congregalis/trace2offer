package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrToolNotFound = errors.New("tool not found")

// Definition describes a callable tool for the model.
type Definition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Tool is executable capability used by agent loop.
type Tool interface {
	Definition() Definition
	Run(ctx context.Context, input json.RawMessage) (string, error)
}

// Registry is a lightweight tool manager.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry(tools ...Tool) (*Registry, error) {
	registry := &Registry{tools: map[string]Tool{}}
	for _, item := range tools {
		if err := registry.Register(item); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(item Tool) error {
	if r == nil {
		return errors.New("tool registry is nil")
	}
	if item == nil {
		return errors.New("tool is nil")
	}

	definition := item.Definition()
	name := strings.TrimSpace(definition.Name)
	if name == "" {
		return errors.New("tool name is required")
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool already registered: %s", name)
	}

	r.tools[name] = item
	return nil
}

func (r *Registry) Definitions() []Definition {
	if r == nil {
		return nil
	}

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	definitions := make([]Definition, 0, len(names))
	for _, name := range names {
		definitions = append(definitions, r.tools[name].Definition())
	}
	return definitions
}

func (r *Registry) Call(ctx context.Context, name string, input json.RawMessage) (string, error) {
	if r == nil {
		return "", ErrToolNotFound
	}

	item, ok := r.tools[strings.TrimSpace(name)]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	return item.Run(ctx, input)
}
