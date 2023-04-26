package di

import "log"

// App represents whatever an actual application as a function would be.
type App func() error

func makeAppFeature(name string, logger *log.Logger) App {
	return func() error {
		logger.Println(name)
		return nil
	}
}
