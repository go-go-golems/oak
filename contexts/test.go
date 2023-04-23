package main

import "fmt"

type Foo struct {
	a int `json:"a" yaml:"a"`
	b int
}

func foo(a int) int {
	return a + 1
}

func (f *Foo) Bar() {
	if true {
		return
	}
	a := 1
	f.a = a
	a = foo(a)
	for i := 0; i < 10; i++ {
		a++
	}
	if a > 10 {
		return
	}
	v := map[string]int{}
	_ = v
	fmt.Printf("bar %d %d", f.a, f.b)
}

func main() {
	fmt.Println("foobar")
}
