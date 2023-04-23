package pkg

import (
	sitter "github.com/smacker/go-tree-sitter"
	"regexp"
)

// The MIT License (MIT)
//
// Copyright (c) 2019 Maxim Sukharev
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
//  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

func QueryPredicateStepTypeToString(t sitter.QueryPredicateStepType) string {
	switch t {
	case sitter.QueryPredicateStepTypeCapture:
		return "capture"
	case sitter.QueryPredicateStepTypeString:
		return "string"
	case sitter.QueryPredicateStepTypeDone:
		return "done"
	default:
		return "unknown"
	}
}

type StepString struct {
	Type  string
	Value string
}

// FilterPredicates is my own copy of sitter.FilterPredicates, because I was having trouble getting it to work.
func FilterPredicates(q *sitter.Query, m *sitter.QueryMatch, input []byte) *sitter.QueryMatch {
	qm := &sitter.QueryMatch{
		ID:           m.ID,
		PatternIndex: m.PatternIndex,
	}

	steps := q.PredicatesForPattern(uint32(qm.PatternIndex))
	if len(steps) == 0 {
		qm.Captures = m.Captures
		return qm
	}

	// this section is just a helper to view the values easier in the debugger
	stepStrings := []StepString{}
	for _, step := range steps {
		typeString := QueryPredicateStepTypeToString(step.Type)
		var value string
		switch step.Type {
		case sitter.QueryPredicateStepTypeString:
			value = q.StringValueForId(step.ValueId)
		case sitter.QueryPredicateStepTypeCapture:
			value = q.CaptureNameForId(step.ValueId)
		case sitter.QueryPredicateStepTypeDone:
			value = "done"
		}
		stepStrings = append(stepStrings, StepString{
			Type:  typeString,
			Value: value})
	}

	_ = stepStrings

	operator := q.StringValueForId(steps[0].ValueId)

	switch operator {
	case "eq?", "not-eq?":
		isPositive := operator == "eq?"

		expectedCaptureNameLeft := q.CaptureNameForId(steps[1].ValueId)

		if steps[2].Type == sitter.QueryPredicateStepTypeCapture {
			expectedCaptureNameRight := q.CaptureNameForId(steps[2].ValueId)

			var nodeLeft, nodeRight *sitter.Node

			found := false

			for _, c := range m.Captures {
				captureName := q.CaptureNameForId(c.Index)
				qm.Captures = append(qm.Captures, c)

				if captureName == expectedCaptureNameLeft {
					nodeLeft = c.Node
				}
				if captureName == expectedCaptureNameRight {
					nodeRight = c.Node
				}

				if nodeLeft != nil && nodeRight != nil {
					if (nodeLeft.Content(input) == nodeRight.Content(input)) == isPositive {
						found = true
					}
					break
				}
			}

			if !found {
				qm.Captures = nil
			}
		} else {
			expectedValueRight := q.StringValueForId(steps[2].ValueId)

			found := false
			for _, c := range m.Captures {
				captureName := q.CaptureNameForId(c.Index)

				qm.Captures = append(qm.Captures, c)
				if expectedCaptureNameLeft != captureName {
					continue
				}

				if (c.Node.Content(input) == expectedValueRight) == isPositive {
					found = true
				}
			}

			if !found {
				qm.Captures = nil
			}
		}

	case "match?", "not-match?":
		isPositive := operator == "match?"

		expectedCaptureName := q.CaptureNameForId(steps[1].ValueId)
		regex := regexp.MustCompile(q.StringValueForId(steps[2].ValueId))

		for _, c := range m.Captures {
			captureName := q.CaptureNameForId(c.Index)
			if expectedCaptureName != captureName {
				continue
			}

			if regex.Match([]byte(c.Node.Content(input))) == isPositive {
				qm.Captures = append(qm.Captures, c)
			}
		}
	}

	return qm
}
