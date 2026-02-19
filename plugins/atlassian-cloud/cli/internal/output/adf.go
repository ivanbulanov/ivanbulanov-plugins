package output

import (
	"encoding/json"
	"fmt"
	"strings"
)

type adfDoc struct {
	Type    string    `json:"type"`
	Version int       `json:"version"`
	Content []adfNode `json:"content"`
}

type adfNode struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Marks   []adfMark      `json:"marks,omitempty"`
	Content []adfNode      `json:"content,omitempty"`
}

type adfMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

func ADFToMarkdown(adfJSON string) (string, error) {
	var doc adfDoc
	if err := json.Unmarshal([]byte(adfJSON), &doc); err != nil {
		return "", fmt.Errorf("invalid ADF JSON: %w", err)
	}

	var sb strings.Builder
	for _, node := range doc.Content {
		convertNode(&sb, node, "")
	}
	return sb.String(), nil
}

func convertNode(sb *strings.Builder, node adfNode, prefix string) {
	switch node.Type {
	case "paragraph":
		sb.WriteString(prefix)
		convertInlineContent(sb, node.Content)
		sb.WriteString("\n")

	case "heading":
		level := 1
		if l, ok := node.Attrs["level"]; ok {
			if f, ok := l.(float64); ok {
				level = int(f)
			}
		}
		sb.WriteString(strings.Repeat("#", level))
		sb.WriteString(" ")
		convertInlineContent(sb, node.Content)
		sb.WriteString("\n")

	case "codeBlock":
		lang := ""
		if l, ok := node.Attrs["language"]; ok {
			if s, ok := l.(string); ok {
				lang = s
			}
		}
		sb.WriteString("```")
		sb.WriteString(lang)
		sb.WriteString("\n")
		for _, child := range node.Content {
			sb.WriteString(child.Text)
		}
		sb.WriteString("\n```\n")

	case "blockquote":
		for _, child := range node.Content {
			convertNode(sb, child, "> ")
		}

	case "bulletList":
		for _, item := range node.Content {
			if item.Type == "listItem" {
				for _, child := range item.Content {
					convertNode(sb, child, "- ")
				}
			}
		}

	case "orderedList":
		for i, item := range node.Content {
			if item.Type == "listItem" {
				for _, child := range item.Content {
					convertNode(sb, child, fmt.Sprintf("%d. ", i+1))
				}
			}
		}

	case "table":
		convertTable(sb, node)

	case "mediaSingle", "mediaGroup":
		for _, child := range node.Content {
			if child.Type == "media" {
				alt := ""
				if a, ok := child.Attrs["alt"]; ok {
					if s, ok := a.(string); ok {
						alt = s
					}
				}
				url := ""
				if u, ok := child.Attrs["url"]; ok {
					if s, ok := u.(string); ok {
						url = s
					}
				}
				if url == "" {
					if id, ok := child.Attrs["id"]; ok {
						if s, ok := id.(string); ok {
							url = fmt.Sprintf("attachment:%s", s)
						}
					}
				}
				sb.WriteString(fmt.Sprintf("![%s](%s)\n", alt, url))
			}
		}

	case "rule":
		sb.WriteString("---\n")

	case "panel":
		panelType := "info"
		if pt, ok := node.Attrs["panelType"]; ok {
			if s, ok := pt.(string); ok {
				panelType = s
			}
		}
		sb.WriteString(fmt.Sprintf("> **%s**\n", strings.ToUpper(panelType)))
		for _, child := range node.Content {
			convertNode(sb, child, "> ")
		}

	case "expand":
		title := "Details"
		if t, ok := node.Attrs["title"]; ok {
			if s, ok := t.(string); ok {
				title = s
			}
		}
		sb.WriteString(fmt.Sprintf("<details><summary>%s</summary>\n\n", title))
		for _, child := range node.Content {
			convertNode(sb, child, "")
		}
		sb.WriteString("</details>\n")

	default:
		// Fallback: render children
		for _, child := range node.Content {
			convertNode(sb, child, prefix)
		}
	}
}

func convertInlineContent(sb *strings.Builder, nodes []adfNode) {
	for _, node := range nodes {
		switch node.Type {
		case "text":
			text := node.Text
			text = applyMarks(text, node.Marks)
			sb.WriteString(text)

		case "mention":
			name := "@unknown"
			if t, ok := node.Attrs["text"]; ok {
				if s, ok := t.(string); ok {
					name = s
				}
			}
			sb.WriteString(name)

		case "emoji":
			if shortName, ok := node.Attrs["shortName"]; ok {
				if s, ok := shortName.(string); ok {
					sb.WriteString(s)
				}
			}

		case "inlineCard":
			url := ""
			if u, ok := node.Attrs["url"]; ok {
				if s, ok := u.(string); ok {
					url = s
				}
			}
			sb.WriteString(fmt.Sprintf("[link](%s)", url))

		case "status":
			text := ""
			if t, ok := node.Attrs["text"]; ok {
				if s, ok := t.(string); ok {
					text = s
				}
			}
			sb.WriteString(fmt.Sprintf("[%s]", text))

		case "hardBreak":
			sb.WriteString("\n")

		default:
			// Recurse into unknown inline nodes
			convertInlineContent(sb, node.Content)
		}
	}
}

func applyMarks(text string, marks []adfMark) string {
	for _, mark := range marks {
		switch mark.Type {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
		case "underline":
			text = "<u>" + text + "</u>"
		case "link":
			if href, ok := mark.Attrs["href"]; ok {
				if s, ok := href.(string); ok {
					text = fmt.Sprintf("[%s](%s)", text, s)
				}
			}
		}
	}
	return text
}

func convertTable(sb *strings.Builder, node adfNode) {
	if len(node.Content) == 0 {
		return
	}

	for rowIdx, row := range node.Content {
		if row.Type != "tableRow" {
			continue
		}

		sb.WriteString("|")
		for _, cell := range row.Content {
			sb.WriteString(" ")
			var cellBuf strings.Builder
			for _, child := range cell.Content {
				convertInlineContent(&cellBuf, child.Content)
			}
			sb.WriteString(strings.TrimSpace(cellBuf.String()))
			sb.WriteString(" |")
		}
		sb.WriteString("\n")

		// Separator after first row (header)
		if rowIdx == 0 {
			sb.WriteString("|")
			for range row.Content {
				sb.WriteString("---|")
			}
			sb.WriteString("\n")
		}
	}
}
