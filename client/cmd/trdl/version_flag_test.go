package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	return &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
}

func TestValidateVersionArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "channel: repo+group ok", args: []string{"repo", "1.2"}, wantErr: false},
		{name: "channel: repo+group+channel ok", args: []string{"repo", "1.2", "stable"}, wantErr: false},
		{name: "channel: only repo errors", args: []string{"repo"}, wantErr: true},
		{name: "channel: too many args errors", args: []string{"repo", "1.2", "stable", "extra"}, wantErr: true},
		{name: "version: repo+version ok", args: []string{"repo", "v1.2.3"}, wantErr: false},
		{name: "version: with channel errors (mutual excl)", args: []string{"repo", "v1.2.3", "stable"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVersionArgs(newTestCmd(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateVersionArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestProcessExecArgs_versionFlow(t *testing.T) {
	cmd := newTestCmd()

	data, err := processExecArgs(cmd, []string{"repo", "v1.2.3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.version != "v1.2.3" || data.group != "" || data.optionalChannel != "" {
		t.Fatalf("unexpected data: %+v", data)
	}
	if data.optionalBinaryName != "" {
		t.Fatalf("expected no binary name, got %q", data.optionalBinaryName)
	}

	data, err = processExecArgs(cmd, []string{"repo", "v1.2.3", "mybin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.version != "v1.2.3" || data.optionalBinaryName != "mybin" {
		t.Fatalf("unexpected data: %+v", data)
	}

	if _, err := processExecArgs(cmd, []string{"repo", "v1.2.3", "a", "b"}); err == nil {
		t.Fatalf("expected mutual-exclusivity error for extra positional args")
	}
}

func TestProcessExecArgs_channelFlow(t *testing.T) {
	cmd := newTestCmd()

	data, err := processExecArgs(cmd, []string{"repo", "1.2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.group != "1.2" || data.version != "" {
		t.Fatalf("unexpected data: %+v", data)
	}

	data, err = processExecArgs(cmd, []string{"repo", "1.2", "stable", "mybin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.group != "1.2" || data.optionalChannel != "stable" || data.optionalBinaryName != "mybin" {
		t.Fatalf("unexpected data: %+v", data)
	}
}
