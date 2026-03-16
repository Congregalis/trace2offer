package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrProviderNotFound = errors.New("model provider not found")

// Message is an LLM input message.
type Message struct {
	Role    string
	Content string
}

// Request is a model generation request.
type Request struct {
	Model    string
	Messages []Message
}

// Response is the raw model output.
type Response struct {
	Content string
}

// Provider is the runtime abstraction for model backends.
type Provider interface {
	Name() string
	Generate(ctx context.Context, request Request) (Response, error)
}

// Manager manages registered model providers.
type Manager struct {
	defaultProvider string
	providers       map[string]Provider
}

func NewManager(defaultProvider string, providers ...Provider) (*Manager, error) {
	manager := &Manager{
		defaultProvider: strings.TrimSpace(defaultProvider),
		providers:       map[string]Provider{},
	}

	for _, item := range providers {
		if err := manager.Register(item); err != nil {
			return nil, err
		}
	}

	if manager.defaultProvider == "" && len(manager.providers) == 1 {
		for name := range manager.providers {
			manager.defaultProvider = name
		}
	}

	if manager.defaultProvider == "" {
		return nil, errors.New("default model provider is required")
	}
	if _, ok := manager.providers[manager.defaultProvider]; !ok {
		return nil, fmt.Errorf("default provider %q not registered", manager.defaultProvider)
	}

	return manager, nil
}

func (m *Manager) Register(item Provider) error {
	if m == nil {
		return errors.New("provider manager is nil")
	}
	if item == nil {
		return errors.New("provider is nil")
	}

	name := strings.TrimSpace(item.Name())
	if name == "" {
		return errors.New("provider name is required")
	}
	if _, exists := m.providers[name]; exists {
		return fmt.Errorf("provider already exists: %s", name)
	}
	m.providers[name] = item
	return nil
}

func (m *Manager) Get(name string) (Provider, error) {
	if m == nil {
		return nil, ErrProviderNotFound
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = m.defaultProvider
	}

	item, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return item, nil
}

func (m *Manager) Default() (Provider, error) {
	return m.Get(m.defaultProvider)
}
