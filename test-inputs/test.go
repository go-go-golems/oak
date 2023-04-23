//nolint

package main

import (
	"fmt"
)

// This is a comment
type MyStruct struct {
	// This is a decorator
	Name string
	Ints []int
}

type MyInterface interface {
	MethodOne()
	MethodTwo()
}

const (
	CONSTANT_ONE = 1
	CONSTANT_TWO = 2
)

func foo(s string) string {
	return s + "foo"
}

func main() {
	// Binary expressions with number literals
	result := 1 + 2
	_ = result

	// Binary expressions with string literals
	str := "Hello, " + "World!"
	str = foo(str)

	// Assignment expressions with member expression
	myStruct := MyStruct{Name: "John"}
	myStruct.Name = "Doe"

	// Type declarations without type parameters
	type MyType int

	// Assignment of function to identifier
	myFunc := someFunction
	myFunc()

	// Method definitions in struct
	myStruct.MethodOne()

	// Sequence of comments
	// Comment 1
	// Comment 2
	// Comment 3

	// Type declarations with decorators
	// Decorator
	type DecoratedType string

	// Function calls with string argument
	printString("Hello, World!")

	// Comment followed by function declaration
	// This is a comment
	someFunction()

	// Comma-separated series of numbers
	numbers := []int{1, 2, 3, 4, 5}
	_ = numbers

	// Call to variable or object property
	myFunc()
	myStruct.MethodOne()

	// Keyword tokens
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			fmt.Println("Even")
		} else {
			fmt.Println("Odd")
		}
	}

	// Any node inside call
	someFunction()

	// First identifier in array
	var myArray []MyType
	_ = myArray

	// Last expression in block
	{
		fmt.Println("Inside block")
	}

	// Consecutive identifiers in dotted name
	myStruct.MethodOne()

	// Identifier in screaming snake case
	fmt.Println(CONSTANT_ONE)

	// Key-value pairs with same name
	myMap := map[string]string{
		"key": "key",
	}
	_ = myMap
}

func someFunction() {
	fmt.Println("Inside someFunction")
}

func (s MyStruct) MethodOne() {
	fmt.Println("Inside MyStruct.MethodOne")
}

func printString(s string) {
	fmt.Println(s)
}
