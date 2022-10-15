package izidic

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	s1 = func(c *Container) (any, error) {
		return "s1", nil
	}
	s2 = func(c *Container) (any, error) {
		s1, err := c.Service("s1")
		if err != nil {
			return nil, fmt.Errorf("could not get service s1: %w", err)
		}
		return s1.(string) + "s2", nil
	}
)

func TestContainer_Param(t *testing.T) {
	type kvs []struct {
		k string
		v any
	}
	tests := [...]struct {
		name         string
		stored       kvs
		expectations kvs
	}{
		{"happy", kvs{{"k", "v"}}, kvs{{"k", "v"}}},
		{"overwrite", kvs{{"k", "v"}, {"k", "w"}}, kvs{{"k", "w"}}},
		{"missing", nil, kvs{{"k", nil}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dic := New()
			for _, kv := range test.stored {
				dic.Store(kv.k, kv.v)
			}
			for _, expectation := range test.expectations {
				actual, _ := dic.Param(expectation.k)
				if actual != expectation.v {
					t.Errorf("got %#v for key %q, but expected %#v",
						actual, expectation.k, expectation.v)
				}
			}
		})
	}
}

func TestContainer_MustParam(t *testing.T) {
	const expectedFormat = "parameter not found: %q"
	defer func() {
		rec := recover()
		actual, ok := rec.(error)
		if !ok {
			t.Fatalf("got %#v, but expected an error", rec)
		}
		expected := fmt.Sprintf(expectedFormat, "k2")
		if actual.Error() != expected {
			t.Fatalf("got %q, but expected %q", actual.Error(), expected)
		}
	}()
	dic := New()
	// Happy path
	dic.Store("k", "v")
	actual := dic.MustParam("k").(string)
	if actual != "v" {
		t.Fatalf("got %#v, but expected %q", actual, "v")
	}

	// Sad path
	dic.MustParam("k2")
}

func TestContainer_Service(t *testing.T) {
	const expected = "s1s2"
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

func TestContainer_MustService_Missing(t *testing.T) {
	const expectedFormat = "service not found: %q"
	defer func() {
		rec := recover()
		actual, ok := rec.(error)
		if !ok {
			t.Fatalf("got %#v, but expected an error", rec)
		}
		expected := fmt.Sprintf(expectedFormat, "k2")
		if actual.Error() != expected {
			t.Fatalf("got %q, but expected %q", actual.Error(), expected)
		}
	}()
	dic := New()
	// Happy path
	s := func(*Container) (any, error) { return 42, nil }
	dic.Register("s", s)
	actual := dic.MustService("s").(int)
	expected, _ := s(dic)
	if actual != expected {
		t.Fatalf("got %#v, but expected %#v", actual, expected)
	}

	// Sad path
	dic.MustService("k2")
}

func TestContainer_Service_Failing(t *testing.T) {
	instErr := errors.New("failed")
	s := func(dic *Container) (any, error) {
		return nil, instErr
	}
	dic := New()
	dic.Register("s", s)
	actualService, err := dic.Service("s")
	if actualService != nil {
		t.Errorf("got service %#v, but expected nil", actualService)
	}
	actualErr := err.Error()
	expected := fmt.Sprintf("failed instantiating service %s: %s", "s", instErr)
	if actualErr != expected {
		t.Errorf("got error %q but expected %q", actualErr, expected)
	}

}

func TestContainer_Service_Reuse(t *testing.T) {
	const name = "s"
	counter := 0
	service := func(dic *Container) (any, error) {
		counter++
		return counter, nil
	}

	dic := New()
	dic.Register(name, service)
	actual := dic.MustService(name).(int)
	if actual != 1 {
		t.Fatalf("got %d but expected 1", actual)
	}
	actual = dic.MustService(name).(int)
	if actual != 1 {
		t.Fatalf("got %d but expected 1", actual)
	}
}

func TestContainer_Names(t *testing.T) {
	var (
		vpt *string
		vt  string
	)
	dic := New()
	dic.Store("p1", vt)
	dic.Store("p2", vpt)
	dic.Register("s1", s1)
	dic.Register("s2", s2)

	actual := dic.Names()
	expected := map[string][]string{
		"params":   {"p1", "p2"},
		"services": {"s1", "s2"},
	}
	if !cmp.Equal(actual, expected) {
		t.Logf("unequal results: %v", cmp.Diff(actual, expected))
	}
}

func TestContainer_Freeze(t *testing.T) {
	tests := [...]struct {
		name     string
		attempt  func(*Container)
		expected string
	}{
		{"register", func(dic *Container) { dic.Register("p", nil) }, "Cannot register services on frozen container"},
		{"store", func(dic *Container) { dic.Store("p", "v") }, "Cannot store parameters on frozen container"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				rec := recover()
				msg, ok := rec.(string)
				if !ok {
					t.Fatalf("recovered a non-string: %#v", rec)
				}
				if msg != test.expected {
					t.Fatalf("Got %s but expected %s", msg, test.expected)
				}
			}()
			dic := New()
			dic.Freeze()
			test.attempt(dic)
		})
	}
}
