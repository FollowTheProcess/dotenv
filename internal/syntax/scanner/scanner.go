// Package scanner provides a scanner for .env files, processing the raw
// input text and emitting tokens from the token package.
package scanner

import (
	"bytes"
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
		return s.token(token.Eq)
	case '\'':
		return s.scanRawString()
	case '"':
		return s.scanString()
	case '$':
		return s.scanExpansion()
	default:
		switch {
		case isIdent(char):
			return s.scanIdent()
		case isValue(char):
			return s.scanValue()
		default:
			return s.errorf("unrecognised character: %q", char)
		}
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

// rest returns the rest of src from the scanners current position to eof.
func (s *Scanner) rest() []byte {
	if s.pos >= len(s.src) {
		return nil
	}

	return s.src[s.pos:]
}

// discard discards the current token start position, effectively discarding
// anything the scanner has scanned over in the meantime.
func (s *Scanner) discard() {
	s.start = s.pos
}

// take consumes a run of characters if and only if they precisely match
// the ones provided.
//
// If it does not match, take returns false and the scanner state is not modified.
//
// If however, the next characters in the input do match those provided, they are
// consumed and take returns true.
func (s *Scanner) take(chars string) bool {
	if !bytes.HasPrefix(s.rest(), []byte(chars)) {
		return false
	}

	for range chars {
		s.next()
	}

	return true
}

// skip ignores any characters for which the predicate returns true, stopping at the
// first one that returns false such that after it returns, [Scanner.next] returns the
// first 'false' char.
//
// The scanner start position is brought up to the current position before returning, effectively
// ignoring everything it's travelled over in the meantime.
func (s *Scanner) skip(predicate func(r rune) bool) {
	s.takeWhile(predicate)
	s.discard()
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

// takeWhile consumes characters so long as the predicate returns true, stopping at the
// first one that returns false such that after it returns, [Scanner.next] returns the first 'false' rune.
func (s *Scanner) takeWhile(predicate func(r rune) bool) {
	for predicate(s.peek()) {
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

	s.discard() // Reset the state, we already have the token
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

// scanRawString scans a single quoted string literal.
//
// These are treated as a raw string with no variable interpolation
// or command substitution allowed.
func (s *Scanner) scanRawString() token.Token {
	// We track start and end separately here so we can chop the quotes
	// off the start and end of the string
	start := s.pos

	s.takeUntil('\'', eof)
	if s.peek() == eof {
		return s.error("unterminated string literal")
	}

	end := s.pos
	tok := token.Token{
		Kind:  token.RawString,
		Start: start,
		End:   end,
	}

	s.next()    // Consume the closing quote
	s.discard() // Reset, we already have the token

	return tok
}

// scanString scans a double quoted string literal.
//
// Unlike a raw string with single quotes, a double quoted literal may contain
// variable and/or command interpolation as well as escape sequences.
func (s *Scanner) scanString() token.Token {
	// The opening '"' has already been consumed
	if s.take(`""`) {
		s.discard()
		return s.scanMultilineString()
	}

	// We track start and end separately here so we can chop the quotes
	// off the start and end of the string
	start := s.pos

	s.takeUntil('"', eof)
	if s.peek() == eof {
		return s.error("unterminated string literal")
	}

	// TODO(@FollowTheProcess): Interpolation, I think escape sequences
	// we can just handle later with Go's string stuff so we don't
	// need to do anything special here

	end := s.pos
	tok := token.Token{
		Kind:  token.String,
		Start: start,
		End:   end,
	}

	s.next()    // Consume the closing '"'
	s.discard() // Reset, we have the token now

	return tok
}

// scanMultilineString scans a '"""' multiline string.
//
// The opening 3 quotes have already been consumed.
func (s *Scanner) scanMultilineString() token.Token {
	// We track start and end separately to chop off the quotes
	start := s.pos

	s.takeUntil('"', eof)
	if s.peek() == eof {
		return s.error("unterminated string literal")
	}

	end := s.pos

	// Unlike normal strings, we actually need 2 more quotes to
	// properly terminate it
	if !s.take(`"""`) {
		return s.error("unterminated multiline string")
	}

	// TODO(@FollowTheProcess): Interpolation

	tok := token.Token{
		Kind:  token.String,
		Start: start,
		End:   end,
	}

	s.discard()
	return tok
}

// scanExpansion scans an expansion begun with a '$' in any of the following forms:
//   - $VAR - Normal env var expansion and replacement
//   - ${VAR} - As above but typically used inside strings for interpolation
//   - $(<cmd>) - Command substitution
//
// The opening '$' has already been consumed by Scan.
func (s *Scanner) scanExpansion() token.Token {
	// We don't care about the '$' other than the fact it got us here
	s.discard()

	switch next := s.next(); next {
	case '{':
		// ${VAR}
		s.discard() // We don't actually want the '{'
		start := s.pos
		s.takeUntil('}', eof)
		if s.peek() == eof {
			return s.error("unterminated variable expansion")
		}
		end := s.pos
		s.next() // Consume the closing '}'
		s.discard()
		return token.Token{
			Kind:  token.VarInterp,
			Start: start,
			End:   end,
		}
	case '(':
		// $(<cmd>)
		s.discard() // We don't want the '(' either
		start := s.pos
		s.takeUntil(')', eof)
		if s.peek() == eof {
			return s.error("unterminated command expansion")
		}
		end := s.pos
		s.next() // Consume the closing ')'
		s.discard()
		return token.Token{
			Kind:  token.CmdInterp,
			Start: start,
			End:   end,
		}
	default:
		if isIdent(next) {
			// $VAR
			s.takeWhile(isIdent)
			return s.token(token.VarInterp)
		}
		return s.errorf("unexpected char %q, '$' must be followed by one of '(' or '{'", next)
	}
}

// scanIdent scans a raw identifier e.g. name of an env var.
func (s *Scanner) scanIdent() token.Token {
	// export is ignored, but the 'e' has already been consumed when Scan
	// called next()
	if s.take("xport") {
		s.discard()
		s.skip(unicode.IsSpace)
	}

	s.takeWhile(isIdent)
	return s.token(token.Ident)
}

// scanValue scans an env var value.
//
// It is emitted as a String.
func (s *Scanner) scanValue() token.Token {
	s.takeWhile(isValue)
	return s.token(token.String)
}

// isAlpha reports whether r is an alpha character.
func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isDigit reports whether r is a valid ASCII digit.
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isIdent reports whether r is a valid identifier character.
func isIdent(r rune) bool {
	return isAlpha(r) || isDigit(r) || r == '_' || r == '-'
}

// isValue reports whether r is valid in an environment variable value.
//
// Basically anything other than a space is okay really.
func isValue(r rune) bool {
	return !unicode.IsSpace(r) && r != eof
}
