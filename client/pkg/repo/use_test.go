package repo

import (
	"regexp"
	"strings"
	"testing"
)

func TestFormatSourceScriptEnvExports_pwshLiteralEscaping(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "dollar not expanded", value: "a$b", want: `'a$b'`},
		{name: "single quote doubled", value: "x'y", want: `'x''y'`},
		{name: "command substitution inert", value: "$(rm -rf /)", want: `'$(rm -rf /)'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := formatSourceScriptEnvExports("pwsh", []sourceScriptEnv{{Name: "NAME", Value: tt.value}})
			if !strings.Contains(out, tt.want) {
				t.Fatalf("pwsh literal %q: got %q, want substring %q", tt.value, out, tt.want)
			}
		})
	}
}

func TestFormatSourceScriptEnvExports_pwshExpression(t *testing.T) {
	out := formatSourceScriptEnvExports("pwsh", []sourceScriptEnv{{Name: "NAME", Value: "$(cmd)", Expression: true}})
	if !strings.Contains(out, `"$(cmd)"`) {
		t.Fatalf("pwsh expression: got %q, want double-quoted expression", out)
	}
}

func TestFormatSourceScriptEnvExports_pwshUnset(t *testing.T) {
	out := formatSourceScriptEnvExports("pwsh", []sourceScriptEnv{{Name: "NAME", Unset: true}})
	if !strings.Contains(out, `[System.Environment]::SetEnvironmentVariable('NAME',$null,[System.EnvironmentVariableTarget]::Process);`) {
		t.Fatalf("pwsh unset: got %q, want SetEnvironmentVariable with $null", out)
	}
}

func TestFormatSourceScriptEnvExports_unixLiteralEscaping(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "dollar not expanded", value: "a$b", want: `export NAME='a$b'`},
		{name: "single quote escaped", value: "x'y", want: `export NAME='x'\''y'`},
		{name: "command substitution inert", value: "$(rm -rf /)", want: `export NAME='$(rm -rf /)'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := formatSourceScriptEnvExports("unix", []sourceScriptEnv{{Name: "NAME", Value: tt.value}})
			if !strings.Contains(out, tt.want) {
				t.Fatalf("unix literal %q: got %q, want substring %q", tt.value, out, tt.want)
			}
		})
	}
}

func TestFormatSourceScriptEnvExports_unixUnset(t *testing.T) {
	out := formatSourceScriptEnvExports("unix", []sourceScriptEnv{{Name: "NAME", Unset: true}})
	if !strings.Contains(out, "unset NAME") {
		t.Fatalf("unix unset: got %q, want \"unset NAME\"", out)
	}
}

func TestFormatSourceScriptEnvExports_unixExpression(t *testing.T) {
	out := formatSourceScriptEnvExports("unix", []sourceScriptEnv{{Name: "NAME", Value: "$(cmd)", Expression: true}})
	if !strings.Contains(out, `export NAME="$(cmd)"`) {
		t.Fatalf("unix expression: got %q, want double-quoted expression", out)
	}
}

var slugKeyRegexp = regexp.MustCompile(`^[0-9a-f]{16}$`)

func TestSlugifyConstraint(t *testing.T) {
	inputs := []string{">=1.0 || <2.0", "1.2.*", "!=1.2.3", "1.2 - 1.4"}

	seen := make(map[string]string, len(inputs))
	for _, in := range inputs {
		got := slugifyConstraint(in)

		if !slugKeyRegexp.MatchString(got) {
			t.Fatalf("slugifyConstraint(%q) = %q, want match %s", in, got, slugKeyRegexp)
		}

		if again := slugifyConstraint(in); again != got {
			t.Fatalf("slugifyConstraint(%q) not deterministic: %q != %q", in, got, again)
		}

		if prev, ok := seen[got]; ok {
			t.Fatalf("slug collision: %q and %q both map to %q", prev, in, got)
		}
		seen[got] = in
	}
}
