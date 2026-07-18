//go:build ai_tests

package main

import "testing"

func TestAI_isVersionArg(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		wantVersion bool
	}{
		{name: "plain major is group", arg: "1", wantVersion: false},
		{name: "plain major.minor is group", arg: "1.2", wantVersion: false},
		{name: "plain major.minor.patch is group", arg: "1.2.3", wantVersion: false},
		{name: "plain prerelease is group", arg: "1.2.3-beta", wantVersion: false},
		{name: "plain build metadata is group", arg: "1.2.3+build", wantVersion: false},
		{name: "non-semver name is group", arg: "vendor", wantVersion: false},

		{name: "v-prefixed major.minor is version", arg: "v1.2", wantVersion: true},
		{name: "v-prefixed major is version", arg: "v3", wantVersion: true},
		{name: "gte constraint is version", arg: ">=1.2.0", wantVersion: true},
		{name: "x-range constraint is version", arg: "1.2.x", wantVersion: true},
		{name: "star-range constraint is version", arg: "1.2.*", wantVersion: true},
		{name: "not-equal constraint is version", arg: "!=1.2.3", wantVersion: true},
		{name: "tilde constraint is version", arg: "~1.2", wantVersion: true},
		{name: "caret constraint is version", arg: "^1", wantVersion: true},
		{name: "or constraint is version", arg: ">=1.0 || <2.0", wantVersion: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVersionArg(tt.arg); got != tt.wantVersion {
				t.Fatalf("isVersionArg(%q) = %v, want %v", tt.arg, got, tt.wantVersion)
			}
		})
	}
}
