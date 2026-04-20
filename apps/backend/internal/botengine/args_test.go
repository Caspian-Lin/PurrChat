package botengine

import (
	"testing"
)

func TestReplaceArgsVars(t *testing.T) {
	tests := []struct {
		name   string
		result string
		input  string
		want   string
	}{
		{
			name:   "{args} without index returns all except first",
			result: "{args}",
			input:  "/echo hello world",
			want:   "hello world",
		},
		{
			name:   "{args} with single arg",
			result: "{args}",
			input:  "/echo hello",
			want:   "hello",
		},
		{
			name:   "{args} with no args returns empty",
			result: "{args}",
			input:  "/echo",
			want:   "",
		},
		{
			name:   "{args:0} returns first word",
			result: "{args:0}",
			input:  "/echo hello world",
			want:   "/echo",
		},
		{
			name:   "{args:1} returns second word",
			result: "{args:1}",
			input:  "/echo hello world",
			want:   "hello",
		},
		{
			name:   "{args:2} returns third word",
			result: "{args:2}",
			input:  "/echo hello world",
			want:   "world",
		},
		{
			name:   "{args:5} out of bounds returns empty",
			result: "{args:5}",
			input:  "/echo hello world",
			want:   "",
		},
		{
			name:   "multiple args mixed",
			result: "{args:0} {args:1} -> {args}",
			input:  "/echo hello world",
			want:   "/echo hello -> hello world",
		},
		{
			name:   "args in sentence",
			result: "你说: {args}, 第一个词是 {args:0}",
			input:  "/echo hello world",
			want:   "你说: hello world, 第一个词是 /echo",
		},
		{
			name:   "no args placeholders returns unchanged",
			result: "hello {input} {username}",
			input:  "anything",
			want:   "hello {input} {username}",
		},
		{
			name:   "empty input",
			result: "{args} {args:0}",
			input:  "",
			want:   " ",
		},
		{
			name:   "non-command input (not triggered by command)",
			result: "你说了: {args}",
			input:  "hello world foo bar",
			want:   "你说了: world foo bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplaceArgsVars(tt.result, tt.input)
			if got != tt.want {
				t.Errorf("ReplaceArgsVars(%q, %q) = %q, want %q", tt.result, tt.input, got, tt.want)
			}
		})
	}
}
