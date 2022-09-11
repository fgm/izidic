// Package izidic defines a tiny dependency injection container,
// loosely inspired by a subset of silexphp/pimple.
//
// Like Pimple, the basic feature is that storing service definitions does not
// create instances, allowing users to store definitions of services requiring
// other services before those are actually defined.
package izidic

import (
	"errors"
	"fmt"
	"sync"
)

// Service is the type used to define container serviceDefs accessors.
//
// It takes an instance of the container and returns an instance of the desired service,
// which should then be type-asserted before use.
//
// Any access to a service from the container returns the same instance.
type Service func(dic *Container) (any, error)

// Container is the container, holding both parameters and services
type Container struct {
	sync.Mutex
	parameters  map[string]any
	serviceDefs map[string]Service
	services    map[string]any
}

// Register registers a service with the container.
func (dic *Container) Register(name string, fn Service) {
	dic.Lock()
	defer dic.Unlock()
	dic.serviceDefs[name] = fn
}

// Store stores a parameter in the container.
func (dic *Container) Store(name string, param any) {
	dic.Lock()
	defer dic.Unlock()
	dic.parameters[name] = param
}

// Service returns the single instance of the requested service on success.
func (dic *Container) Service(name string) (any, error) {
	dic.Lock()
	defer dic.Unlock()

	// Reuse existing instance if any.
	instance, found := dic.services[name]
	if found {
		return instance, nil
	}

	// Otherwise instantiate.
	service, found := dic.serviceDefs[name]
	if !found {
		return nil, fmt.Errorf("service %s not found", name)
	}
	instance, err := service(dic)
	if err != nil {
		return nil, fmt.Errorf("failed instantiating service %s: %w", name, err)
	}
	dic.services[name] = instance

	return instance, nil
}

func (dic *Container) MustService(name string) any {
	instance, err := dic.Service(name)
	if err != nil {
		panic(err)
	}
	return instance
}

func (dic *Container) Param(name string) (any, error) {
	dic.Lock()
	defer dic.Unlock()
	p, found := dic.parameters[name]
	if !found {
		return nil, errors.New("parameter not found")
	}
	return p, nil
}

func (dic *Container) MustParam(name string) any {
	p, err := dic.Param(name)
	if err != nil {
		panic(err)
	}
	return p
}
