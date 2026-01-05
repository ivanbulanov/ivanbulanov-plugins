#!/bin/bash
#
# adf-to-markdown.sh - Convert Atlassian Document Format (ADF) JSON to Markdown
#
# Usage:
#   cat issue.json | ./adf-to-markdown.sh
#   acli jira workitem view KEY-123 --json | ./adf-to-markdown.sh
#   ./adf-to-markdown.sh < issue.json
#
# Extracts the description field from JIRA issue JSON and converts ADF to markdown.
# Can also process raw ADF documents directly.
#

set -euo pipefail

# Check for jq dependency
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed." >&2
    echo "Install with: brew install jq (macOS) or apt install jq (Linux)" >&2
    exit 1
fi

# Read input
input=$(cat)

# Determine if this is a JIRA issue response or raw ADF
if echo "$input" | jq -e '.fields.description' > /dev/null 2>&1; then
    # Extract description ADF from JIRA issue
    adf=$(echo "$input" | jq '.fields.description')
elif echo "$input" | jq -e '.type == "doc"' > /dev/null 2>&1; then
    # Already raw ADF
    adf="$input"
else
    echo "Error: Input is not a JIRA issue or ADF document" >&2
    exit 1
fi

# Handle null description
if [ "$adf" = "null" ]; then
    echo "(No description)"
    exit 0
fi

# Convert ADF to markdown using jq
# This is a simplified converter handling common node types
echo "$adf" | jq -r '
def process_marks($text; $marks):
  if $marks == null or ($marks | length) == 0 then
    $text
  else
    reduce $marks[] as $mark ($text;
      if $mark.type == "strong" then "**" + . + "**"
      elif $mark.type == "em" then "*" + . + "*"
      elif $mark.type == "code" then "`" + . + "`"
      elif $mark.type == "strike" then "~~" + . + "~~"
      elif $mark.type == "link" then "[" + . + "](" + $mark.attrs.href + ")"
      else .
      end
    )
  end;

def process_node:
  if .type == "doc" then
    [.content[]? | process_node] | join("")
  elif .type == "paragraph" then
    ([.content[]? | process_node] | join("")) + "\n\n"
  elif .type == "heading" then
    (("#" * .attrs.level) + " " + ([.content[]? | process_node] | join("")) + "\n\n")
  elif .type == "text" then
    process_marks(.text; .marks)
  elif .type == "hardBreak" then
    "\n"
  elif .type == "bulletList" then
    ([.content[]? | "- " + ([.content[]? | process_node] | join("") | gsub("\n\n$"; ""))] | join("\n")) + "\n\n"
  elif .type == "orderedList" then
    ([.content[]? | . as $item | "1. " + ([$item.content[]? | process_node] | join("") | gsub("\n\n$"; ""))] | join("\n")) + "\n\n"
  elif .type == "listItem" then
    [.content[]? | process_node] | join("")
  elif .type == "codeBlock" then
    "```" + (.attrs.language // "") + "\n" + ([.content[]? | .text] | join("")) + "\n```\n\n"
  elif .type == "blockquote" then
    ([.content[]? | process_node] | join("") | split("\n") | map(if . != "" then "> " + . else . end) | join("\n"))
  elif .type == "rule" then
    "---\n\n"
  elif .type == "table" then
    (
      # Process all rows
      [.content[]? |
        [.content[]? |
          "| " + (if .content then ([.content[]? | process_node] | join("") | gsub("\n\n$"; "") | gsub("\n"; " ")) else "" end) + " "
        ] | join("") + "|"
      ] | . as $rows |
      # Add header separator after first row
      if ($rows | length) > 0 then
        $rows[0] + "\n" +
        ($rows[0] | split("|") | map(if . == "" then "" else "---" end) | join("|")) + "\n" +
        ($rows[1:] | join("\n"))
      else
        ""
      end
    ) + "\n\n"
  elif .type == "tableRow" then
    ""
  elif .type == "tableCell" or .type == "tableHeader" then
    [.content[]? | process_node] | join("")
  elif .type == "panel" then
    "> **" + (.attrs.panelType | ascii_upcase) + ":** " + ([.content[]? | process_node] | join("") | gsub("\n\n$"; "")) + "\n\n"
  elif .type == "mention" then
    "@" + .attrs.text
  elif .type == "emoji" then
    ":" + .attrs.shortName + ":"
  elif .type == "inlineCard" then
    "[" + (.attrs.url | split("/") | last) + "](" + .attrs.url + ")"
  elif .type == "mediaSingle" or .type == "mediaGroup" then
    "[Media: " + ([.content[]? | .attrs.id // "embedded"] | join(", ")) + "]\n\n"
  elif .type == "media" then
    ""
  elif .type == "status" then
    "[" + .attrs.text + "]"
  else
    # Unknown type - try to process content if present
    if .content then
      [.content[]? | process_node] | join("")
    else
      ""
    end
  end;

process_node
' | sed 's/\n\n\n*/\n\n/g' | sed 's/^[[:space:]]*$//'
