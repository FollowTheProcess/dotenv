package dotenv_test

import (
	"testing"

	"go.followtheprocess.codes/dotenv"
)

func TestHello(t *testing.T) {
	got := dotenv.Hello()
	want := "Hello dotenv"

	if got != want {
		t.Errorf("got %s, wanted %s", got, want)
	}
}
