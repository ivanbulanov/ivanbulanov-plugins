package output

import (
	"strings"
	"testing"
)

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
		{
			name: "ordered list",
			input: `{"type":"doc","version":1,"content":[{"type":"orderedList","content":[
				{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"first"}]}]},
				{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"second"}]}]},
				{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"third"}]}]}
			]}]}`,
			want: "1. first\n2. second\n3. third\n",
		},
		{
			name: "table with header and data rows",
			input: `{"type":"doc","version":1,"content":[{"type":"table","content":[
				{"type":"tableRow","content":[
					{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Name"}]}]},
					{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Age"}]}]}
				]},
				{"type":"tableRow","content":[
					{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"Alice"}]}]},
					{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"30"}]}]}
				]}
			]}]}`,
			want: "| Name | Age |\n|---|---|\n| Alice | 30 |\n",
		},
		{
			name: "table with block content in cells",
			input: `{"type":"doc","version":1,"content":[{"type":"table","content":[
				{"type":"tableRow","content":[
					{"type":"tableHeader","content":[{"type":"heading","attrs":{"level":3},"content":[{"type":"text","text":"Code"}]}]},
					{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Output"}]}]}
				]},
				{"type":"tableRow","content":[
					{"type":"tableCell","content":[{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"fmt.Println()"}]}]},
					{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]}
				]}
			]}]}`,
			want: "| ### Code | Output |\n|---|---|\n| ```go fmt.Println() ``` | hello |\n",
		},
		{
			name: "empty table",
			input: `{"type":"doc","version":1,"content":[{"type":"table","content":[]}]}`,
			want:  "",
		},
		{
			name: "mediaSingle with url",
			input: `{"type":"doc","version":1,"content":[{"type":"mediaSingle","content":[
				{"type":"media","attrs":{"alt":"screenshot","url":"https://example.com/img.png"}}
			]}]}`,
			want: "![screenshot](https://example.com/img.png)\n",
		},
		{
			name: "mediaGroup with attachment id",
			input: `{"type":"doc","version":1,"content":[{"type":"mediaGroup","content":[
				{"type":"media","attrs":{"alt":"file","id":"abc-123"}}
			]}]}`,
			want: "![file](attachment:abc-123)\n",
		},
		{
			name: "mediaSingle with no url and no id",
			input: `{"type":"doc","version":1,"content":[{"type":"mediaSingle","content":[
				{"type":"media","attrs":{"alt":"orphan"}}
			]}]}`,
			want: "![orphan]()\n",
		},
		{
			name:  "rule",
			input: `{"type":"doc","version":1,"content":[{"type":"rule"}]}`,
			want:  "---\n",
		},
		{
			name: "panel",
			input: `{"type":"doc","version":1,"content":[{"type":"panel","attrs":{"panelType":"warning"},"content":[
				{"type":"paragraph","content":[{"type":"text","text":"Be careful!"}]}
			]}]}`,
			want: "> **WARNING**\n> Be careful!\n",
		},
		{
			name: "panel defaults to info",
			input: `{"type":"doc","version":1,"content":[{"type":"panel","content":[
				{"type":"paragraph","content":[{"type":"text","text":"Note"}]}
			]}]}`,
			want: "> **INFO**\n> Note\n",
		},
		{
			name: "expand with title",
			input: `{"type":"doc","version":1,"content":[{"type":"expand","attrs":{"title":"More info"},"content":[
				{"type":"paragraph","content":[{"type":"text","text":"hidden content"}]}
			]}]}`,
			want: "<details><summary>More info</summary>\n\nhidden content\n</details>\n",
		},
		{
			name: "expand defaults to Details",
			input: `{"type":"doc","version":1,"content":[{"type":"expand","content":[
				{"type":"paragraph","content":[{"type":"text","text":"stuff"}]}
			]}]}`,
			want: "<details><summary>Details</summary>\n\nstuff\n</details>\n",
		},
		{
			name: "emoji inline",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"emoji","attrs":{"shortName":":thumbsup:"}}
			]}]}`,
			want: ":thumbsup:\n",
		},
		{
			name: "emoji with empty shortName",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"emoji","attrs":{"shortName":""}}
			]}]}`,
			want: "\n",
		},
		{
			name: "inlineCard",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"inlineCard","attrs":{"url":"https://example.com/page"}}
			]}]}`,
			want: "[link](https://example.com/page)\n",
		},
		{
			name: "status inline",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"status","attrs":{"text":"IN PROGRESS","color":"blue"}}
			]}]}`,
			want: "[IN PROGRESS]\n",
		},
		{
			name: "hardBreak inline",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"text","text":"line one"},
				{"type":"hardBreak"},
				{"type":"text","text":"line two"}
			]}]}`,
			want: "line one\nline two\n",
		},
		{
			name: "em mark",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"text","text":"italic","marks":[{"type":"em"}]}
			]}]}`,
			want: "*italic*\n",
		},
		{
			name: "strike mark",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"text","text":"removed","marks":[{"type":"strike"}]}
			]}]}`,
			want: "~~removed~~\n",
		},
		{
			name: "underline mark",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"text","text":"underlined","marks":[{"type":"underline"}]}
			]}]}`,
			want: "<u>underlined</u>\n",
		},
		{
			name: "multiple marks on one node",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[
				{"type":"text","text":"important","marks":[{"type":"strong"},{"type":"em"}]}
			]}]}`,
			want: "***important***\n",
		},
		{
			name: "multi-node document: heading + paragraph + list",
			input: `{"type":"doc","version":1,"content":[
				{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Title"}]},
				{"type":"paragraph","content":[{"type":"text","text":"Some text."}]},
				{"type":"bulletList","content":[
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"a"}]}]},
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"b"}]}]}
				]}
			]}`,
			want: "# Title\nSome text.\n- a\n- b\n",
		},
		{
			name: "nested bullet lists",
			input: `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[
				{"type":"listItem","content":[
					{"type":"paragraph","content":[{"type":"text","text":"parent"}]},
					{"type":"bulletList","content":[
						{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"child 1"}]}]},
						{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"child 2"}]}]}
					]}
				]}
			]}]}`,
			want: "- parent\n  - child 1\n  - child 2\n",
		},
		{
			name: "nested ordered lists",
			input: `{"type":"doc","version":1,"content":[{"type":"orderedList","content":[
				{"type":"listItem","content":[
					{"type":"paragraph","content":[{"type":"text","text":"parent"}]},
					{"type":"orderedList","content":[
						{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"sub a"}]}]},
						{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"sub b"}]}]}
					]}
				]}
			]}]}`,
			want: "1. parent\n   1. sub a\n   2. sub b\n",
		},
		{
			name: "unknown node type passes through children",
			input: `{"type":"doc","version":1,"content":[{"type":"unknownBlock","content":[
				{"type":"paragraph","content":[{"type":"text","text":"fallback"}]}
			]}]}`,
			want: "fallback\n",
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

func TestADFToMarkdown_InvalidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "completely invalid JSON",
			input: `{not json at all`,
		},
		{
			name:  "empty string",
			input: ``,
		},
		{
			name:  "truncated JSON",
			input: `{"type":"doc","version":1,"content":[{"type":"para`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ADFToMarkdown(tt.input)
			if err == nil {
				t.Fatal("expected error for invalid JSON, got nil")
			}
			if !strings.Contains(err.Error(), "invalid ADF JSON") {
				t.Errorf("expected error to contain 'invalid ADF JSON', got: %v", err)
			}
		})
	}
}

func TestRenderADF(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		heading string
		want    string
	}{
		{
			name:    "valid ADF uses markdown output",
			raw:     `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}`,
			heading: "Description",
			want:    "\n### Description\n\nHello\n",
		},
		{
			name:    "invalid JSON falls back to raw",
			raw:     "just plain text",
			heading: "Body",
			want:    "\n### Body\n\njust plain text\n",
		},
		{
			name:    "empty ADF doc falls back to raw",
			raw:     `{"type":"doc","version":1,"content":[]}`,
			heading: "Notes",
			want:    "\n### Notes\n\n{\"type\":\"doc\",\"version\":1,\"content\":[]}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderADF(tt.raw, tt.heading)
			if got != tt.want {
				t.Errorf("renderADF:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
