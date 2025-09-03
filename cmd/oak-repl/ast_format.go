package main

import (
	"fmt"
	"strings"

	pm "github.com/go-go-golems/oak/pkg/patternmatcher"
)

// prettyPrintLisp renders pm.Expression using width-aware formatting and per-language rules.
func prettyPrintLisp(expr pm.Expression, lang string) string {
	var b strings.Builder
	writeExprFit(&b, expr, 0, astMaxWidth, lang)
	return b.String()
}

func writeExprFit(b *strings.Builder, expr pm.Expression, indent, width int, lang string) {
	inline := inlineString(expr)
	avail := width - indent
	if avail <= 0 {
		avail = 1
	}
	if len(inline) <= avail && compactOnly(lang) {
		b.WriteString(inline)
		return
	}
	if cons, ok := expr.(pm.Cons); ok {
		writeListFit(b, cons, indent, width, lang)
		return
	}
	// atom
	b.WriteString(inline)
}

func writeListFit(b *strings.Builder, cons pm.Cons, indent, width int, lang string) {
	elems, dot := collectList(cons)
	if len(elems) == 0 {
		b.WriteString("()")
		return
	}
	innerIndent := indent + 2
	b.WriteString("(")
	firstInline := inlineString(elems[0])
	b.WriteString(firstInline)
	lineLen := 1 + len(firstInline)
	forceBreak := headBreaks(lang, elems[0])
	for i := 1; i < len(elems); i++ {
		el := elems[i]
		elInline := inlineString(el)
		isField := isFieldList(el)
		fieldName := fieldHeadName(el)
		fieldInline := isField && allowInlineField(lang, fieldName)
		if !forceBreak && (fieldInline || (!isField)) && lineLen+1+len(elInline) <= width {
			b.WriteString(" ")
			b.WriteString(elInline)
			lineLen += 1 + len(elInline)
			continue
		}
		b.WriteString("\n")
		b.WriteString(strings.Repeat(" ", innerIndent))
		writeExprFit(b, el, innerIndent, width, lang)
		lineLen = width
	}
	if dot != nil {
		dotInline := inlineString(dot)
		if !forceBreak && lineLen+3+len(dotInline) <= width {
			b.WriteString(" . ")
			b.WriteString(dotInline)
		} else {
			b.WriteString("\n")
			b.WriteString(strings.Repeat(" ", innerIndent))
			b.WriteString(". ")
			writeExprFit(b, dot, innerIndent+2, width, lang)
		}
	}
	b.WriteString(")")
}

func inlineString(expr pm.Expression) string {
	if cons, ok := expr.(pm.Cons); ok {
		elems, dot := collectList(cons)
		parts := make([]string, 0, len(elems))
		for _, el := range elems {
			parts = append(parts, inlineString(el))
		}
		s := "(" + strings.Join(parts, " ")
		if dot != nil {
			s += " . " + inlineString(dot)
		}
		s += ")"
		return s
	}
	return fmt.Sprint(expr)
}

func collectList(expr pm.Cons) ([]pm.Expression, pm.Expression) {
	var elems []pm.Expression
	var tail pm.Expression
	cur := pm.Expression(expr)
	for {
		c, ok := cur.(pm.Cons)
		if !ok {
			tail = cur
			break
		}
		elems = append(elems, c.Car)
		cur = c.Cdr
	}
	return elems, tail
}

func headBreaks(lang string, head pm.Expression) bool {
	m := astBreakHeads[lang]
	if m == nil {
		return false
	}
	if s, ok := head.(pm.Symbol); ok {
		_, yes := m[s.Name]
		return yes
	}
	return false
}

func isFieldList(expr pm.Expression) bool {
	c, ok := expr.(pm.Cons)
	if !ok {
		return false
	}
	elems, _ := collectList(c)
	if len(elems) == 0 {
		return false
	}
	if s, ok := elems[0].(pm.Symbol); ok {
		return s.Name != ""
	}
	return false
}

func fieldHeadName(expr pm.Expression) string {
	c, ok := expr.(pm.Cons)
	if !ok {
		return ""
	}
	elems, _ := collectList(c)
	if len(elems) == 0 {
		return ""
	}
	if s, ok := elems[0].(pm.Symbol); ok {
		return s.Name
	}
	return ""
}

func allowInlineField(lang, field string) bool {
	m := astInlineFields[lang]
	if m == nil {
		return false
	}
	_, ok := m[field]
	return ok
}

func compactOnly(lang string) bool {
	_, ok := astCompactLangs[lang]
	return ok
}
