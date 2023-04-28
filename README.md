# IziDIC

[![Tests](https://github.com/fgm/izidic/actions/workflows/go.yml/badge.svg)](https://github.com/fgm/izidic/actions/workflows/go.yml)
[![CodeQL](https://github.com/fgm/izidic/actions/workflows/codeql.yml/badge.svg)](https://github.com/fgm/izidic/actions/workflows/codeql.yml)
[![codecov](https://codecov.io/gh/fgm/izidic/branch/main/graph/badge.svg?token=R5BMHL3CSH)](https://codecov.io/gh/fgm/izidic)
[![Go Report Card](https://goreportcard.com/badge/github.com/fgm/container)](https://goreportcard.com/report/github.com/fgm/container)


## Description

Package [izidic](https://github.com/fgm/izidic) defines a tiny dependency injection container for Go projects.

That container can hold two different kinds of data:

- parameters, which are mutable data without any dependency;
- services, which are functions returning a typed object providing a feature,
  and may depend on other services and parameters.

The basic feature is that storing service definitions does not create instances,
allowing users to store definitions of services requiring other services,
before those are actually defined.

Notice that parameters do not need to be primitive types.
For instance, most applications are likely to store a `stdout` object with value `os.Stdout`.

Unlike heavyweights like google/wire or uber/zap, it works as a single step,
explicit process, without reflection or code generation, to keep everything in sight.

## Usage

### Setup and use

| Step                                | Code examples                           |
|:------------------------------------|-----------------------------------------|
| Import the package                  | `import "github.com/fgm/izidic"`        |
| Initialize a container              | `dic := izidic.New()`                   |
| Store parameters in the DIC         | `dic.Store("executable", os.Args[0])`   |
| Register services with the DIC      | `dic.Register("logger", loggerService)` |
| Freeze the container                | `dic.Freeze()`                          |
| Read a parameter from the DIC       | `p, err := dic.Param(name)`             |
| Get a service instance from the DIC | `s, err := dic.Service(name)`           |

Freezing applies once all parameters and services are stored and registered,
and enables concurrent access to the container.


## Defining parameters

Parameters can be any value type. They can be stored in the container in any order.


## Writing services

Services like `loggerService` in the previous example are instances ot the `Service` type,
which is defined as:

`type Service func(Container) (any, error)`

- Services can reference any other service and parameters from the container, to return the instance they
  build. The only restriction is that cycles are not supported.
- Like parameters, services can be registered in any order on the container,
  so feel free to order the registrations in alphabetical order for readability.
- Services are lazily instantiated on the first actual use: subsequent references will reuse the same instance. 
  This means that the service function is a good place to perform one-time operations
  needed for configuration related to the service, like initializing the
  default `log` logger while building a logger service with `log.SetOutput()`.


### Accessing the container

- Parameter access: `s, err := dic.Param("name")`
  - Check the error against `nil`
  - Type-assert the parameter value: `name, ok := s.(string)`
  - Or use shortcut: `name := dic.MustParam("name").(string)` 
- Service access: `s, err := dic.Service("logger")`
  - Check the error against `nil`
  - Type-assert the service instance value: `logger, ok := s.(*log.Logger)`
  - Or use shortcut: `logger := dic.MustService("logger").(*log.Logger)`


## Best practices
### Create a simpler developer experience

One limitation of having `Container.(Must)Param()` and `Container.(Must)Service()`
return untyped results as `any` is the need to type-assert results on every access.

To make this safer and better looking, a neat approach is to define an application
container type wrapping an `izidic.Container` and adding fully typed facade methods
as in this example:

```go
package di

import (
	"io"
	"log"

	"github.com/fgm/izidic"
)

type Container struct {
	izidic.Container
}

// Logger is a typed service accessor.
func (c *Container) Logger() *log.Logger { 
	return c.MustService("logger").(*log.Logger)
}

// Name is a types parameter accessor.
func (c *Container) Name() string {
	return c.MustParam("name").(string)
}

// loggerService is an izidic.Service also containing a one-time initialization action.
func loggerService(dic izidic.Container) (any, error) {
	w := dic.MustParam("writer").(io.Writer)
	log.SetOutput(w) // Support dependency code not taking an injected logger. 
	logger := log.New(w, "", log.LstdFlags)
	return logger, nil
}

func appService(dic izidic.Container) (any, error) {
	wdic := Container{dic}  // wrapped container with typed accessors
	logger := wdic.Logger() // typed service instance
	name := wdic.Name()     // typed parameter value
	appFeature := makeAppFeature(name, logger)
	return appFeature, nil
}

func Resolve(w io.Writer, name string, args []string) izidic.Container {
	dic := izidic.New()
	dic.Store("name", name)
	dic.Store("writer", w)
	dic.Register("logger", loggerService)
	dic.Register("app", appService)
	// others...
	dic.Freeze()
	return dic
}
```
 
These accessors will be useful when defining services, as in `appService` above,
or in the boot sequence, which typically needs at least a `logger` and one or
more application-domain service instances.


### Create the container in a `Resolve` function

The cleanest way to initialize a container is to have the
project contain an function, conventionally called `Resolve`, which takes all globals used in the project,
and returns an instance of the custom container type defined above, as in [examples/di/di.go](examples/di/di.go).


### Do not pass the container

Passing the container to application code, although it works, defines the "service locator" anti-pattern.

Because the container is a complex object with variable contents,
code receiving the container is hard to test.
That is the reason why receiving it is typically limited to `izidic.Service` functions,
which are simple-minded initializers that do not need testing.

Instead, in the service providing a given feature, use something like `appService`:
- obtain values from the container 
- pass them to a domain-level factory receiving exactly the typed arguments it needs and nothing more.

In most cases, the value obtained thus will be a `struct` or a `func`,
ready to be used without further data from the container.

See a complete demo in [examples/demo.go](examples/demo.go).
