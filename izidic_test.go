package izidic

import (
	"fmt"
	"testing"
)

func TestContainer_Service(t *testing.T) {
	const expected = "s1s2"
	s1 := func(c *Container) (any, error) {
		return "s1", nil
	}
	s2 := func(c *Container) (any, error) {
		s1, err := c.Service("s1")
		if err != nil {
			return nil, fmt.Errorf("could not get service s1: %w", err)
		}
		return s1.(string) + "s2", nil
	}

	dic := New()
	dic.Register("s1", s1)
	dic.Register("s2", s2)
	s, err := dic.Service("s2")
	if err != nil {
		t.Fatal(err)
	}
	actual, ok := s.(string)
	if !ok {
		t.Fatalf("unexpected type for s2: %T", s)
	}
	if actual != expected {
		t.Fatalf("got %q but expected %q", actual, expected)
	}
}
