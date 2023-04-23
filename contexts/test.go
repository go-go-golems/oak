package main

import "fmt"

type Foo struct {
	a int
	b int
}

func (f *Foo) Bar() {
	fmt.Printf("bar %d %d", f.a, f.b)
}

func main() {
	fmt.Println("foobar")
}
