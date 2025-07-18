// Package token provides the lexical tokens expected in a .env file.
package token

import (
	"fmt"
	"slices"
)

// Kind is the kind of token.
type Kind int

//go:generate stringer -type Kind -linecomment
const (
	EOF       Kind = iota // EOF
	Error                 // Error
	Comment               // Comment
	Eq                    // Eq
	RawString             // RawString
	String                // String
	Ident                 // Ident
	VarInterp             // VarInterp
	CmdInterp             // CmdInterp
)

// Token is a lexical token in a .env file.
type Token struct {
	Kind  Kind // The type of token this is
	Start int  // Byte offset from the start of the file to the start of this token
	End   int  // Byte offset from the start of the file to the end of this token
}

// String implement [fmt.Stringer] for a [Token].
func (t Token) String() string {
	return fmt.Sprintf("<Token::%s start=%d, end=%d>", t.Kind, t.Start, t.End)
}

// Is reports whether the token is any of the provided [Kind]s.
func (t Token) Is(kinds ...Kind) bool {
	return slices.Contains(kinds, t.Kind)
}
