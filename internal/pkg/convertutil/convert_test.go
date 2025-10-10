package convertutil

import (
	"testing"

	"github.com/samber/mo"
)

type stringerType struct{ s string }

func (t stringerType) String() string { return t.s }

func TestOptionalString(t *testing.T) {
	// none -> none
	if got := OptionalString[stringerType](mo.None[stringerType]()); got.IsPresent() {
		t.Fatalf("expected none, got %v", got)
	}

	// some -> some string
	if got := OptionalString[stringerType](mo.Some(stringerType{s: "abc"})); !got.IsPresent() || got.MustGet() != "abc" {
		t.Fatalf("expected some('abc'), got %v", got)
	}
}

func TestStringify(t *testing.T) {
	in := []stringerType{{"a"}, {"b"}, {"c"}}
	got := Stringify[stringerType](in)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}
