// Package scanner provides a scanner for .env files, processing the raw
// input text and emitting tokens from the token package.
package scanner

import (
	"fmt"
	"iter"
	"slices"
	"unicode"
	"unicode/utf8"

	"go.followtheprocess.codes/dotenv/internal/syntax"
	"go.followtheprocess.codes/dotenv/internal/syntax/token"
)

// eof signifies we have reached the end of the input.
const eof = rune(-1)

// Scanner is the .env file scanner.
type Scanner struct {
	handler    syntax.ErrorHandler // The error handler
	name       string              // The name of the input file
	src        []byte              // Raw source text
	start      int                 // The start position of the current token
	pos        int                 // Current scanner position in src (bytes, 0 indexed)
	line       int                 // Current line number, 1 indexed
	lineOffset int                 // Offset at which the current line started
}

// New returns a new [Scanner].
func New(name string, src []byte, handler syntax.ErrorHandler) *Scanner {
	s := &Scanner{
		handler: handler,
		name:    name,
		src:     src,
		line:    1,
	}

	return s
}

// Scan scans the input and returns the next token.
func (s *Scanner) Scan() token.Token {
	s.skip(unicode.IsSpace)

	switch char := s.next(); char {
	case eof:
		return s.token(token.EOF)
	case '#':
		return s.scanComment()
	case '=':
		return s.scanEq()
	case '\'':
		return s.scanRawString()
	default:
		return s.errorf("unrecognised character: %q", char)
	}
}

// All returns an iterator that emits tokens. The caller must check for EOF or Error.
func (s *Scanner) All() iter.Seq[token.Token] {
	return func(yield func(token.Token) bool) {
		for {
			tok := s.Scan()
			if tok.Is(token.Error, token.EOF) {
				yield(tok)
				return // Stop iterating
			}
			if !yield(tok) {
				return
			}
		}
	}
}

// next returns the next utf8 rune in the input, or [eof], and advances the scanner
// over that rune such that successive calls to [Scanner.next] iterate through
// src one rune at a time.
func (s *Scanner) next() rune {
	if s.pos >= len(s.src) {
		return eof
	}

	char, width := utf8.DecodeRune(s.src[s.pos:])
	s.pos += width

	if char == '\n' {
		s.line++
		s.lineOffset = s.pos
	}

	return char
}

// peek returns the next utf8 rune in the input, or [eof], but does not
// advance the scanner.
//
// Successive calls to peek simply return the same rune again and again.
func (s *Scanner) peek() rune {
	if s.pos >= len(s.src) {
		return eof
	}

	char, _ := utf8.DecodeRune(s.src[s.pos:])

	return char
}

// skip ignores any characters for which the predicate returns true, stopping at the
// first one that returns false such that after it returns, [Scanner.next] returns the
// first 'false' char.
//
// The scanner start position is brought up to the current position before returning, effectively
// ignoring everything it's travelled over in the meantime.
func (s *Scanner) skip(predicate func(r rune) bool) {
	for predicate(s.peek()) {
		s.next()
	}

	s.start = s.pos
}

// takeUntil consumes characters until it hits any of the specified runes.
//
// It stops before it consumes the first specified rune such that after it returns,
// the next call to [Scanner.next] returns the offending rune.
//
//	s.takeUntil('\n', '\t') // Consume runes until you hit a newline or a tab
func (s *Scanner) takeUntil(runes ...rune) {
	for {
		next := s.peek()
		if slices.Contains(runes, next) {
			return
		}
		// Otherwise, advance the scanner
		s.next()
	}
}

// token returns a given token type using the scanner's internal
// state to populate position information.
//
// The scanner's start position is reset just before returning the token.
func (s *Scanner) token(kind token.Kind) token.Token {
	tok := token.Token{
		Kind:  kind,
		Start: s.start,
		End:   s.pos,
	}

	s.start = s.pos
	return tok
}

// error calculates the position information and calls the installed error handler
// with the information, emitting an error token in the process.
//
// The returned token is a [token.Error].
func (s *Scanner) error(msg string) token.Token {
	// So that even if there is no handler installed, we still know something
	// went wrong
	tok := s.token(token.Error)

	if s.handler == nil {
		// Nothing more to do
		return tok
	}

	// Column is the number of bytes between the last newline and the current position
	// +1 because columns are 1 indexed
	startCol := 1 + s.start - s.lineOffset
	endCol := 1 + s.pos - s.lineOffset

	position := syntax.Position{
		Name:     s.name,
		Offset:   s.pos,
		Line:     s.line,
		StartCol: startCol,
		EndCol:   endCol,
	}

	s.handler(position, msg)

	return tok
}

// errorf calls error with a formatted message.
func (s *Scanner) errorf(format string, a ...any) token.Token {
	return s.error(fmt.Sprintf(format, a...))
}

// scanComment scans a line comment e.g. '# This is a comment'.
//
// Effectively, everything up to the next '\n' (or eof) is considered part
// of the comment.
func (s *Scanner) scanComment() token.Token {
	s.takeUntil('\n', eof)
	return s.token(token.Comment)
}

// scanEq scans a '=' literal.
func (s *Scanner) scanEq() token.Token {
	s.next() // Absorb the '='
	return s.token(token.Eq)
}

// scanRawString scans a single quoted string literal.
//
// These are treated as a raw string with no variable interpolation
// or command substitution allowed.
func (s *Scanner) scanRawString() token.Token {
	s.takeUntil('\'', '\n', eof)
	if s.peek() == eof {
		return s.error("unterminated string literal")
	}

	s.next() // Consume the closing single quote
	return s.token(token.RawString)
}
