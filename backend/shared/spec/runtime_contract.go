package spec

// Operation is runtime metadata derived from the TypeSpec HTTP contract.
type Operation struct {
	Method     string
	Path       string
	Deprecated bool
}

// RuntimeContract is the small generated subset of the language-independent
// contract that runtime endpoints need for self-description.
type RuntimeContract struct {
	Operations   map[string]Operation
	Deprecations map[string]Deprecation
}

// Deprecation carries the dates required for RFC 9745 and RFC 8594 headers.
// The key in RuntimeContract.Deprecations is a TypeSpec operation name.
type Deprecation struct {
	Since  string
	Sunset string
}

// CurrentRuntimeContract returns the contract compiled into this build.
func CurrentRuntimeContract() *RuntimeContract {
	return &RuntimeContract{Operations: generatedOperations, Deprecations: map[string]Deprecation{}}
}

func (c *RuntimeContract) Operation(name string) (Operation, bool) {
	if c == nil {
		return Operation{}, false
	}
	operation, ok := c.Operations[name]
	return operation, ok
}
