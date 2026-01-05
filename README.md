# acli-jira Plugin for Claude Code

Retrieve JIRA issues, comments, and attachments using the Atlassian CLI (acli) with token-efficient output and intelligent ADF parsing.

## Features

- **Token-efficient retrieval**: Uses text output by default (5x smaller than JSON)
- **Adaptive parsing**: Automatically switches to JSON+ADF parsing when complex content detected
- **URL parsing**: Extracts issue keys from Atlassian URLs
- **Full context**: Retrieves issues, comments, and attachments in one flow
- **ADF to Markdown**: Converts Atlassian Document Format to readable markdown

## Prerequisites

- [Atlassian CLI (acli)](https://developer.atlassian.com/cloud/acli/) installed and authenticated
- Claude Code

### Installing acli

```bash
# macOS
brew install atlassian/tap/acli

# Linux/Windows
# Download from https://developer.atlassian.com/cloud/acli/guides/install-acli/
```

### Authenticating acli

```bash
acli jira auth
```

## Installation

### From GitHub

```bash
# Clone the repository
git clone https://github.com/ivanbulanov/acli-jira-plugin.git

# Use with Claude Code
claude --plugin-dir /path/to/acli-jira-plugin
```

### As a local plugin

Copy to your Claude Code plugins directory:

```bash
cp -r acli-jira-plugin ~/.claude/plugins/
```

## Usage

The skill triggers automatically when you:

- Ask about a JIRA issue: "What does DEV-123 say?"
- Paste a JIRA URL: "https://yoursite.atlassian.net/browse/DEV-123"
- Request issue details: "Get the full context for ticket ABC-456"
- Need comments: "Show me the comments on PROJ-789"

### Example Prompts

```
Fetch JIRA issue DEV-5152
```

```
https://social.atlassian.net/browse/DEV-5152 - summarize this ticket
```

```
Get comments and attachments for ABC-123
```

## Skill Structure

```
skills/
└── jira-retrieval/
    ├── SKILL.md              # Main skill instructions
    ├── references/
    │   ├── adf-format.md     # ADF structure guide
    │   ├── acli-commands.md  # acli quick reference
    │   └── field-mappings.md # JIRA field reference
    ├── scripts/
    │   └── adf-to-markdown.sh # ADF converter
    └── examples/
        ├── basic-retrieval.md
        └── full-context.md
```

## License

MIT
