package output

import "testing"

func TestADFToMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "paragraph with text",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`,
			want:  "Hello world\n",
		},
		{
			name: "heading",
			input: `{"type":"doc","version":1,"content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Title"}]}]}`,
			want:  "## Title\n",
		},
		{
			name: "bold text",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"bold","marks":[{"type":"strong"}]}]}]}`,
			want:  "**bold**\n",
		},
		{
			name: "code block",
			input: `{"type":"doc","version":1,"content":[{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"fmt.Println()"}]}]}`,
			want:  "```go\nfmt.Println()\n```\n",
		},
		{
			name: "bullet list",
			input: `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item 1"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item 2"}]}]}]}]}`,
			want:  "- item 1\n- item 2\n",
		},
		{
			name: "link",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"click","marks":[{"type":"link","attrs":{"href":"https://example.com"}}]}]}]}`,
			want:  "[click](https://example.com)\n",
		},
		{
			name: "mention",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"text":"@John Doe"}}]}]}`,
			want:  "@John Doe\n",
		},
		{
			name: "blockquote",
			input: `{"type":"doc","version":1,"content":[{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"quoted"}]}]}]}`,
			want:  "> quoted\n",
		},
		{
			name: "inline code",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"code","marks":[{"type":"code"}]}]}]}`,
			want:  "`code`\n",
		},
		{
			name:  "empty doc",
			input: `{"type":"doc","version":1,"content":[]}`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ADFToMarkdown(tt.input)
			if err != nil {
				t.Fatalf("ADFToMarkdown error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ADFToMarkdown:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
