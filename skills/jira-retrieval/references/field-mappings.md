# JIRA Field Mappings Reference

Complete reference of available JIRA fields for use with `--fields` parameter.

## Core Fields

| Field | Description | Always Present |
|-------|-------------|----------------|
| `key` | Issue key (e.g., DEV-123) | Yes |
| `id` | Numeric issue ID | Yes |
| `self` | API URL for issue | Yes |

## Standard Fields

### Issue Metadata

| Field | Description | Type |
|-------|-------------|------|
| `summary` | Issue title | String |
| `description` | Full description | ADF/Text |
| `issuetype` | Issue type (Bug, Story, etc.) | Object |
| `project` | Project details | Object |
| `created` | Creation timestamp | DateTime |
| `updated` | Last update timestamp | DateTime |
| `resolutiondate` | Resolution timestamp | DateTime |

### Status & Workflow

| Field | Description | Type |
|-------|-------------|------|
| `status` | Current status | Object |
| `resolution` | Resolution type | Object |
| `statuscategorychangedate` | Last status change | DateTime |

### People

| Field | Description | Type |
|-------|-------------|------|
| `assignee` | Assigned user | Object |
| `reporter` | Issue creator | Object |
| `creator` | Original creator | Object |
| `watches` | Watch count | Object |
| `votes` | Vote count | Object |

### Hierarchy

| Field | Description | Type |
|-------|-------------|------|
| `parent` | Parent issue | Object |
| `subtasks` | Child issues | Array |
| `issuelinks` | Linked issues | Array |

### Planning

| Field | Description | Type |
|-------|-------------|------|
| `priority` | Priority level | Object |
| `labels` | Labels/tags | Array |
| `components` | Components | Array |
| `fixVersions` | Fix versions | Array |
| `versions` | Affected versions | Array |

### Time Tracking

| Field | Description | Type |
|-------|-------------|------|
| `timeoriginalestimate` | Original estimate (seconds) | Number |
| `timeestimate` | Remaining estimate | Number |
| `timespent` | Time logged | Number |
| `worklog` | Work log entries | Object |

### Attachments & Comments

| Field | Description | Type |
|-------|-------------|------|
| `attachment` | Attachments list | Array |
| `comment` | Comments | Object |

### Sprint & Agile

| Field | Description | Type |
|-------|-------------|------|
| `sprint` | Sprint(s) | Array |
| `customfield_xxxxx` | Story points, etc. | Varies |

## Field Specifiers

### Include Patterns

```bash
# Specific fields
--fields "key,summary,status"

# All fields
--fields "*all"

# Navigable fields (visible in UI)
--fields "*navigable"
```

### Exclude Patterns

```bash
# All except description
--fields "*all,-description"

# Navigable except comments
--fields "*navigable,-comment"
```

### Combined Patterns

```bash
# Navigable plus worklog
--fields "*navigable,worklog"

# Specific fields with exclusion
--fields "key,summary,description,-comment"
```

## Common Field Combinations

### Quick Status Check

```bash
--fields "key,summary,status,assignee"
```

Returns: Key, title, current status, who's working on it.

### Full Context

```bash
--fields "key,issuetype,summary,status,assignee,description,comment"
```

Returns: Complete issue context including all comments.

### Planning View

```bash
--fields "key,summary,status,priority,sprint,labels,components"
```

Returns: Planning-relevant fields.

### Time Tracking

```bash
--fields "key,summary,timeoriginalestimate,timeestimate,timespent,worklog"
```

Returns: All time-related fields.

### Minimal

```bash
--fields "key,summary"
```

Returns: Just the essentials.

## Field Object Structures

### Status Object

```json
{
  "self": "https://...",
  "description": "",
  "iconUrl": "https://...",
  "name": "In Progress",
  "id": "3",
  "statusCategory": {
    "self": "https://...",
    "id": 4,
    "key": "indeterminate",
    "colorName": "yellow",
    "name": "In Progress"
  }
}
```

**Extract name:** `status.name` → "In Progress"

### Assignee Object

```json
{
  "self": "https://...",
  "accountId": "712020:...",
  "emailAddress": "user@example.com",
  "displayName": "John Doe",
  "active": true,
  "timeZone": "Europe/London"
}
```

**Extract:** `assignee.displayName` or `assignee.emailAddress`

### Issue Type Object

```json
{
  "self": "https://...",
  "id": "10001",
  "description": "A task that needs to be done.",
  "iconUrl": "https://...",
  "name": "Story",
  "subtask": false,
  "hierarchyLevel": 0
}
```

**Extract:** `issuetype.name` → "Story"

### Comment Object

```json
{
  "comments": [
    {
      "self": "https://...",
      "id": "10001",
      "author": { /* user object */ },
      "body": { /* ADF content */ },
      "created": "2024-01-15T10:30:00.000+0000",
      "updated": "2024-01-15T10:30:00.000+0000"
    }
  ],
  "maxResults": 50,
  "total": 3,
  "startAt": 0
}
```

### Attachment Object

```json
{
  "self": "https://...",
  "id": "10001",
  "filename": "screenshot.png",
  "author": { /* user object */ },
  "created": "2024-01-15T10:30:00.000+0000",
  "size": 245760,
  "mimeType": "image/png",
  "content": "https://..."
}
```

## Custom Fields

Custom fields use IDs like `customfield_10001`. Common examples:

| Common Name | Typical ID Pattern |
|-------------|-------------------|
| Story Points | `customfield_10004` |
| Epic Link | `customfield_10008` |
| Sprint | `customfield_10007` |
| Team | `customfield_10100` |

**Finding custom field IDs:**

```bash
# Get all fields with IDs
acli jira workitem view DEV-123 --fields "*all" --json | jq 'keys'
```

## Tips

1. **Start minimal** - Request only needed fields, expand as required
2. **Use navigable** - `*navigable` gives UI-visible fields without noise
3. **Exclude heavy fields** - `-attachment,-comment` when not needed
4. **Check custom fields** - Each JIRA instance has different custom field IDs
5. **Cache field mappings** - Custom field IDs are stable per instance
