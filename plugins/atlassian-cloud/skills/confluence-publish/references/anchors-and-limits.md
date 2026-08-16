# Confluence anchors and publish limits

Measured, not inferred. Do not re-derive these.

## The anchor algorithm

Confluence Cloud renders pages with `@atlaskit/renderer`. `ReactSerializer.getHeadingId`
is:

```js
// per heading, over its direct children only
raw  = children.map(getText).join('')          // text | attrs.text | attrs.shortName | "[type]"
base = raw.trim().replace(/\s/g, '-')          // every whitespace char, individually
// empty base yields no id at all
// uniqueness: base, then base.1, base.2 … first free wins
```

No lowercasing, no character stripping, no truncation. A run of two spaces
becomes `--`. `\s` covers tabs and non-breaking spaces. `h4` behaves exactly
like `h2`. Inline formatting is stripped because it lives in ADF marks, not
in text. Emoji contribute `:shortname:`, mentions and status contribute their
label, and a date contributes the literal `[date]`. Atlassian's own comment
above the function instructs callers to `encodeURIComponent` the id when
building a URL.

**Uniqueness is the only order-dependent part.** With no duplicate heading
text the map does not depend on traversal order at all, which is what makes
it robust. That is why the publish command treats a cross-reference to
duplicated heading text as a failure rather than a warning.

**Headings inside expand macros are numbered in a separate namespace.** The
serialiser keeps a second array, `expandHeadingIds`, with Atlassian's own
comment calling it a known problem (`ED-9668`), and the nested ids are
produced only when heading anchor links are enabled. A heading inside an
expand can therefore collide with one outside it, and which one wins is not
derivable. The tool refuses to publish a document whose converted content
contains a heading inside an expand.

## The REST API reports different anchors, and they are wrong

`body-format=view`, `export_view`, `styled_view` and `anonymous_export_view`
all return heading ids of a different form — page title prefixed, spaces
deleted, duplicates suffixed `.1` — for example `ExampleDesignDoc-5.API` for
a page titled "ExampleDesignDoc" with a heading "5. API". The browser
instead renders `5.-API`. This is
[CONFCLOUD-73142](https://community.developer.atlassian.com/t/differences-in-heading-ids-when-rendering-page-in-confluence-vs-through-rest-api/53391),
confirmed by Atlassian. Any implementation that trusts those ids ships a page
of dead links that look plausible. Never read anchors from those endpoints.

## Confluence converts Markdown for us

`POST /wiki/rest/api/contentbody/convert/{to}` accepts
`representation: "markdown"` and handles GFM tables, fenced code with a
language (becoming a code macro), nested lists, blockquotes, inline code,
task lists, strikethrough, autolinks, reference-style links, setext headings,
thematic breaks, hard breaks and HTML entities correctly. Raw `ac:` tags in
the Markdown are escaped rather than passed through, so macros and images are
spliced into the returned storage afterwards.

Using Atlassian's own converter means the output matches what Confluence
does, needs no maintenance, and adds no third-party Markdown-to-storage
library. The endpoint is undocumented for the `markdown` representation, so
the tool runs a pre-flight check to catch it if that ever changes.

## What the converter does not support

Tested, with the tool's response:

| Construct | What Confluence does | Tool response |
|---|---|---|
| HTML comment `<!-- x -->` | **renders as visible literal text** | strip every comment before conversion |
| Raw HTML block, e.g. `<div>` | renders as visible literal text | fail, naming the line |
| `<br>` / `<br/>` | renders as visible literal text | rewrite to a real `<br/>` in the storage, which becomes a hard break |
| Footnote `[^1]` | renders as literal text, definition becomes a stray paragraph | fail, naming the line |
| Nested blockquote | wrapped in a `legacy-content` extension | fail, naming the line |
| Table inside a list item | wrapped in a `legacy-content` extension | de-indent to top level, warn |
| `![alt](./local.png)` | a media node pointing at a relative URL — broken | upload as an attachment and splice `<ac:image>`, exactly as for diagrams |
| `![alt](https://…)` | external media node | left alone |
| Definition list | renders as literal text | fail, naming the line |

The `<br/>` rewrite operates on the parsed storage, not on the string: an
escaped `&lt;br/&gt;` inside `<code>` or `<ac:plain-text-body>` is content and
must survive untouched.

## Intra-page links

`<ac:link ac:anchor="...">` is dead in Cloud: it converts to an
unsupported-content extension. The only working form is a plain `<a href>`
with a percent-encoded fragment.

Percent-encoding is confirmed correct in all three cases, each landing the
target 96–152px from the top: click on an unencoded fragment, click on an
encoded one (`%E2%80%94`, `%2C`, `%2F`), and a fresh page load with a fully
encoded fragment. Confluence's own table-of-contents macro encodes
identically, which is independent corroboration.

**A click during initial page settling silently does nothing** — the URL
updates, the heading exists, and the scroll container does not move. This is
a Confluence rendering behaviour, not a link defect. Any browser-based
verification must wait for the document to settle before clicking, or it
will report false failures.

Absolute URLs are used rather than bare `#fragment`, both of which work, so
that a paragraph copied into a Jira ticket keeps a working link.
