# Atlassian Document Format (ADF) Reference

ADF is a JSON-based rich text format used by JIRA and Confluence for descriptions, comments, and page content.

## ADF Structure

ADF documents are hierarchical with nodes containing content and marks:

```json
{
  "type": "doc",
  "version": 1,
  "content": [
    {
      "type": "paragraph",
      "content": [
        {
          "type": "text",
          "text": "Hello world"
        }
      ]
    }
  ]
}
```

## Node Types

### Block Nodes

| Type | Description | Markdown Equivalent |
|------|-------------|---------------------|
| `paragraph` | Text paragraph | Plain text with newlines |
| `heading` | Heading (attrs.level: 1-6) | `# ` to `###### ` |
| `bulletList` | Unordered list | `- item` |
| `orderedList` | Numbered list | `1. item` |
| `listItem` | List item | Inside list |
| `codeBlock` | Code block (attrs.language) | ` ```language ` |
| `blockquote` | Quote block | `> text` |
| `table` | Table container | Markdown table |
| `tableRow` | Table row | `\| cell \| cell \|` |
| `tableCell` | Table cell | Cell content |
| `tableHeader` | Header cell | `\| **header** \|` |
| `rule` | Horizontal rule | `---` |
| `panel` | Info/warning panel | Blockquote with type |
| `mediaSingle` | Single media item | `![alt](url)` |
| `mediaGroup` | Multiple media | Multiple images |

### Inline Nodes

| Type | Description | Markdown Equivalent |
|------|-------------|---------------------|
| `text` | Plain text | Text content |
| `hardBreak` | Line break | `\n` or `<br>` |
| `emoji` | Emoji | `:emoji_name:` |
| `mention` | User mention | `@username` |
| `inlineCard` | Link card | `[title](url)` |
| `status` | Status label | `[STATUS]` |

### Text Marks

Marks apply styling to text nodes:

| Mark | Description | Markdown |
|------|-------------|----------|
| `strong` | Bold | `**text**` |
| `em` | Italic | `*text*` |
| `code` | Inline code | `` `code` `` |
| `strike` | Strikethrough | `~~text~~` |
| `underline` | Underline | `<u>text</u>` |
| `link` | Hyperlink | `[text](url)` |
| `subsup` | Subscript/superscript | `<sub>`/`<sup>` |
| `textColor` | Text color | No direct equivalent |

## Parsing Algorithm

To convert ADF to markdown, traverse the document tree recursively:

```
function parseNode(node):
  if node.type == "doc":
    return parseChildren(node.content)

  if node.type == "paragraph":
    return parseChildren(node.content) + "\n\n"

  if node.type == "heading":
    prefix = "#" * node.attrs.level
    return prefix + " " + parseChildren(node.content) + "\n\n"

  if node.type == "text":
    text = node.text
    for mark in node.marks:
      text = applyMark(text, mark)
    return text

  if node.type == "bulletList":
    return parseListItems(node.content, "- ")

  if node.type == "orderedList":
    return parseListItems(node.content, "1. ")

  if node.type == "codeBlock":
    lang = node.attrs.language or ""
    code = parseChildren(node.content)
    return "```" + lang + "\n" + code + "\n```\n\n"

  if node.type == "table":
    return parseTable(node)

  # ... handle other node types
```

## Table Parsing

Tables have a specific structure:

```json
{
  "type": "table",
  "content": [
    {
      "type": "tableRow",
      "content": [
        {
          "type": "tableHeader",
          "content": [{ "type": "paragraph", "content": [...] }]
        },
        {
          "type": "tableHeader",
          "content": [{ "type": "paragraph", "content": [...] }]
        }
      ]
    },
    {
      "type": "tableRow",
      "content": [
        {
          "type": "tableCell",
          "content": [{ "type": "paragraph", "content": [...] }]
        }
      ]
    }
  ]
}
```

Markdown output:

```markdown
| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
```

## Code Block Parsing

Code blocks include language information:

```json
{
  "type": "codeBlock",
  "attrs": {
    "language": "python"
  },
  "content": [
    {
      "type": "text",
      "text": "print('hello')"
    }
  ]
}
```

## Mention Parsing

User mentions include account ID:

```json
{
  "type": "mention",
  "attrs": {
    "id": "712020:91e263ea-e545-4f5a-905a-e24f73a63987",
    "text": "@John Doe",
    "accessLevel": ""
  }
}
```

Convert to: `@John Doe`

## Panel Parsing

Panels have types (info, note, warning, error, success):

```json
{
  "type": "panel",
  "attrs": {
    "panelType": "info"
  },
  "content": [...]
}
```

Convert to blockquote with type indicator:

```markdown
> **Info:** Panel content here
```

## Media Parsing

Media nodes reference attachments:

```json
{
  "type": "mediaSingle",
  "attrs": {
    "layout": "center"
  },
  "content": [
    {
      "type": "media",
      "attrs": {
        "id": "abc123",
        "type": "file",
        "collection": "contentId-123"
      }
    }
  ]
}
```

Reference attachment by ID or note as embedded media.

## Edge Cases

### Nested Lists

Lists can be nested with listItem containing another list:

```json
{
  "type": "bulletList",
  "content": [
    {
      "type": "listItem",
      "content": [
        { "type": "paragraph", "content": [...] },
        {
          "type": "bulletList",
          "content": [...]
        }
      ]
    }
  ]
}
```

### Empty Nodes

Handle empty content arrays gracefully:

```json
{
  "type": "paragraph",
  "content": []
}
```

### Unknown Node Types

Log and skip unknown node types rather than failing.

## Full Example

Input ADF:

```json
{
  "type": "doc",
  "version": 1,
  "content": [
    {
      "type": "heading",
      "attrs": { "level": 2 },
      "content": [
        { "type": "text", "text": "Requirements" }
      ]
    },
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "The API must support " },
        {
          "type": "text",
          "text": "pagination",
          "marks": [{ "type": "strong" }]
        }
      ]
    },
    {
      "type": "codeBlock",
      "attrs": { "language": "json" },
      "content": [
        { "type": "text", "text": "{ \"page\": 1 }" }
      ]
    }
  ]
}
```

Output Markdown:

```markdown
## Requirements

The API must support **pagination**

```json
{ "page": 1 }
```
```
