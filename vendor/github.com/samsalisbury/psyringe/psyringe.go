/*
Package psyringe provides an easy to use, lazy and concurrent dependency
injector.

Psyringe makes dependency injection very easy for well-written Go code. It
uses Go's type system to decide what to inject, and uses channels to orchestrate
value construction, automatically being as concurrent as your dependency graph
allows.

Psyringe does not rely on messy struct field tags nor verbose graph construction
syntax. It is very flexible and has a small interface, allowing you to tailor
things like scopes and object lifetimes very easily using standard Go code.

The example tests should speak for themselves, but if you want a deeper
explanation of how Psyringe works, read on.

Injection Type

Constructors and values added to psyringe have an implicit "injection type".
This is the type of value that constructor or value represents in the graph. For
non-constructor values, the injection type is the type of the value itself,
determined by reflect.GetType(). For constructors, it is the type of the first
output (return) value. It is important to understand this concept, since a
single psyringe can have only one value or constructor per injection type.

Constructors

Go does not have an explicit concept of "constructor". In Psyringe, constructors
are defined as any function that returns either a single value, or two values
where the second is an error. They can have any number of input parameters.

How Injection Works

A Psyringe knows how to populate fields in a struct with values of any injection
type that has been added to it.

When called upon to generate a value, via a call to Inject, the Psyringe
implicitly constructs a directed acyclic graph (DAG) from the constructors and
values, channelling values of each injection type into the relevant parameter
of any constructors which require it, and ultimately into any fields of that
type in the target struct which require it.

For a given Psyringe, each constructor will be called at most once. After that,
the generated value is provided directly without calling the constructor again.
Thus every value in a Psyringe is effectively a singleton. The Clone method
allows taking snapshots of a Psyringe in order to re-use its constructor graph
whilst generating new values. It is idiomatic to use multiple Psyringes with
differing scopes to inject different fields into the same object.
*/
package psyringe

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"sync"

	"github.com/pkg/errors"
)

// Psyringe is a dependency injection container.
type Psyringe struct {
	parent         *Psyringe
	scope          string
	injectionTypes *injectionTypes
	Hooks          Hooks
	allowAddCycle  bool
}

// New creates a new Psyringe, and adds the provided constructors and values to
// it. New will panic if any two arguments have the same injection type. See
// package level documentation for definition of "injection type".
func New(constructorsAndValues ...interface{}) *Psyringe {
	p := newPsyringe()
	if err := p.addErr(constructorsAndValues...); err != nil {
		panic(err)
	}
	return p
}

// newPsyringe is used to initialise a new Psyringe.
func newPsyringe() *Psyringe {
	return &Psyringe{
		scope:          "<root>",
		injectionTypes: newInjectionTypes(),
		Hooks:          newHooks(),
	}
}

// NewErr is similar to New, but returns an error instead of panicking. This is
// useful if you are dynamically generating the arguments.
func NewErr(constructorsAndValues ...interface{}) (*Psyringe, error) {
	p := newPsyringe()
	return p, p.addErr(constructorsAndValues...)
}

// Add adds constructors and values to the Psyringe. It panics if any
// constructor or value has the same injection type as any other already Added
// to this Psyringe or its ancestors (see Scope). See package documentation for
// definition of "injection type".
//
// Add uses reflection to determine whether each passed value is a constructor
// or not. For each constructor, it then generates a generic function in terms
// of reflect.Values ready to be used by a call to Inject. As such, Add is a
// relatively expensive call. See Clone for how to avoid calling Add too often.
func (p *Psyringe) Add(constructorsAndValues ...interface{}) {
	if err := p.addErr(constructorsAndValues...); err != nil {
		panic(err)
	}
}

// AddErr is similar to Add, but returns an error instead of panicking. This is
// useful if you are dynamically generating the arguments.
func (p *Psyringe) AddErr(constructorsAndValues ...interface{}) error {
	return p.addErr(constructorsAndValues...)
}

// addErr just exists to make callerinfo consistent in Psyringe.add.
func (p *Psyringe) addErr(constructorsAndValues ...interface{}) error {
	for i, thing := range constructorsAndValues {
		if thing == nil {
			return fmt.Errorf("cannot add nil (argument %d)", i)
		}
		if err := p.add(thing); err != nil {
			return err
		}
	}
	return nil
}

func (p *Psyringe) add(thing interface{}) error {
	v := reflect.ValueOf(thing)
	t := v.Type()
	if c := newCtor(t, v); c != nil {
		return errors.Wrapf(p.addCtor(c), "adding constructor %s failed", c.funcType)
	}
	return errors.Wrapf(p.addValue(t, v), "adding %s value failed", t)
}

// Clone returns a clone of this Psyringe.
//
// Clone exists to provide efficiency by allowing you to Add constructors and
// values once, and then invoke them multiple times for different instances.
// This is especially important in long-running applications where the cost of
// calling Add or New repeatedly may get expensive.
func (p *Psyringe) Clone() *Psyringe {
	q := *p
	q.injectionTypes = p.injectionTypes.Clone()
	return &q
}

// Inject takes a list of targets, which must be pointers to structs. It
// tries to inject a value for each field in each target, if a value is known
// for that field's type. All targets, and all fields in each target, are
// resolved concurrently where the graph allows. In the instance that the
// Psyringe knows no injection type for a given field's type, that field is
// passed over, leaving it with whatever value it already had.
//
// See package documentation for details on how a Psyringe injects values.
func (p *Psyringe) Inject(targets ...interface{}) error {
	wg := sync.WaitGroup{}
	wg.Add(len(targets))
	errs := make(chan error)
	go func() {
		wg.Wait()
		close(errs)
	}()
	for _, t := range targets {
		go func(target interface{}) {
			defer wg.Done()
			if err := p.inject(target); err != nil {
				errs <- errors.Wrapf(err, "inject into %T target failed", target)
			}
		}(t)
	}
	return <-errs
}

// MustInject wraps Inject and panics if Inject returns an error.
func (p *Psyringe) MustInject(targets ...interface{}) {
	if err := p.Inject(targets...); err != nil {
		panic(err)
	}
}

// Test checks that all constructors' parameters are satisfied within this
// Psyringe, and that there are no dependency cycles.
// This method can be used in your own tests to ensure you have a complete
// acyclic graph. Generally it is not recommended to use Test outside of your
// tests, as it is not built for speed.
func (p *Psyringe) Test() error {
	// Get sorted types - as this is a test better to have consistent output.
	ctors := p.injectionTypes.AddedAsCtors()
	ctorTypes := ctors.Keys()
	for _, outType := range ctorTypes {
		c := ctors.GetOrNil(outType).Ctor
		if err := c.testParametersAreRegisteredIn(p); err != nil {
			return errors.Wrapf(err, "unable to satisfy constructor %s", c.funcType)
		}
	}
	for _, outType := range ctorTypes {
		c := ctors.GetOrNil(outType).Ctor
		s := seen{}
		if err := p.detectCycle(s, c); err != nil {
			return errors.Wrapf(err, "dependency cycle: %s", outType)
		}
	}
	return nil
}

// Scope creates a child psyringe with p as its parent. Calls to Clone on this
// Psyringe will clone everything added directly to the child, but they will all
// share a reference to p. The name parameter is used for error messages only,
// to aid debugging.
//
// One use of Scope is to allow per-request constructors to be cloned on each
// request cheaply, whilst allowing those constructors access to all the values
// in the parent graph, p.
//
// Scope panics if the name is already used by this psyringe's parents, or any
// of its parents, recursively.
func (p *Psyringe) Scope(name string) (child *Psyringe) {
	if p.scopeNameInUse(name) {
		panic(fmt.Errorf("scope %q already defined", name))
	}
	q := New()
	q.parent = p
	q.scope = name
	q.Hooks = q.parent.Hooks
	return q
}

type seen map[reflect.Type]struct{}

func (s seen) clone() seen {
	t := make(seen, len(s))
	for k := range s {
		t[k] = struct{}{}
	}
	return t
}

// detectCycle returns an error if constructing rootType depends on rootType
// transitively.
func (p *Psyringe) detectCycle(s seen, c *ctor) error {
	// We have now seen the injection type of c.
	s = s.clone()
	s[c.outType] = struct{}{}
	for _, t := range c.inTypes {
		if _, ok := s[t]; ok {
			return fmt.Errorf("depends on %s", t)
		}
		c, ok := p.injectionTypes.AddedAsCtors().Get(t)
		if !ok {
			continue
		}
		if err := p.detectCycle(s, c.Ctor); err != nil {
			return errors.Wrapf(err, "depends on %s", t)
		}
	}
	return nil
}

// inject just tries to inject a value for each field in target, no errors if it
// doesn't know how to inject a value for a given field's type, those fields are
// just left as-is.
func (p *Psyringe) inject(target interface{}) error {
	v := reflect.ValueOf(target)
	ptr := v.Type()
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	t := ptr.Elem()
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}
	if v.IsNil() {
		return fmt.Errorf("target is nil")
	}
	debugf("injecting into a %s", ptr)
	nfs := t.NumField()
	wg := sync.WaitGroup{}
	wg.Add(nfs)
	errs := make(chan error)
	go func() {
		wg.Wait()
		close(errs)
	}()
	for i := 0; i < nfs; i++ {
		go func(f reflect.Value, field reflect.StructField) {
			defer wg.Done()
			if field.PkgPath != "" {
				debugf("not injecting unexported field %s.%s (%s)", ptr, field.Name, field.Type)
				return
			}
			debugf("injecting field %s.%s (%s)", ptr, field.Name, field.Type)
			parentName := fmt.Sprintf("%T", target)
			if fv, ok, err := p.getValueForStructField(p.Hooks, parentName, field); ok && err == nil {
				f.Set(fv)
			} else if err != nil {
				errs <- err
			}
			// If !ok there is no value for this field type, that's OK continue.
		}(v.Elem().Field(i), t.Field(i))
	}

	return <-errs
}

func (p *Psyringe) getValueForStructField(leafHooks Hooks, parentTypeName string, field reflect.StructField) (reflect.Value, bool, error) {
	t := field.Type
	name := field.Name
	if v, ok := p.injectionTypes.AddedAsValues().Get(t); ok {
		// We have a value, return it.
		return v.Value, true, nil
	}
	if c, ok := p.injectionTypes.AddedAsCtors().Get(t); ok {
		// We have a constructor, call it.
		v, err := c.Ctor.getValue(p)
		return v, true, errors.Wrapf(err, "getting field %s (%s) failed", name, t)
	}
	// Look in higher scopes.
	if p.parent != nil {
		// We have a parent, so try to get the value from there.
		return p.parent.getValueForStructField(leafHooks, parentTypeName, field)
	}
	// We have no value, constructor, nor parent. Give up.
	return reflect.Value{}, false, leafHooks.NoValueForStructField(parentTypeName, field)
}

func (p *Psyringe) getValueForConstructor(forCtor *ctor, paramIndex int, t reflect.Type) (reflect.Value, error) {
	debugf("getting a %s for arg %d for constructor of %s", t, paramIndex, forCtor.outType)
	if v, ok := p.injectionTypes.WithRealisedValues().Get(t); ok {
		return v.Value, nil
	}
	c, ok := p.injectionTypes.AddedAsCtors().Get(t)
	if !ok {
		return reflect.Value{}, errors.Errorf("no constructor or value for %s", t)
	}
	v, err := c.Ctor.getValue(p)
	return v, errors.Wrapf(err, "getting argument %d failed", paramIndex)
}

func (p *Psyringe) addCtor(c *ctor) error {
	return p.registerInjectionType(c.outType, &injectionType{Ctor: c})
}

func (p *Psyringe) addValue(t reflect.Type, v reflect.Value) error {
	return p.registerInjectionType(t, &injectionType{Value: v})
}

func (p *Psyringe) injectionTypeIsRegisteredAtThisScope(t reflect.Type) bool {
	_, registered := p.injectionTypes.Get(t)
	return registered
}

func (p *Psyringe) injectionTypeRegistrationScope(t reflect.Type) (*Psyringe, bool) {
	if p.injectionTypes.Contains(t) {
		return p, true
	}
	if p.parent == nil {
		return nil, false
	}
	return p.parent.injectionTypeRegistrationScope(t)
}

func (p *Psyringe) scopeNameInUse(name string) bool {
	if p.scope == name {
		return true
	}
	if p.parent == nil {
		return false
	}
	return p.parent.scopeNameInUse(name)
}

func (p *Psyringe) registerInjectionType(t reflect.Type, it *injectionType) error {
	if scopedPsyringe, registered := p.injectionTypeRegistrationScope(t); registered {
		message := fmt.Sprintf("injection type %s already registered at %s",
			t, scopedPsyringe.injectionTypes.GetOrNil(t).DebugAddedLocation)
		if scopedPsyringe.scope == p.scope {
			return errors.New(message)
		}
		return fmt.Errorf("%s (scope %s)", message, scopedPsyringe.scope)
	}
	_, file, line, _ := runtime.Caller(5)
	it.DebugAddedLocation = fmt.Sprintf("%s:%d", file, line)
	if err := p.injectionTypes.Add(t, it); err != nil {
		return err
	}
	if p.allowAddCycle || it.Ctor == nil {
		return nil
	}
	return errors.Wrapf(p.detectCycle(seen{}, it.Ctor),
		"dependency cycle: %s", it.Ctor.outType)
}

func (p *Psyringe) testValueOrConstructorIsRegistered(paramType reflect.Type) error {
	if p.injectionTypes.Contains(paramType) {
		return nil
	}
	return errors.Errorf("no constructor or value for %s", paramType)
}

var debugf = func(string, ...interface{}) {}

const debugFileKey = "PSYRINGE_DEBUG_FILE"

func init() {
	if debugFile := os.Getenv(debugFileKey); debugFile != "" {
		if fd, err := os.Create(debugFile); err != nil {
			log.Printf("psyringe: Unable to open %q (set by %s)", debugFile, debugFileKey)
		} else {
			l := log.New(fd, "psyringe: DEBUG: ", log.LstdFlags)
			debugf = func(format string, a ...interface{}) {
				l.Printf(format, a...)
			}
		}
	}
}
