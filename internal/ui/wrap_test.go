package ui

import "testing"

func TestWrapText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "short line unchanged",
			input: "hello world",
			width: 20,
			want:  "hello world",
		},
		{
			name:  "wraps at word boundary",
			input: "the quick brown fox jumps over",
			width: 15,
			want:  "the quick brown\nfox jumps over",
		},
		{
			name:  "preserves existing newlines",
			input: "line one\nline two",
			width: 80,
			want:  "line one\nline two",
		},
		{
			name:  "hard breaks long word",
			input: "abcdefghijklmnop",
			width: 10,
			want:  "abcdefghij\nklmnop",
		},
		{
			name:  "zero width returns input",
			input: "hello",
			width: 0,
			want:  "hello",
		},
		{
			name:  "multiple wraps on one line",
			input: "a b c d e f g h i j k l m",
			width: 5,
			want:  "a b c\nd e f\ng h i\nj k l\nm",
		},
		{
			name:  "wrap with mixed short and long",
			input: "short\nthis line is way too long to fit",
			width: 15,
			want:  "short\nthis line is\nway too long to\nfit",
		},
		{
			name:  "empty string",
			input: "",
			width: 10,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("wrapText(%q, %d)\ngot:  %q\nwant: %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}
