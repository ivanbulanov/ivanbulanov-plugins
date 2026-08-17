---
name: confluence-publish
description: Use when the user asks to "publish this design to Confluence", "push this markdown to a wiki page", "update the Confluence page from this document", or names a Markdown file and a Confluence page together. Also triggers on "republish", "sync the design doc", and requests to turn section cross-references into working Confluence links.
---

# Publishing a Markdown document to Confluence

Publishes one Markdown file to one existing Confluence Cloud page: cross-references
become working deep links, Mermaid diagrams are rendered and embedded, and a
table of contents is inserted by default. The CLI does all of it; do not
rewrite links by hand.

**The target page must already exist.** This command does not create pages —
use the atlassian-confluence skill to look one up first if you only have a
space and a title.

## Setup

Run once at the start of any operation:

```bash
${CLAUDE_PLUGIN_ROOT}/scripts/setup.sh
```

This builds the CLI if needed, checks authentication, and reports whether
`mmdc` is on `PATH`. If it fails with a Go error, guide the user to install Go
(`mise install go@latest` or https://go.dev/dl/). If it reports `mmdc` missing
and the document contains Mermaid fences, give the user the install command
from **Diagrams** below before going any further — never install it yourself.

**CLI path**: `${CLAUDE_PLUGIN_ROOT}/cli/bin/atlassian-cloud` — shown as
`atlassian-cloud` below.

If not authenticated, follow the **Guided API Token Setup** in the
atlassian-jira skill — the same auth config covers both Jira and Confluence.
API-token users need no further setup: the token carries the user's own
permissions. OAuth users need a token granting `write:confluence-content`
and `write:confluence-file`; if authorisation predates those scopes, the
publish fails with a 403 until `atlassian-cloud auth login` is run again.

**Sandboxing:** If you have enabled Claude Code's Bash sandbox (it is off by
default), the build step and the CLI need network access. See the plugin
README's **Sandboxing** section for the `settings.json` allowlist.

## Always dry-run first

```bash
atlassian-cloud confluence publish design.md --page <id-or-url> --dry-run
```

`--page` accepts a numeric page id (`--page 123456`) or a full page URL
(`--page https://example.atlassian.net/wiki/spaces/DOCS/pages/123456/Design`).
It can be omitted if the document itself carries a
`<!-- confluence-page: <id-or-url> -->` comment.

Report the summary to the user — references linked, tables hoisted, links
unwrapped — and the path of the generated storage dump (written under
`--assets-dir`, which defaults to the source document's own directory).
**Publishing writes to the company wiki: ask before the real run.**

## Publishing

```bash
atlassian-cloud confluence publish design.md --page <id-or-url>
```

Useful flags: `--title` to rename the page, `--no-toc` to skip the
table-of-contents macro (one is inserted at the top by default, or wherever a
`<!-- confluence-toc -->` comment sits — the comment only chooses the
position, never whether one appears), `--link-refs none` to leave
cross-references as plain text instead of linking them, `--force` to publish
over a page this tool did not last write, and `--assets-dir` to change where
rendered diagrams go.

Exit codes: `0` success, `1` error, `2` authentication needed, `3` refused
before writing anything, `4` published but verification failed.

## Diagrams

Mermaid fences are rendered locally with `mmdc` from
`@mermaid-js/mermaid-cli` (`npm install -g @mermaid-js/mermaid-cli`), which
the user must install themselves — the tool never installs or downloads it.
Point at a specific binary with the `MMDC` environment variable. Diagram
source is never sent anywhere but the user's own Confluence site — no hosted
renderer is used.

## When it refuses

| Message | What to do |
|---|---|
| constructs Confluence cannot represent | Show the user the file and line. Do not strip the construct without asking |
| cross-references resolve to no heading | The document names a section that does not exist. Report the tokens |
| a heading whose text is duplicated | Two headings share text. Ask the user to disambiguate one |
| not written by this tool | Someone edited the page in Confluence. Show the diff and ask before `--force` |
| mmdc not found | Give the user the install command above. Never install it yourself |
| cannot write to … / go build failed | Claude Code's Bash sandbox is blocking the build. `setup.sh` prints the exact `settings.json` entry with the real path filled in — relay it verbatim. Never edit the user's settings yourself |

## Never

- Never pass `--force` without the user agreeing.
- Never send diagram source to kroki.io, mermaid.ink or any other host.
- Never read heading anchors from the REST API's `view` or `export_view`
  bodies; they are a different, wrong scheme. See
  `${CLAUDE_SKILL_DIR}/references/anchors-and-limits.md`.

## Diagram files accumulate

Rendered diagrams are content-addressed, so republishing an unchanged
document re-uses them. Editing a diagram produces a new file and a new
attachment, and **the superseded ones are not removed** — neither the SVG in
`--assets-dir` nor the attachment on the page. Nothing breaks; the page keeps
showing the current diagram. If the user cares about the clutter, point it out
rather than deleting anything yourself.
