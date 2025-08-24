package patternmatcher

import (
	"testing"
)

func TestParseSymbol(t *testing.T) {
	expr, err := Parse("hello")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	sym, ok := expr.(Symbol)
	if !ok {
		t.Fatalf("Expected Symbol, got %T", expr)
	}
	
	if sym.Name != "hello" {
		t.Fatalf("Expected 'hello', got '%s'", sym.Name)
	}
}

func TestParseNumber(t *testing.T) {
	expr, err := Parse("42")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	atom, ok := expr.(Atom)
	if !ok {
		t.Fatalf("Expected Atom, got %T", expr)
	}
	
	if atom.Value != int64(42) {
		t.Fatalf("Expected 42, got %v", atom.Value)
	}
}

func TestParseList(t *testing.T) {
	expr, err := Parse("(a b c)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	cons, ok := expr.(Cons)
	if !ok {
		t.Fatalf("Expected Cons, got %T", expr)
	}
	
	// Check first element
	if !cons.Car.Equal(Symbol{Name: "a"}) {
		t.Fatalf("Expected first element to be 'a', got %v", cons.Car)
	}
}

func TestParseNestedList(t *testing.T) {
	expr, err := Parse("(a (b c) d)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	expected := "(a (b c) d)"
	if expr.String() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, expr.String())
	}
}

func TestParseVariable(t *testing.T) {
	expr, err := Parse("?x")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	if !IsVariable(expr) {
		t.Fatalf("Expected ?x to be recognized as variable")
	}
}

func TestParsePattern(t *testing.T) {
	expr, err := Parse("(?is ?x numberp)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	if !IsSinglePattern(expr) {
		t.Fatalf("Expected (?is ?x numberp) to be recognized as single pattern")
	}
}

func TestParseSegmentPattern(t *testing.T) {
	expr, err := Parse("(?* ?x)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	if !IsSegmentPattern(expr) {
		t.Fatalf("Expected (?* ?x) to be recognized as segment pattern")
	}
}

