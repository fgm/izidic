// Package izidic defines a tiny dependency injection container.
//
// The basic feature is that storing service definitions does not create instances,
// allowing users to store definitions of services requiring other services
// before those are actually defined.
//
// Container writes are not concurrency-safe, so they are locked with Container.Freeze()
// after the initial setup, which is assumed to be non-concurrent.
package izidic

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
)

// Service is the type used to define container serviceDefs accessors.
//
// It takes an instance of the container and returns an instance of the desired service,
// which should then be type-asserted before use.
//
// Any access to a service from the container returns the same instance.
type Service func(dic Container) (any, error)

// Container represents any implementation of a dependency injection container.
type Container interface {
	Freeze()
	MustParam(name string) any
	MustService(name string) any
	Names() map[string][]string
	Param(name string) (any, error)
	Register(name string, fn Service)
	Store(name string, param any)
	Service(name string) (any, error)
}

// container is the container, holding both parameters and services
type container struct {
	sync.RWMutex // Lock for service instances
	frozen       bool
	parameters   map[string]any
	serviceDefs  map[string]Service
	services     map[string]any
}

// Freeze converts the container from build mode, which does not support
// concurrency, to run mode, which does.
func (dic *container) Freeze() {
	dic.frozen = true
}

func (dic *container) MustParam(name string) any {
	p, err := dic.Param(name)
	if err != nil {
		panic(err)
	}
	return p
}

func (dic *container) MustService(name string) any {
	instance, err := dic.Service(name)
	if err != nil {
		panic(err)
	}
	return instance
}

// Names returns the names of all the parameters and instances defined on the container.
func (dic *container) Names() map[string][]string {
	dump := map[string][]string{
		"params":   make([]string, 0, len(dic.parameters)),
		"services": make([]string, 0, len(dic.serviceDefs)),
	}
	dic.RLock()
	defer dic.RUnlock()
	for k := range dic.parameters {
		dump["params"] = append(dump["params"], k)
	}
	sort.Strings(dump["params"])
	for k := range dic.serviceDefs {
		dump["services"] = append(dump["services"], k)
	}
	sort.Strings(dump["services"])
	return dump
}

func (dic *container) Param(name string) (any, error) {
	dic.RLock()
	defer dic.RUnlock()

	p, found := dic.parameters[name]
	if !found {
		return nil, fmt.Errorf("parameter not found: %q", name)
	}
	return p, nil
}

// Register registers a service with the container.
func (dic *container) Register(name string, fn Service) {
	if dic.frozen {
		panic("Cannot register services on frozen container")
	}
	dic.serviceDefs[name] = fn
}

// Service returns the single instance of the requested service on success.
func (dic *container) Service(name string) (any, error) {
	// Reuse existing instance if any.
	dic.RLock()
	instance, found := dic.services[name]
	dic.RUnlock()
	if found {
		return instance, nil
	}

	// Otherwise instantiate. No lock because no concurrent writes can happen:
	// - during build, recursive calls may happen, but not concurrently
	// - after freeze, no new services may be created: see container.Register
	service, found := dic.serviceDefs[name]
	if !found {
		return nil, fmt.Errorf("service not found: %q", name)
	}

	// Loop detection: if the call stack contains more calls to Service reaching
	// this step than there are services defined in the container, then resolution
	// for at least one service was attempted more than once, which implies a
	// dependency cycle.
	const funcName = "github.com/fgm/izidic.(*container).Service"
	// We need a vastly oversized value to cover the case of deeply nested dic.Service() calls.
	pcs := make([]uintptr, 1e6)
	n := runtime.Callers(1, pcs)
	pcs = pcs[:n]
	frames := runtime.CallersFrames(pcs)
	serviceCalls := 0
	for {
		frame, more := frames.Next()
		if frame.Func.Name() == funcName {
			serviceCalls++
		}
		if !more {
			break
		}
	}
	if serviceCalls > len(dic.serviceDefs) {
		return nil, errors.New("circular dependency detected")
	}

	instance, err := service(dic)
	if err != nil {
		return nil, fmt.Errorf("failed instantiating service %s: %w", name, err)
	}

	dic.Lock()
	defer dic.Unlock()
	dic.services[name] = instance

	return instance, nil
}

// Store stores a parameter in the container.
func (dic *container) Store(name string, param any) {
	if dic.frozen {
		panic("Cannot store parameters on frozen container")
	}
	dic.parameters[name] = param
}

// New creates a container ready for use.
func New() Container {
	return &container{
		RWMutex:     sync.RWMutex{},
		parameters:  make(map[string]any),
		serviceDefs: make(map[string]Service),
		services:    make(map[string]any),
	}
}
