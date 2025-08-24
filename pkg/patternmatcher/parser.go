package patternmatcher

import (
	"fmt"
	"strconv"
	"unicode"
)

// Tokenizer
type Token struct {
	Type  string
	Value string
}

const (
	TokenLParen = "LPAREN"
	TokenRParen = "RPAREN"
	TokenSymbol = "SYMBOL"
	TokenNumber = "NUMBER"
	TokenString = "STRING"
	TokenEOF    = "EOF"
)

func Tokenize(input string) ([]Token, error) {
	var tokens []Token
	i := 0
	
	for i < len(input) {
		// Skip whitespace
		if unicode.IsSpace(rune(input[i])) {
			i++
			continue
		}
		
		switch input[i] {
		case '(':
			tokens = append(tokens, Token{TokenLParen, "("})
			i++
		case ')':
			tokens = append(tokens, Token{TokenRParen, ")"})
			i++
		case '"':
			// String literal
			i++
			start := i
			for i < len(input) && input[i] != '"' {
				i++
			}
			if i >= len(input) {
				return nil, fmt.Errorf("unterminated string")
			}
			tokens = append(tokens, Token{TokenString, input[start:i]})
			i++ // skip closing quote
		default:
			// Symbol or number
			start := i
			for i < len(input) && !unicode.IsSpace(rune(input[i])) && 
				input[i] != '(' && input[i] != ')' {
				i++
			}
			value := input[start:i]
			
			// Check if it's a number
			if _, err := strconv.ParseFloat(value, 64); err == nil {
				tokens = append(tokens, Token{TokenNumber, value})
			} else {
				tokens = append(tokens, Token{TokenSymbol, value})
			}
		}
	}
	
	tokens = append(tokens, Token{TokenEOF, ""})
	return tokens, nil
}

// Parser
type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{TokenEOF, ""}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) ParseExpression() (Expression, error) {
	token := p.current()
	
	switch token.Type {
	case TokenLParen:
		return p.parseList()
	case TokenSymbol:
		p.advance()
		return Symbol{Name: token.Value}, nil
	case TokenNumber:
		p.advance()
		if val, err := strconv.ParseInt(token.Value, 10, 64); err == nil {
			return Atom{Value: val}, nil
		} else if val, err := strconv.ParseFloat(token.Value, 64); err == nil {
			return Atom{Value: val}, nil
		}
		return nil, fmt.Errorf("invalid number: %s", token.Value)
	case TokenString:
		p.advance()
		return Atom{Value: token.Value}, nil
	case TokenEOF:
		return nil, fmt.Errorf("unexpected end of input")
	default:
		return nil, fmt.Errorf("unexpected token: %s", token.Value)
	}
}

func (p *Parser) parseList() (Expression, error) {
	if p.current().Type != TokenLParen {
		return nil, fmt.Errorf("expected '('")
	}
	p.advance() // consume '('
	
	var elements []Expression
	
	for p.current().Type != TokenRParen && p.current().Type != TokenEOF {
		expr, err := p.ParseExpression()
		if err != nil {
			return nil, err
		}
		elements = append(elements, expr)
	}
	
	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')'")
	}
	p.advance() // consume ')'
	
	// Convert slice to nested Cons cells
	return SliceToCons(elements), nil
}

// Helper function to convert slice to Cons cells
func SliceToCons(elements []Expression) Expression {
	if len(elements) == 0 {
		return nil
	}
	
	result := Cons{Car: elements[len(elements)-1], Cdr: nil}
	for i := len(elements) - 2; i >= 0; i-- {
		result = Cons{Car: elements[i], Cdr: result}
	}
	return result
}

// Main parse function
func Parse(input string) (Expression, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}
	
	parser := NewParser(tokens)
	return parser.ParseExpression()
}

// Helper function to parse multiple expressions
func ParseAll(input string) ([]Expression, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}
	
	parser := NewParser(tokens)
	var expressions []Expression
	
	for parser.current().Type != TokenEOF {
		expr, err := parser.ParseExpression()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)
	}
	
	return expressions, nil
}

