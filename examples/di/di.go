package di

import (
	"io"
	"log"

	"github.com/fgm/izidic"
)

// Container is an application-specific wrapper for a basic izidic container,
// adding typed accessors for simpler use by application code, obviating the need
// for type assertions.
type Container struct {
	izidic.Container
}

// Logger is a typed service accessor.
func (c *Container) Logger() *log.Logger {
	return c.MustService("logger").(*log.Logger)
}

// Name is a typed parameter accessor.
func (c *Container) Name() string {
	return c.MustParam("name").(string)
}

// Resolve is the location where the parameters and services in the container
//
//	are assembled and the container readied for use.
func Resolve(w io.Writer, name string, args []string) izidic.Container {
	dic := izidic.New()
	dic.Store("name", name)
	dic.Store("writer", w)
	dic.Register("app", appService)
	dic.Register("logger", loggerService)
	dic.Freeze()
	return dic
}

func appService(dic izidic.Container) (any, error) {
	wdic := Container{dic}  // wrapped Container with typed accessors
	logger := wdic.Logger() // typed service instance: *log.Logger
	name := wdic.Name()     // typed parameter value: string
	appFeature := makeAppFeature(name, logger)
	return appFeature, nil
}

// loggerService is an izidic.Service also containing a one-time initialization action.
//
// Keep in mind that the initialization will only be performed once the service has
// actually been instantiated.
func loggerService(dic izidic.Container) (any, error) {
	w := dic.MustParam("writer").(io.Writer)
	log.SetOutput(w) // Support dependency code not taking an injected logger.
	logger := log.New(w, "", log.LstdFlags)
	return logger, nil
}
