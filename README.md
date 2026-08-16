# Ivan Bulanov's Claude Code Plugins

A collection of Claude Code plugins for productivity and integrations.

## Available Plugins

| Plugin | Description | Category |
|--------|-------------|----------|
| [atlassian-cloud](./plugins/atlassian-cloud) | Jira and Confluence Cloud access via Go CLI with OAuth2, progressive disclosure, ADF-to-markdown conversion, and Markdown-to-Confluence publishing with locally rendered Mermaid diagrams | Productivity |
| [redis-reader](./plugins/redis-reader) | Read-only Redis querying with command allowlisting, output size control, and cluster support | Productivity |

<details>
<summary>Deprecated plugins</summary>

| Plugin | Description | Superseded by |
|--------|-------------|---------------|
| [acli-jira](./plugins/acli-jira) | JIRA issue retrieval using acli CLI | [atlassian-cloud](./plugins/atlassian-cloud) |

</details>

## Installation

### 1. Add the marketplace

```bash
claude plugin marketplace add ivanbulanov/ivanbulanov-plugins
```

### 2. Install a plugin

```bash
claude plugin install atlassian-cloud@ivanbulanov-plugins
```

### 3. Enable the plugin

```bash
claude plugin enable atlassian-cloud@ivanbulanov-plugins
```

### 4. Restart Claude Code

Restart your Claude Code session to load the new plugin.

## Updating

```bash
claude plugin marketplace update ivanbulanov-plugins
claude plugin update atlassian-cloud@ivanbulanov-plugins
```

## Plugin Development

This repository follows the Claude Code marketplace structure:

```
.
├── .claude-plugin/
│   └── marketplace.json    # Marketplace manifest
├── plugins/
│   └── <plugin-name>/      # Individual plugins
│       ├── .claude-plugin/
│       │   └── plugin.json
│       ├── skills/         # Plugin skills
│       └── README.md
├── README.md
└── LICENSE
```

To add a new plugin:

1. Create a directory under `plugins/`
2. Add `.claude-plugin/plugin.json` with plugin metadata
3. Add skills, commands, or hooks as needed
4. Update `marketplace.json` to include the new plugin
5. Validate with `claude plugin validate .`

## License

MIT
