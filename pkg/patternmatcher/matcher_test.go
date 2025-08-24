package patternmatcher

import (
	"testing"
)

func TestBasicPatternMatching(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		shouldMatch bool
		description string
	}{
		// Basic variable matching
		{"?x", "hello", true, "Variable matches symbol"},
		{"?x", "42", true, "Variable matches number"},
		
		// Exact matching
		{"hello", "hello", true, "Exact symbol match"},
		{"42", "42", true, "Exact number match"},
		{"hello", "world", false, "Different symbols don't match"},
		
		// List matching
		{"(a b c)", "(a b c)", true, "Exact list match"},
		{"(a ?x c)", "(a b c)", true, "List with variable"},
		{"(a ?x c)", "(a b d)", false, "List with wrong element"},
		
		// Nested lists
		{"(a (b ?x) d)", "(a (b c) d)", true, "Nested list with variable"},
		{"(a (?x ?y) d)", "(a (b c) d)", true, "Nested list with multiple variables"},
	}
	
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			pattern, err := Parse(test.pattern)
			if err != nil {
				t.Fatalf("Failed to parse pattern '%s': %v", test.pattern, err)
			}
			
			input, err := Parse(test.input)
			if err != nil {
				t.Fatalf("Failed to parse input '%s': %v", test.input, err)
			}
			
			result := PatMatch(pattern, input, NoBindings)
			matched := !IsFail(result)
			
			if matched != test.shouldMatch {
				t.Errorf("Pattern '%s' vs input '%s': expected match=%v, got match=%v, bindings=%v",
					test.pattern, test.input, test.shouldMatch, matched, result)
			}
		})
	}
}

func TestVariableBinding(t *testing.T) {
	pattern, _ := Parse("(?x ?y ?x)")
	input, _ := Parse("(a b a)")
	
	result := PatMatch(pattern, input, NoBindings)
	if IsFail(result) {
		t.Fatal("Pattern should match")
	}
	
	// Check bindings
	xVal := Lookup("?x", result)
	yVal := Lookup("?y", result)
	
	if xVal == nil || !xVal.Equal(Symbol{Name: "a"}) {
		t.Errorf("Expected ?x to be bound to 'a', got %v", xVal)
	}
	
	if yVal == nil || !yVal.Equal(Symbol{Name: "b"}) {
		t.Errorf("Expected ?y to be bound to 'b', got %v", yVal)
	}
}

func TestVariableConsistency(t *testing.T) {
	// Same variable must bind to same value
	pattern, _ := Parse("(?x ?x)")
	input1, _ := Parse("(a a)")
	input2, _ := Parse("(a b)")
	
	result1 := PatMatch(pattern, input1, NoBindings)
	result2 := PatMatch(pattern, input2, NoBindings)
	
	if IsFail(result1) {
		t.Error("Pattern (?x ?x) should match (a a)")
	}
	
	if !IsFail(result2) {
		t.Error("Pattern (?x ?x) should not match (a b)")
	}
}

func TestIsPattern(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		shouldMatch bool
		description string
	}{
		{"(?is ?x numberp)", "42", true, "Number matches numberp"},
		{"(?is ?x numberp)", "hello", false, "Symbol doesn't match numberp"},
		{"(?is ?x symbolp)", "hello", true, "Symbol matches symbolp"},
		{"(?is ?x symbolp)", "42", false, "Number doesn't match symbolp"},
		{"(?is ?x oddp)", "3", true, "Odd number matches oddp"},
		{"(?is ?x oddp)", "4", false, "Even number doesn't match oddp"},
		{"(?is ?x evenp)", "4", true, "Even number matches evenp"},
		{"(?is ?x evenp)", "3", false, "Odd number doesn't match evenp"},
	}
	
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			pattern, err := Parse(test.pattern)
			if err != nil {
				t.Fatalf("Failed to parse pattern: %v", err)
			}
			
			input, err := Parse(test.input)
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}
			
			result := PatMatch(pattern, input, NoBindings)
			matched := !IsFail(result)
			
			if matched != test.shouldMatch {
				t.Errorf("Expected match=%v, got match=%v", test.shouldMatch, matched)
			}
		})
	}
}

func TestAndPattern(t *testing.T) {
	// (?and (?is ?x numberp) (?is ?x oddp)) should match odd numbers
	pattern, _ := Parse("(?and (?is ?x numberp) (?is ?x oddp))")
	
	input1, _ := Parse("3")  // odd number
	input2, _ := Parse("4")  // even number
	input3, _ := Parse("hello") // not a number
	
	result1 := PatMatch(pattern, input1, NoBindings)
	result2 := PatMatch(pattern, input2, NoBindings)
	result3 := PatMatch(pattern, input3, NoBindings)
	
	if IsFail(result1) {
		t.Error("Should match odd number 3")
	}
	
	if !IsFail(result2) {
		t.Error("Should not match even number 4")
	}
	
	if !IsFail(result3) {
		t.Error("Should not match non-number hello")
	}
}

func TestOrPattern(t *testing.T) {
	// (?or < = >) should match comparison operators
	pattern, _ := Parse("(?or < = >)")
	
	input1, _ := Parse("<")
	input2, _ := Parse("=")
	input3, _ := Parse(">")
	input4, _ := Parse("+")
	
	result1 := PatMatch(pattern, input1, NoBindings)
	result2 := PatMatch(pattern, input2, NoBindings)
	result3 := PatMatch(pattern, input3, NoBindings)
	result4 := PatMatch(pattern, input4, NoBindings)
	
	if IsFail(result1) {
		t.Error("Should match <")
	}
	
	if IsFail(result2) {
		t.Error("Should match =")
	}
	
	if IsFail(result3) {
		t.Error("Should match >")
	}
	
	if !IsFail(result4) {
		t.Error("Should not match +")
	}
}

func TestNotPattern(t *testing.T) {
	// (?not hello) should match anything except hello
	pattern, _ := Parse("(?not hello)")
	
	input1, _ := Parse("world")
	input2, _ := Parse("hello")
	
	result1 := PatMatch(pattern, input1, NoBindings)
	result2 := PatMatch(pattern, input2, NoBindings)
	
	if IsFail(result1) {
		t.Error("Should match 'world' (not hello)")
	}
	
	if !IsFail(result2) {
		t.Error("Should not match 'hello'")
	}
}

func TestComplexPatterns(t *testing.T) {
	// Test pattern from PAIP: (?x (?or < = >) ?y)
	pattern, _ := Parse("(?x (?or < = >) ?y)")
	
	input1, _ := Parse("(3 < 4)")
	input2, _ := Parse("(5 = 5)")
	input3, _ := Parse("(7 > 6)")
	input4, _ := Parse("(3 + 4)")
	
	result1 := PatMatch(pattern, input1, NoBindings)
	result2 := PatMatch(pattern, input2, NoBindings)
	result3 := PatMatch(pattern, input3, NoBindings)
	result4 := PatMatch(pattern, input4, NoBindings)
	
	if IsFail(result1) {
		t.Error("Should match (3 < 4)")
	}
	
	if IsFail(result2) {
		t.Error("Should match (5 = 5)")
	}
	
	if IsFail(result3) {
		t.Error("Should match (7 > 6)")
	}
	
	if !IsFail(result4) {
		t.Error("Should not match (3 + 4)")
	}
	
	// Check bindings for successful match
	if !IsFail(result1) {
		xVal := Lookup("?x", result1)
		yVal := Lookup("?y", result1)
		
		if !xVal.Equal(Atom{Value: int64(3)}) {
			t.Errorf("Expected ?x=3, got %v", xVal)
		}
		
		if !yVal.Equal(Atom{Value: int64(4)}) {
			t.Errorf("Expected ?y=4, got %v", yVal)
		}
	}
}

// Test examples from PAIP chapter
func TestPAIPExamples(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		shouldMatch bool
		description string
	}{
		// Basic examples
		{"?x", "hello", true, "Variable matches anything"},
		{"(?is ?n numberp)", "34", true, "Number predicate"},
		{"(?is ?n numberp)", "x", false, "Non-number fails numberp"},
		
		// Relational expressions
		{"(?x (?or < = >) ?y)", "(3 < 4)", true, "Relational expression"},
		{"(?x (?or < = >) ?y)", "(3 + 4)", false, "Non-relational operator"},
		
		// Complex patterns
		{"(?and (?is ?n numberp) (?is ?n oddp))", "3", true, "Odd number"},
		{"(?and (?is ?n numberp) (?is ?n oddp))", "4", false, "Even number"},
		{"(?and (?is ?n numberp) (?is ?n oddp))", "x", false, "Non-number"},
		
		// Negation
		{"(?x (?not ?x))", "(3 4)", true, "Different values"},
		{"(?x (?not ?x))", "(3 3)", false, "Same values"},
	}
	
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			pattern, err := Parse(test.pattern)
			if err != nil {
				t.Fatalf("Failed to parse pattern '%s': %v", test.pattern, err)
			}
			
			input, err := Parse(test.input)
			if err != nil {
				t.Fatalf("Failed to parse input '%s': %v", test.input, err)
			}
			
			result := PatMatch(pattern, input, NoBindings)
			matched := !IsFail(result)
			
			if matched != test.shouldMatch {
				t.Errorf("Pattern '%s' vs input '%s': expected match=%v, got match=%v",
					test.pattern, test.input, test.shouldMatch, matched)
			}
		})
	}
}

