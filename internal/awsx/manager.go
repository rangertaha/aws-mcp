// SPDX-License-Identifier: MIT

// Package awsx owns AWS SDK v2 configuration and clients for aws-mcp. It is
// named awsx (not aws) to avoid colliding with aws-sdk-go-v2's own "aws"
// package throughout this codebase.
//
// Credentials are never read directly here: aws-mcp uses the standard AWS
// credential chain (environment, shared config/credentials files, SSO, or an
// attached IAM role) via aws-sdk-go-v2, resolved per active profile by
// Manager.
package awsx

import (
	"context"
	"fmt"
	"sync"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
)

// Manager resolves AWS configuration for the active profile and lazily
// builds/caches SDK service clients from a registry.ClientFactory map. All
// methods are safe for concurrent use.
type Manager struct {
	mu        sync.RWMutex
	factories map[string]registry.ClientFactory
	region    string
	profile   string
	configs   map[string]awssdk.Config  // profile -> resolved config
	clients   map[string]map[string]any // profile -> service -> client
}

// NewManager creates a Manager. region overrides the credential-chain region
// when non-empty; profile is the initially active profile ("" means the SDK's
// own default resolution, not necessarily a profile literally named
// "default").
func NewManager(factories map[string]registry.ClientFactory, region, profile string) *Manager {
	return &Manager{
		factories: factories,
		region:    region,
		profile:   profile,
		configs:   make(map[string]awssdk.Config),
		clients:   make(map[string]map[string]any),
	}
}

// Profile returns the currently active profile name.
func (m *Manager) Profile() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.profile
}

// Config returns the resolved aws.Config for the active profile, loading and
// caching it on first use.
func (m *Manager) Config(ctx context.Context) (awssdk.Config, error) {
	m.mu.RLock()
	profile := m.profile
	cfg, ok := m.configs[profile]
	m.mu.RUnlock()
	if ok {
		return cfg, nil
	}
	return m.loadConfig(ctx, profile)
}

// loadConfig resolves and caches the config for the given profile, regardless
// of which profile is currently active.
func (m *Manager) loadConfig(ctx context.Context, profile string) (awssdk.Config, error) {
	var opts []func(*awsconfig.LoadOptions) error
	if m.region != "" {
		opts = append(opts, awsconfig.WithRegion(m.region))
	}
	if profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(profile))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return awssdk.Config{}, fmt.Errorf("loading AWS config (profile=%q): %w", profile, err)
	}

	m.mu.Lock()
	m.configs[profile] = cfg
	m.mu.Unlock()
	return cfg, nil
}

// Client returns the SDK client for the named service (as registered in the
// factory map), lazily built and cached per active profile.
func (m *Manager) Client(ctx context.Context, service string) (any, error) {
	factory, ok := m.factories[service]
	if !ok {
		return nil, fmt.Errorf("unknown AWS service %q", service)
	}

	m.mu.RLock()
	profile := m.profile
	client, ok := m.clients[profile][service]
	m.mu.RUnlock()
	if ok {
		return client, nil
	}

	cfg, err := m.Config(ctx)
	if err != nil {
		return nil, err
	}
	client = factory(cfg)

	m.mu.Lock()
	if m.clients[profile] == nil {
		m.clients[profile] = make(map[string]any)
	}
	m.clients[profile][service] = client
	m.mu.Unlock()

	return client, nil
}

// UseProfile switches the active profile, eagerly resolving its
// configuration so an unknown or misconfigured profile fails fast rather than
// on first use. An empty name reverts to the SDK's default resolution.
func (m *Manager) UseProfile(ctx context.Context, profile string) error {
	if profile != "" {
		names, err := ListProfiles()
		if err != nil {
			return fmt.Errorf("listing AWS profiles: %w", err)
		}
		found := false
		for _, n := range names {
			if n == profile {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown AWS profile %q", profile)
		}
	}

	if _, err := m.loadConfig(ctx, profile); err != nil {
		return err
	}

	m.mu.Lock()
	m.profile = profile
	m.mu.Unlock()
	return nil
}
