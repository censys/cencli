package jq

import (
	"encoding/json"
	"fmt"
	"testing"
)

func decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("invalid test JSON: %v", err)
	}
	return v
}

func TestEval(t *testing.T) {
	host := decode(t, `{
		"host": {
			"ip": "1.2.3.4",
			"services": [
				{"port": 443, "protocol": "HTTP"},
				{"port": 22,  "protocol": "SSH"}
			],
			"location": {"country": "US"}
		}
	}`)

	cases := []struct {
		expr string
		want []any
	}{
		// identity
		{expr: ".", want: []any{host}},
		// simple field
		{expr: ".host.ip", want: []any{"1.2.3.4"}},
		// nested field
		{expr: ".host.location.country", want: []any{"US"}},
		// array expand + field
		{expr: ".host.services[].port", want: []any{float64(443), float64(22)}},
		{expr: ".host.services[].protocol", want: []any{"HTTP", "SSH"}},
		// missing field returns nil (not an error)
		{expr: ".host.missing", want: []any{nil}},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			got, err := Eval(tc.expr, host)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %d values, want %d: %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if fmt.Sprint(got[i]) != fmt.Sprint(tc.want[i]) {
					t.Errorf("[%d] got %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestEval_TopLevelArray(t *testing.T) {
	arr := decode(t, `[{"ip":"1.1.1.1"},{"ip":"2.2.2.2"}]`)
	got, err := Eval(".[]", arr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
}

func TestParse_InvalidExpression(t *testing.T) {
	if _, err := Parse("host.ip"); err == nil {
		t.Fatal("expected error for expression not starting with '.'")
	}
}

func TestFormatValue(t *testing.T) {
	cases := []struct {
		input any
		want  string
	}{
		{nil, "null"},
		{"hello", "hello"},
		{float64(443), "443"},
		{float64(1.5), "1.5"},
		{true, "true"},
		{false, "false"},
		{map[string]any{"k": "v"}, `{"k":"v"}`},
	}
	for _, tc := range cases {
		got := FormatValue(tc.input)
		if got != tc.want {
			t.Errorf("FormatValue(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
