package scanner_test

import (
	"slices"
	"testing"

	"go.followtheprocess.codes/dotenv/internal/syntax"
	"go.followtheprocess.codes/dotenv/internal/syntax/scanner"
	"go.followtheprocess.codes/dotenv/internal/syntax/token"
	"go.followtheprocess.codes/test"
)

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
				{Kind: token.RawString, Start: 0, End: 47},
				{Kind: token.EOF, Start: 47, End: 47},
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

// testFailHandler returns a [syntax.ErrorHandler] that handles scanning errors by failing
// the enclosing test.
func testFailHandler(tb testing.TB) syntax.ErrorHandler {
	tb.Helper()

	return func(pos syntax.Position, msg string) {
		tb.Fatalf("%s: %s", pos, msg)
	}
}
