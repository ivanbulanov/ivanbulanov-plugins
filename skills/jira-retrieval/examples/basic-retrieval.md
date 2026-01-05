# Basic Issue Retrieval Example

This example demonstrates the standard workflow for retrieving a JIRA issue.

## Scenario

User asks: "What does DEV-123 say?"

## Workflow

### Step 1: Extract Issue Key

From the user's query, extract the issue key: `DEV-123`

### Step 2: Fetch Issue (Text Output)

```bash
acli jira workitem view DEV-123
```

**Example output:**

```
Key: DEV-123
Type: Story
Summary: Implement user authentication endpoint
Status: In Progress
Assignee: developer@example.com
Description: Create a REST API endpoint for user authentication.

The endpoint should:
- Accept username and password
- Validate credentials against database
- Return JWT token on success
- Return 401 on failure

Acceptance Criteria:
- POST /api/auth/login works
- Token expires in 24 hours
- Rate limiting applied
```

### Step 3: Assess Complexity

The description contains:
- Bullet points (rendered correctly in text)
- No tables
- No code blocks
- No embedded media

**Decision:** Text output is sufficient. No need for JSON + ADF parsing.

### Step 4: Present to User

```markdown
**DEV-123**: Implement user authentication endpoint
**Status**: In Progress
**Assignee**: developer@example.com

**Description**:
Create a REST API endpoint for user authentication.

The endpoint should:
- Accept username and password
- Validate credentials against database
- Return JWT token on success
- Return 401 on failure

**Acceptance Criteria:**
- POST /api/auth/login works
- Token expires in 24 hours
- Rate limiting applied
```

## When to Switch to JSON

If the text output showed signs of lost structure (e.g., table columns merged, code formatting missing), re-fetch with JSON:

```bash
acli jira workitem view DEV-123 --json | ${CLAUDE_PLUGIN_ROOT}/skills/jira-retrieval/scripts/adf-to-markdown.sh
```
