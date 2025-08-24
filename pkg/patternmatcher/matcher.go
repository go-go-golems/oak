package patternmatcher

import (
	"strconv"
)

// PatMatch is the main pattern matching function
func PatMatch(pattern Expression, input Expression, bindings Binding) Binding {
	if IsFail(bindings) {
		return Fail
	}
	
	// Variable pattern
	if IsVariable(pattern) {
		return MatchVariable(pattern, input, bindings)
	}
	
	// Exact match
	if pattern.Equal(input) {
		return bindings
	}
	
	// Segment pattern
	if IsSegmentPattern(pattern) {
		return SegmentMatcher(pattern, input, bindings)
	}
	
	// Single pattern
	if IsSinglePattern(pattern) {
		return SingleMatcher(pattern, input, bindings)
	}
	
	// Compound pattern (both are lists)
	if patternCons, ok := pattern.(Cons); ok {
		if inputCons, ok := input.(Cons); ok {
			// Match first elements, then rest
			firstMatch := PatMatch(patternCons.Car, inputCons.Car, bindings)
			if IsFail(firstMatch) {
				return Fail
			}
			
			// Handle nil Cdr properly
			if patternCons.Cdr == nil && inputCons.Cdr == nil {
				return firstMatch
			} else if patternCons.Cdr == nil || inputCons.Cdr == nil {
				return Fail
			} else {
				return PatMatch(patternCons.Cdr, inputCons.Cdr, firstMatch)
			}
		}
	}
	
	return Fail
}

// SegmentMatcher handles segment patterns like (?* ?x)
func SegmentMatcher(pattern Expression, input Expression, bindings Binding) Binding {
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	segmentVar, ok := patternCons.Car.(Symbol)
	if !ok {
		return Fail
	}
	
	// Get the segment match function based on the pattern type
	matchFunc := GetSegmentMatchFunc(segmentVar.Name)
	if matchFunc == nil {
		return Fail
	}
	
	return matchFunc(pattern, input, bindings)
}

// SingleMatcher handles single patterns like (?is ?x numberp)
func SingleMatcher(pattern Expression, input Expression, bindings Binding) Binding {
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	operator, ok := patternCons.Car.(Symbol)
	if !ok {
		return Fail
	}
	
	// Get the single match function based on the pattern type
	matchFunc := GetSingleMatchFunc(operator.Name)
	if matchFunc == nil {
		return Fail
	}
	
	return matchFunc(pattern, input, bindings)
}

// Type definitions for match functions
type SegmentMatchFunc func(pattern Expression, input Expression, bindings Binding) Binding
type SingleMatchFunc func(pattern Expression, input Expression, bindings Binding) Binding

// Dispatch tables - initialized in init()
var segmentMatchTable map[string]SegmentMatchFunc
var singleMatchTable map[string]SingleMatchFunc

func init() {
	segmentMatchTable = map[string]SegmentMatchFunc{
		"?*": SegmentMatchStar,
		"?+": SegmentMatchPlus,
		"??": SegmentMatchQuestion,
	}

	singleMatchTable = map[string]SingleMatchFunc{
		"?is":  MatchIs,
		"?and": MatchAnd,
		"?or":  MatchOr,
		"?not": MatchNot,
		"?if":  MatchIf,
	}
}

// GetSegmentMatchFunc returns the appropriate segment match function
func GetSegmentMatchFunc(operator string) SegmentMatchFunc {
	return segmentMatchTable[operator]
}

// GetSingleMatchFunc returns the appropriate single match function
func GetSingleMatchFunc(operator string) SingleMatchFunc {
	return singleMatchTable[operator]
}

// Segment matching functions
func SegmentMatchStar(pattern Expression, input Expression, bindings Binding) Binding {
	// (?* var) matches zero or more elements
	return SegmentMatch(pattern, input, bindings, 0)
}

func SegmentMatchPlus(pattern Expression, input Expression, bindings Binding) Binding {
	// (?+ var) matches one or more elements
	return SegmentMatch(pattern, input, bindings, 1)
}

func SegmentMatchQuestion(pattern Expression, input Expression, bindings Binding) Binding {
	// (?? var) matches zero or one element
	return SegmentMatchZeroOrOne(pattern, input, bindings)
}

// SegmentMatch implements the core segment matching algorithm
func SegmentMatch(pattern Expression, input Expression, bindings Binding, minLength int) Binding {
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	// Extract variable from (?* var) or (?+ var)
	var variable string
	if varCons, ok := patternCons.Cdr.(Cons); ok {
		if varSym, ok := varCons.Car.(Symbol); ok {
			variable = varSym.Name
		} else {
			return Fail
		}
	} else {
		return Fail
	}
	
	// Convert input to slice for easier manipulation
	inputList := ConsToSlice(input)
	
	// Try different segment lengths
	for segmentLen := minLength; segmentLen <= len(inputList); segmentLen++ {
		// Create segment
		segment := SliceToCons(inputList[:segmentLen])
		
		// Try to match variable with this segment
		newBindings := ExtendBindings(variable, segment, bindings)
		if !IsFail(newBindings) {
			return newBindings
		}
	}
	
	return Fail
}

// SegmentMatchZeroOrOne handles (?? var) patterns
func SegmentMatchZeroOrOne(pattern Expression, input Expression, bindings Binding) Binding {
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	// Extract variable
	var variable string
	if varCons, ok := patternCons.Cdr.(Cons); ok {
		if varSym, ok := varCons.Car.(Symbol); ok {
			variable = varSym.Name
		} else {
			return Fail
		}
	} else {
		return Fail
	}
	
	// Try matching zero elements (empty)
	emptyBindings := ExtendBindings(variable, nil, bindings)
	if !IsFail(emptyBindings) {
		return emptyBindings
	}
	
	// Try matching one element
	if inputCons, ok := input.(Cons); ok {
		oneElementBindings := ExtendBindings(variable, inputCons.Car, bindings)
		if !IsFail(oneElementBindings) {
			return oneElementBindings
		}
	}
	
	return Fail
}

// Single pattern matching functions
func MatchIs(pattern Expression, input Expression, bindings Binding) Binding {
	// (?is var predicate) - test predicate on input
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	// Extract variable and predicate
	args := ConsToSlice(patternCons.Cdr)
	if len(args) != 2 {
		return Fail
	}
	
	variable, ok := args[0].(Symbol)
	if !ok {
		return Fail
	}
	
	predicate, ok := args[1].(Symbol)
	if !ok {
		return Fail
	}
	
	// Test predicate
	if TestPredicate(predicate.Name, input) {
		return ExtendBindings(variable.Name, input, bindings)
	}
	
	return Fail
}

func MatchAnd(pattern Expression, input Expression, bindings Binding) Binding {
	// (?and pattern...) - all patterns must match
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	patterns := ConsToSlice(patternCons.Cdr)
	currentBindings := bindings
	
	for _, pat := range patterns {
		currentBindings = PatMatch(pat, input, currentBindings)
		if IsFail(currentBindings) {
			return Fail
		}
	}
	
	return currentBindings
}

func MatchOr(pattern Expression, input Expression, bindings Binding) Binding {
	// (?or pattern...) - any pattern must match
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	patterns := ConsToSlice(patternCons.Cdr)
	
	for _, pat := range patterns {
		result := PatMatch(pat, input, bindings)
		if !IsFail(result) {
			return result
		}
	}
	
	return Fail
}

func MatchNot(pattern Expression, input Expression, bindings Binding) Binding {
	// (?not pattern...) - patterns must not match
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	patterns := ConsToSlice(patternCons.Cdr)
	
	for _, pat := range patterns {
		result := PatMatch(pat, input, bindings)
		if !IsFail(result) {
			return Fail // Pattern matched, so ?not fails
		}
	}
	
	return bindings // No patterns matched, so ?not succeeds
}

func MatchIf(pattern Expression, input Expression, bindings Binding) Binding {
	// (?if condition) - test condition with current bindings
	patternCons, ok := pattern.(Cons)
	if !ok {
		return Fail
	}
	
	args := ConsToSlice(patternCons.Cdr)
	if len(args) != 1 {
		return Fail
	}
	
	condition := args[0]
	
	// Evaluate condition (simplified - just check if it's true)
	if EvaluateCondition(condition, bindings) {
		return bindings
	}
	
	return Fail
}

// Helper functions
func ConsToSlice(expr Expression) []Expression {
	var result []Expression
	current := expr
	
	for current != nil {
		if cons, ok := current.(Cons); ok {
			result = append(result, cons.Car)
			current = cons.Cdr
		} else {
			break
		}
	}
	
	return result
}

func TestPredicate(predicate string, value Expression) bool {
	switch predicate {
	case "numberp":
		if atom, ok := value.(Atom); ok {
			switch atom.Value.(type) {
			case int64, float64:
				return true
			}
		}
		return false
	case "symbolp":
		_, ok := value.(Symbol)
		return ok
	case "atomp":
		_, ok := value.(Atom)
		return ok
	case "oddp":
		if atom, ok := value.(Atom); ok {
			if num, ok := atom.Value.(int64); ok {
				return num%2 == 1
			}
		}
		return false
	case "evenp":
		if atom, ok := value.(Atom); ok {
			if num, ok := atom.Value.(int64); ok {
				return num%2 == 0
			}
		}
		return false
	default:
		return false
	}
}

func EvaluateCondition(condition Expression, bindings Binding) bool {
	// Simplified condition evaluation
	// In a full implementation, this would be more sophisticated
	if cons, ok := condition.(Cons); ok {
		operator := cons.Car
		if opSym, ok := operator.(Symbol); ok {
			args := ConsToSlice(cons.Cdr)
			
			switch opSym.Name {
			case ">":
				if len(args) == 2 {
					return CompareNumbers(args[0], args[1], bindings, ">")
				}
			case "<":
				if len(args) == 2 {
					return CompareNumbers(args[0], args[1], bindings, "<")
				}
			case "=":
				if len(args) == 2 {
					return CompareNumbers(args[0], args[1], bindings, "=")
				}
			}
		}
	}
	return false
}

func CompareNumbers(left, right Expression, bindings Binding, op string) bool {
	// Resolve variables
	leftVal := ResolveValue(left, bindings)
	rightVal := ResolveValue(right, bindings)
	
	// Extract numeric values
	leftNum, leftOk := GetNumber(leftVal)
	rightNum, rightOk := GetNumber(rightVal)
	
	if !leftOk || !rightOk {
		return false
	}
	
	switch op {
	case ">":
		return leftNum > rightNum
	case "<":
		return leftNum < rightNum
	case "=":
		return leftNum == rightNum
	default:
		return false
	}
}

func ResolveValue(expr Expression, bindings Binding) Expression {
	if IsVariable(expr) {
		if sym, ok := expr.(Symbol); ok {
			if value := Lookup(sym.Name, bindings); value != nil {
				return value
			}
		}
	}
	return expr
}

func GetNumber(expr Expression) (float64, bool) {
	if atom, ok := expr.(Atom); ok {
		switch val := atom.Value.(type) {
		case int64:
			return float64(val), true
		case float64:
			return val, true
		case string:
			if num, err := strconv.ParseFloat(val, 64); err == nil {
				return num, true
			}
		}
	}
	return 0, false
}

