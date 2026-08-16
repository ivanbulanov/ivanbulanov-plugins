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

func attrString(attrs map[string]any, key, defaultVal string) string {
	if v, ok := attrs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func attrInt(attrs map[string]any, key string, defaultVal int) int {
	if v, ok := attrs[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return defaultVal
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

func renderADF(raw, heading string) string {
	if md, err := ADFToMarkdown(raw); err == nil && md != "" {
		return fmt.Sprintf("\n### %s\n\n%s", heading, md)
	}
	return fmt.Sprintf("\n### %s\n\n%s\n", heading, raw)
}

func convertNode(sb *strings.Builder, node adfNode, prefix string) {
	switch node.Type {
	case "paragraph":
		sb.WriteString(prefix)
		convertInlineContent(sb, node.Content)
		sb.WriteString("\n")

	case "heading":
		level := attrInt(node.Attrs, "level", 1)
		sb.WriteString(strings.Repeat("#", level))
		sb.WriteString(" ")
		convertInlineContent(sb, node.Content)
		sb.WriteString("\n")

	case "codeBlock":
		lang := attrString(node.Attrs, "language", "")
		fmt.Fprintf(sb, "```%s\n", lang)
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
				convertListItem(sb, item.Content, prefix+"- ", prefix+"  ")
			}
		}

	case "orderedList":
		for i, item := range node.Content {
			if item.Type == "listItem" {
				marker := fmt.Sprintf("%s%d. ", prefix, i+1)
				indent := prefix + strings.Repeat(" ", len(fmt.Sprintf("%d. ", i+1)))
				convertListItem(sb, item.Content, marker, indent)
			}
		}

	case "table":
		convertTable(sb, node)

	case "mediaSingle", "mediaGroup":
		for _, child := range node.Content {
			if child.Type == "media" {
				alt := attrString(child.Attrs, "alt", "")
				url := attrString(child.Attrs, "url", "")
				if url == "" {
					if id := attrString(child.Attrs, "id", ""); id != "" {
						url = "attachment:" + id
					}
				}
				fmt.Fprintf(sb, "![%s](%s)\n", alt, url)
			}
		}

	case "rule":
		sb.WriteString("---\n")

	case "panel":
		panelType := attrString(node.Attrs, "panelType", "info")
		fmt.Fprintf(sb, "> **%s**\n", strings.ToUpper(panelType))
		for _, child := range node.Content {
			convertNode(sb, child, "> ")
		}

	case "expand":
		title := attrString(node.Attrs, "title", "Details")
		fmt.Fprintf(sb, "<details><summary>%s</summary>\n\n", title)
		for _, child := range node.Content {
			convertNode(sb, child, "")
		}
		sb.WriteString("</details>\n")

	default:
		for _, child := range node.Content {
			convertNode(sb, child, prefix)
		}
	}
}

func convertListItem(sb *strings.Builder, children []adfNode, marker, indent string) {
	for i, child := range children {
		if i == 0 {
			convertNode(sb, child, marker)
		} else {
			convertNode(sb, child, indent)
		}
	}
}

func convertInlineContent(sb *strings.Builder, nodes []adfNode) {
	for _, node := range nodes {
		switch node.Type {
		case "text":
			sb.WriteString(applyMarks(node.Text, node.Marks))

		case "mention":
			sb.WriteString(attrString(node.Attrs, "text", "@unknown"))

		case "emoji":
			if s := attrString(node.Attrs, "shortName", ""); s != "" {
				sb.WriteString(s)
			}

		case "inlineCard":
			url := attrString(node.Attrs, "url", "")
			fmt.Fprintf(sb, "[link](%s)", url)

		case "status":
			text := attrString(node.Attrs, "text", "")
			fmt.Fprintf(sb, "[%s]", text)

		case "hardBreak":
			sb.WriteString("\n")

		default:
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
			if href := attrString(mark.Attrs, "href", ""); href != "" {
				text = fmt.Sprintf("[%s](%s)", text, href)
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
				convertNode(&cellBuf, child, "")
			}
			cellText := strings.ReplaceAll(cellBuf.String(), "\n", " ")
			sb.WriteString(strings.TrimSpace(cellText))
			sb.WriteString(" |")
		}
		sb.WriteString("\n")

		if rowIdx == 0 {
			sb.WriteString("|")
			for range row.Content {
				sb.WriteString("---|")
			}
			sb.WriteString("\n")
		}
	}
}
