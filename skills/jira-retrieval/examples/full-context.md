# Full Context Retrieval Example

This example demonstrates retrieving complete issue context including comments and attachments.

## Scenario

User pastes: "https://social.atlassian.net/browse/DEV-456 - I need the full context for this ticket"

## Workflow

### Step 1: Extract Issue Key from URL

URL: `https://social.atlassian.net/browse/DEV-456`
Extracted key: `DEV-456`

### Step 2: Fetch All Components

Run commands in sequence:

```bash
# Issue details
acli jira workitem view DEV-456

# Comments
acli jira workitem comment list --key DEV-456

# Attachments
acli jira workitem attachment list --key DEV-456
```

### Step 3: Issue Output

```
Key: DEV-456
Type: Bug
Summary: Payment processing fails for EU customers
Status: Open
Assignee: backend-team@example.com
Description: When EU customers attempt to pay with credit cards, the transaction fails with error code 4001.

Steps to reproduce:
1. Create account with EU billing address
2. Add item to cart
3. Proceed to checkout
4. Enter valid credit card
5. Submit payment

Expected: Payment succeeds
Actual: Error "Transaction failed: 4001"

Affected regions: Germany, France, Netherlands
```

### Step 4: Comments Output

```
Comment by: qa-lead@example.com (2024-01-15 10:30)
Confirmed reproducible in staging. Tested with German and French addresses.

Comment by: developer@example.com (2024-01-15 14:22)
Investigating. Appears to be related to the recent Stripe API update.

Comment by: developer@example.com (2024-01-16 09:15)
Root cause identified: EU payment methods require 3DS authentication but our integration wasn't passing the required return_url parameter.
```

### Step 5: Attachments Output

```
Filename: error-screenshot.png
Size: 245 KB
Created: 2024-01-15 10:28

Filename: stripe-logs.txt
Size: 12 KB
Created: 2024-01-15 14:30
```

### Step 6: Present Complete Context

```markdown
## DEV-456: Payment processing fails for EU customers

**Status**: Open | **Type**: Bug | **Assignee**: backend-team@example.com

### Description

When EU customers attempt to pay with credit cards, the transaction fails with error code 4001.

**Steps to reproduce:**
1. Create account with EU billing address
2. Add item to cart
3. Proceed to checkout
4. Enter valid credit card
5. Submit payment

**Expected:** Payment succeeds
**Actual:** Error "Transaction failed: 4001"

**Affected regions:** Germany, France, Netherlands

---

### Comments (3)

1. **qa-lead@example.com** (2024-01-15 10:30):
   Confirmed reproducible in staging. Tested with German and French addresses.

2. **developer@example.com** (2024-01-15 14:22):
   Investigating. Appears to be related to the recent Stripe API update.

3. **developer@example.com** (2024-01-16 09:15):
   Root cause identified: EU payment methods require 3DS authentication but our integration wasn't passing the required return_url parameter.

---

### Attachments (2)

- **error-screenshot.png** (245 KB) - Screenshot of error
- **stripe-logs.txt** (12 KB) - Stripe API logs

---

### Summary

This is a bug affecting EU payment processing. The root cause has been identified: missing 3DS authentication return_url parameter in Stripe integration. Reproducible in staging with German and French addresses.
```

## Handling Complex Content

If the issue description contained tables (e.g., test matrix), fetch with JSON:

```bash
acli jira workitem view DEV-456 --fields "description" --json | \
  ${CLAUDE_PLUGIN_ROOT}/skills/jira-retrieval/scripts/adf-to-markdown.sh
```

## Parallel Fetching (Optimization)

For efficiency, commands can be run in parallel:

```bash
# Run in parallel subshells
(acli jira workitem view DEV-456) &
(acli jira workitem comment list --key DEV-456) &
(acli jira workitem attachment list --key DEV-456) &
wait
```
