package scanner_test

import (
	"slices"
	"testing"

	"go.followtheprocess.codes/dotenv/internal/syntax"
	"go.followtheprocess.codes/dotenv/internal/syntax/scanner"
	"go.followtheprocess.codes/dotenv/internal/syntax/token"
	"go.followtheprocess.codes/test"
)

const multiLineString = `
"""This is a literal string,

it could have multiple lines.

And this one does!
"""
`

const fullFile = `
# This is a comment and is ignored by the parser completely
NUMBER_OF_THINGS=123 # Comments can also go on lines
USERNAME=mysuperuser

# Command substitution
API_KEY=$(op read op://MyVault/SomeService/api_key)

# Variable interpolation
EMAIL=${USER}@email.com # We added $USER above
CACHE_DIR=${HOME}/.cache # Can also reference existing system env vars
DATABASE_URL="postgres://${USER}@localhost/my_database"

# Single quotes force the string to be treated as literal
# no interpolation or command substitution will happen here
LITERAL='${USER} should show up literally'

# Multiline strings can be declared with """. Leading and trailing
# whitespace will be trimmed allowing for nicer formatting.
MANY_LINES="""
This is a lot of text with multiple lines

You could use this to store the contents of a file or
an X509 cert, an SSH key etc.
"""

# Escape sequences work as you'd expect
ESCAPE_ME="Newline\n and a tab\t etc."

# You can even use the export keyword to retain compatibility with e.g. bash
export SOMETHING=yes
`

func TestBasics(t *testing.T) {
	tests := []struct {
		name string        // Name of the test case
		src  string        // Source text to scan
		want []token.Token // Expected token stream
	}{
		{
			name: "empty",
			src:  "",
			want: []token.Token{
				{Kind: token.EOF, Start: 0, End: 0},
			},
		},
		{
			name: "comment",
			src:  "# This is a comment",
			want: []token.Token{
				{Kind: token.Comment, Start: 0, End: 19},
				{Kind: token.EOF, Start: 19, End: 19},
			},
		},
		{
			name: "eq",
			src:  "=",
			want: []token.Token{
				{Kind: token.Eq, Start: 0, End: 1},
				{Kind: token.EOF, Start: 1, End: 1},
			},
		},
		{
			name: "raw string literal",
			src:  "'This is a literal string ${VAR} $(echo hello)'",
			want: []token.Token{
				{Kind: token.RawString, Start: 1, End: 46},
				{Kind: token.EOF, Start: 47, End: 47},
			},
		},
		{
			name: "string literal",
			src:  `"This is a literal string"`,
			want: []token.Token{
				{Kind: token.String, Start: 1, End: 25},
				{Kind: token.EOF, Start: 26, End: 26},
			},
		},
		{
			name: "multiline string literal",
			src:  `"""This is a literal string, it could have multiple lines. But this one doesn't"""`,
			want: []token.Token{
				{Kind: token.String, Start: 3, End: 79},
				{Kind: token.EOF, Start: 82, End: 82},
			},
		},
		{
			name: "actual multiline string",
			src:  multiLineString,
			want: []token.Token{
				{Kind: token.String, Start: 4, End: 81},
				{Kind: token.EOF, Start: 85, End: 85},
			},
		},
		{
			name: "ident",
			src:  "SOME_VAR",
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.EOF, Start: 8, End: 8},
			},
		},
		{
			name: "bare var",
			src:  "SOME_VAR=SOME_VALUE",
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.Ident, Start: 9, End: 19},
				{Kind: token.EOF, Start: 19, End: 19},
			},
		},
		{
			name: "digits",
			src:  "SOME_VAR=123",
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.Ident, Start: 9, End: 12},
				{Kind: token.EOF, Start: 12, End: 12},
			},
		},
		{
			name: "bare var spaces",
			src:  "SOME_VAR = SOME_VALUE",
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.Eq, Start: 9, End: 10},
				{Kind: token.Ident, Start: 11, End: 21},
				{Kind: token.EOF, Start: 21, End: 21},
			},
		},
		{
			name: "single quoted var",
			src:  "SOME_VAR='SOME_VALUE'",
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.RawString, Start: 10, End: 20},
				{Kind: token.EOF, Start: 21, End: 21},
			},
		},
		{
			name: "double quoted var",
			src:  `SOME_VAR="SOME_VALUE"`,
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.String, Start: 10, End: 20},
				{Kind: token.EOF, Start: 21, End: 21},
			},
		},
		{
			name: "raw var expansion",
			src:  "SOME_VAR=$ANOTHER_VAR",
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.VarInterp, Start: 10, End: 21},
				{Kind: token.EOF, Start: 21, End: 21},
			},
		},
		{
			name: "bracketed var expansion",
			src:  "SOME_VAR=${ANOTHER_VAR}",
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.VarInterp, Start: 11, End: 22},
				{Kind: token.EOF, Start: 23, End: 23},
			},
		},
		{
			name: "command expansion",
			src:  "SOME_VAR=$(op read op://MyVault/SomeService/api_key)",
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 8},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.CmdInterp, Start: 11, End: 51},
				{Kind: token.EOF, Start: 52, End: 52},
			},
		},
		{
			name: "string var interpolation",
			src:  `URL="https://${ACCESS_TOKEN}.api.com/v1"`,
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 3},
				{Kind: token.Eq, Start: 3, End: 4},
				{Kind: token.String, Start: 5, End: 39},
				{Kind: token.EOF, Start: 40, End: 40},
			},
		},
		{
			name: "string cmd interpolation",
			src:  `SUPER_SECRET="Sentence with a $(cat ./apikey.txt) cmd in it"`,
			want: []token.Token{
				{Kind: token.Ident, Start: 0, End: 12},
				{Kind: token.Eq, Start: 12, End: 13},
				{Kind: token.String, Start: 14, End: 59},
				{Kind: token.EOF, Start: 60, End: 60},
			},
		},
		{
			name: "export is ignored",
			src:  "export SOME_VAR=VALUE",
			want: []token.Token{
				{Kind: token.Ident, Start: 7, End: 15},
				{Kind: token.Eq, Start: 15, End: 16},
				{Kind: token.Ident, Start: 16, End: 21},
				{Kind: token.EOF, Start: 21, End: 21},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := []byte(tt.src)

			scanner := scanner.New(tt.name, src, testFailHandler(t))
			got := slices.Collect(scanner.All())

			test.EqualFunc(t, got, tt.want, slices.Equal, test.Context("token stream mismatch"))
		})
	}
}

func FuzzScanner(f *testing.F) {
	f.Add(fullFile)

	// The scanner must not panic or loop indefinitely
	f.Fuzz(func(t *testing.T, src string) {
		// No error handler installed, it would stop the test instantly
		scanner := scanner.New("fuzz", []byte(src), nil)

		for tok := range scanner.All() {
			// Positions must be positive integers
			test.True(t, tok.Start >= 0, test.Context("token start position (%d) was negative", tok.Start))
			test.True(t, tok.End >= 0, test.Context("token end position (%d) was negative", tok.End))

			// The kind must be one of the known kinds
			test.True(
				t,
				(tok.Kind >= token.EOF) && (tok.Kind <= token.CmdInterp),
				test.Context("token %s was not one of the pre-defined kinds", tok),
			)

			// End must be >= Start
			test.True(t, tok.End >= tok.Start, test.Context("token %s had invalid start and end positions", tok))
		}
	})
}

func BenchmarkScanner(b *testing.B) {
	for b.Loop() {
		scanner := scanner.New("bench", []byte(fullFile), testFailHandler(b))

		for {
			tok := scanner.Scan()
			if tok.Is(token.EOF, token.Error) {
				break
			}
		}
	}
}

// testFailHandler returns a [syntax.ErrorHandler] that handles scanning errors by failing
// the enclosing test.
func testFailHandler(tb testing.TB) syntax.ErrorHandler {
	tb.Helper()

	return func(pos syntax.Position, msg string) {
		tb.Fatalf("%s: %s", pos, msg)
	}
}
