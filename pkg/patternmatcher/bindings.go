package patternmatcher

import "fmt"

// Binding represents variable bindings
type Binding map[string]Expression

// Special binding values
var (
	NoBindings = Binding{}
	Fail       = Binding{"__FAIL__": Symbol{Name: "__FAIL__"}}
)

// IsFail checks if bindings represent failure
func IsFail(bindings Binding) bool {
	_, exists := bindings["__FAIL__"]
	return exists
}

// GetBinding finds a variable binding
func GetBinding(variable string, bindings Binding) (Expression, bool) {
	value, exists := bindings[variable]
	return value, exists
}

// ExtendBindings adds a new variable binding
func ExtendBindings(variable string, value Expression, bindings Binding) Binding {
	if IsFail(bindings) {
		return Fail
	}
	
	// Create new binding map
	newBindings := make(Binding)
	for k, v := range bindings {
		newBindings[k] = v
	}
	newBindings[variable] = value
	return newBindings
}

// Lookup gets the value of a variable from bindings
func Lookup(variable string, bindings Binding) Expression {
	if value, exists := bindings[variable]; exists {
		return value
	}
	return nil
}

// MatchVariable matches a variable pattern against input
func MatchVariable(pattern Expression, input Expression, bindings Binding) Binding {
	if IsFail(bindings) {
		return Fail
	}
	
	sym, ok := pattern.(Symbol)
	if !ok || !IsVariable(pattern) {
		return Fail
	}
	
	variable := sym.Name
	if binding, exists := GetBinding(variable, bindings); exists {
		// Variable already bound, check if it matches
		if binding.Equal(input) {
			return bindings
		} else {
			return Fail
		}
	} else {
		// Bind variable to input
		return ExtendBindings(variable, input, bindings)
	}
}

// String representation of bindings
func (b Binding) String() string {
	if IsFail(b) {
		return "FAIL"
	}
	if len(b) == 0 {
		return "{}"
	}
	
	result := "{"
	first := true
	for k, v := range b {
		if k == "__FAIL__" {
			continue
		}
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s: %s", k, v.String())
		first = false
	}
	result += "}"
	return result
}

