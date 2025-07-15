package token_test

import (
	"fmt"
	"testing"
	"testing/quick"

	"go.followtheprocess.codes/dotenv/internal/syntax/token"
)

func TestString(t *testing.T) {
	// All we really care about is the format, let's let quick handle it!
	f := func(tok token.Token) bool {
		return tok.String() == fmt.Sprintf(
			"<Token::%s start=%d, end=%d>",
			tok.Kind.String(),
			tok.Start,
			tok.End,
		)
	}

	err := quick.Check(f, nil)
	if err != nil {
		t.Fatal(err)
	}
}
