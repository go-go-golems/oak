package main

// Simple is an exported function
func Simple(a int, b string) string {
	return b
}

// complexFunction is an unexported function with multiple parameters
func complexFunction(items []string, callback func(string) bool) []string {
	var result []string
	for _, item := range items {
		if callback(item) {
			result = append(result, item)
		}
	}
	return result
}

// ExampleStruct is a simple struct with methods
type ExampleStruct struct {
	Name string
	Age  int
}

// PublicMethod is an exported method
func (e *ExampleStruct) PublicMethod(param string) error {
	return nil
}

// privateMethod is an unexported method
func (e ExampleStruct) privateMethod() {
	// Do something
}
