package patternmatcher

import (
	"fmt"
	"strings"
)

// Expression represents a Lisp-like expression
type Expression interface {
	String() string
	Equal(other Expression) bool
}

// Symbol represents a Lisp symbol
type Symbol struct {
	Name string
}

func (s Symbol) String() string {
	return s.Name
}

func (s Symbol) Equal(other Expression) bool {
	if sym, ok := other.(Symbol); ok {
		return s.Name == sym.Name
	}
	return false
}

// Atom represents a Lisp atom (number, string, etc.)
type Atom struct {
	Value interface{}
}

func (a Atom) String() string {
	return fmt.Sprintf("%v", a.Value)
}

func (a Atom) Equal(other Expression) bool {
	if atom, ok := other.(Atom); ok {
		return a.Value == atom.Value
	}
	return false
}

// Cons represents a Lisp cons cell (for lists)
type Cons struct {
	Car Expression
	Cdr Expression
}

func (c Cons) String() string {
	if c.Cdr == nil {
		return fmt.Sprintf("(%s)", c.Car.String())
	}
	
	// Handle proper lists
	if IsList(c) {
		elements := []string{}
		current := Expression(c)
		for current != nil {
			if cons, ok := current.(Cons); ok {
				elements = append(elements, cons.Car.String())
				current = cons.Cdr
			} else {
				// Improper list
				elements = append(elements, ".", current.String())
				break
			}
		}
		return fmt.Sprintf("(%s)", strings.Join(elements, " "))
	}
	
	// Improper list
	return fmt.Sprintf("(%s . %s)", c.Car.String(), c.Cdr.String())
}

func (c Cons) Equal(other Expression) bool {
	if cons, ok := other.(Cons); ok {
		return c.Car.Equal(cons.Car) && 
			   ((c.Cdr == nil && cons.Cdr == nil) || 
			    (c.Cdr != nil && cons.Cdr != nil && c.Cdr.Equal(cons.Cdr)))
	}
	return false
}

// Helper functions
func IsList(expr Expression) bool {
	if expr == nil {
		return true
	}
	if cons, ok := expr.(Cons); ok {
		return IsList(cons.Cdr)
	}
	return false
}

func IsVariable(expr Expression) bool {
	if sym, ok := expr.(Symbol); ok {
		return strings.HasPrefix(sym.Name, "?")
	}
	return false
}

func IsSegmentPattern(expr Expression) bool {
	if cons, ok := expr.(Cons); ok {
		if sym, ok := cons.Car.(Symbol); ok {
			return strings.HasPrefix(sym.Name, "?*") || 
				   strings.HasPrefix(sym.Name, "?+") || 
				   strings.HasPrefix(sym.Name, "??")
		}
	}
	return false
}

func IsSinglePattern(expr Expression) bool {
	if cons, ok := expr.(Cons); ok {
		if sym, ok := cons.Car.(Symbol); ok {
			return sym.Name == "?is" || sym.Name == "?and" || 
				   sym.Name == "?or" || sym.Name == "?not" || 
				   sym.Name == "?if"
		}
	}
	return false
}

