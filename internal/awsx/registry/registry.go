// SPDX-License-Identifier: MIT

// Package registry builds a catalog of every AWS SDK v2 operation reachable
// on the configured service clients, purely via reflection over the compiled
// client types (see reflect.go) — no per-operation code is generated or
// hand-written. The catalog backs the aws-mcp meta tools (aws_list_services,
// aws_describe_operation, aws_invoke, ...) so the model can discover and call
// any AWS operation without every operation being registered as its own MCP
// tool.
package registry

import (
	"reflect"
	"sort"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
)

// ClientFactory builds a service client from a resolved aws.Config. Exactly
// one factory is registered per AWS service, in the generated
// zz_generated_clients.go.
type ClientFactory func(cfg awssdk.Config) any

// OperationSpec describes one discovered AWS SDK operation.
type OperationSpec struct {
	Service string
	Name    string // e.g. "ListBuckets"

	InputType  reflect.Type // the Input struct type (not a pointer)
	OutputType reflect.Type // the Output struct type (not a pointer)

	// Mutating reports whether the operation is believed to change AWS
	// state, based on a verb-prefix heuristic (see classify.go). Unknown
	// verbs default to Mutating=true: hiding a safe operation from
	// read-only mode is an acceptable false positive, allowing a mutating
	// call through is not.
	Mutating bool
	// Destructive further marks operations that delete/replace/disable
	// resources. Only meaningful when Mutating is true.
	Destructive bool

	// Unsupported marks an operation whose Input/Output shape can't be
	// safely handled by generic JSON-based dispatch (streaming bodies,
	// union/polymorphic fields, open-content document fields — see
	// unsupported.go). UnsupportedReason explains why.
	Unsupported       bool
	UnsupportedReason string

	// PaginationField names the Output field most likely to carry a
	// continuation token/marker, or "" if none was detected.
	PaginationField string
}

// ServiceSpec describes one discovered AWS service and its operations.
type ServiceSpec struct {
	Name       string
	Operations map[string]*OperationSpec
}

// OperationNames returns every operation name for the service, sorted.
func (s *ServiceSpec) OperationNames() []string {
	names := make([]string, 0, len(s.Operations))
	for n := range s.Operations {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Catalog is the full set of discovered services and operations.
type Catalog struct {
	Services map[string]*ServiceSpec
}

// Build constructs a Catalog by reflecting over one client instance per
// factory. Client construction and reflection require no credentials or
// network access, so a throwaway, unconfigured aws.Config is used — this
// catalog is purely descriptive and is independent of any real,
// profile-specific configuration used later for invocation.
func Build(factories map[string]ClientFactory) *Catalog {
	cat := &Catalog{Services: make(map[string]*ServiceSpec, len(factories))}

	cfg := awssdk.Config{}
	for name, factory := range factories {
		client := factory(cfg)
		spec := &ServiceSpec{Name: name, Operations: make(map[string]*OperationSpec)}
		for _, op := range discoverOperations(name, client) {
			spec.Operations[op.Name] = op
		}
		cat.Services[name] = spec
	}
	return cat
}

// ServiceNames returns every discovered service name, sorted.
func (c *Catalog) ServiceNames() []string {
	names := make([]string, 0, len(c.Services))
	for n := range c.Services {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Service looks up a service by name.
func (c *Catalog) Service(name string) (*ServiceSpec, bool) {
	s, ok := c.Services[name]
	return s, ok
}

// Operation looks up an operation by service and operation name.
func (c *Catalog) Operation(service, operation string) (*OperationSpec, bool) {
	s, ok := c.Services[service]
	if !ok {
		return nil, false
	}
	op, ok := s.Operations[operation]
	return op, ok
}
