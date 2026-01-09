# Ivan Bulanov's Claude Code Plugins

A collection of Claude Code plugins for productivity and integrations.

## Available Plugins

| Plugin | Description | Category |
|--------|-------------|----------|
| [acli-jira](./plugins/acli-jira) | Retrieve JIRA issues using acli CLI with token-efficient output and ADF parsing | Productivity |

## Installation

### 1. Add the marketplace

```bash
claude plugin marketplace add ivanbulanov/acli-jira-plugin
```

### 2. Install a plugin

```bash
claude plugin install acli-jira@ivanbulanov-plugins
```

### 3. Enable the plugin

```bash
claude plugin enable acli-jira@ivanbulanov-plugins
```

### 4. Restart Claude Code

Restart your Claude Code session to load the new plugin.

## Updating

```bash
claude plugin marketplace update ivanbulanov-plugins
claude plugin update acli-jira@ivanbulanov-plugins
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
