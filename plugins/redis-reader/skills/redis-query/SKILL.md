---
name: redis-query
description: Use when the user asks to "query Redis", "check Redis", "look up a Redis key", "inspect cache", "debug session data", "what's in Redis", "show Redis key", "get TTL", "scan Redis keys", "check Redis connection", or any request involving reading data from a Redis instance. Also triggers when the user mentions Redis key patterns like "session:*" or "user:123".
version: 0.1.0
---

# Read-Only Redis Query

Query Redis instances safely using a strict read-only command allowlist. All commands go through a gating script that blocks any mutation.

## Prerequisites

- `redis-cli` must be installed and on PATH
- The user must have an active connection to the Redis instance (typically via SSH tunnel)
- A `.env` file with Redis connection parameters must exist in the current directory

## Step 1: Identify the Environment File

Ask the user which env file contains the Redis connection parameters. Common names:
- `.env`
- `.env.local`
- `.env.redis`
- `.env.development`

Verify the file exists before proceeding.

## Step 2: Verify Connection

Run the PING command to test connectivity:

```bash
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh <env-file> PING
```

If this fails, **stop and tell the user**:
> Cannot connect to Redis. Please establish your tunnel first, for example:
> `ssh -L 6379:<redis-host>:6379 <bastion-host>`

Do not attempt any further queries until PING succeeds.

## Step 3: Execute Queries

All queries must go through the script. Never use `redis-cli` directly.

```bash
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh <env-file> [--max-lines N] <COMMAND> [args...]
```

### Allowed Commands

| Category | Commands |
|----------|----------|
| General | `PING` `INFO` `DBSIZE` `TYPE` `TTL` `PTTL` `EXISTS` `SCAN` `RANDOMKEY` `OBJECT` `DUMP` |
| Strings | `GET` `MGET` `STRLEN` `GETRANGE` |
| Hashes | `HGET` `HGETALL` `HKEYS` `HVALS` `HLEN` `HMGET` `HEXISTS` `HSCAN` |
| Lists | `LRANGE` `LLEN` `LINDEX` |
| Sets | `SMEMBERS` `SCARD` `SISMEMBER` `SRANDMEMBER` `SSCAN` |
| Sorted Sets | `ZRANGE` `ZRANGEBYSCORE` `ZCARD` `ZSCORE` `ZCOUNT` `ZRANK` `ZSCAN` |
| Streams | `XLEN` `XRANGE` `XREVRANGE` `XINFO` |
| Server | `CONFIG GET` |

Any command not in this list will be rejected by the script.

## Common Workflows

### Find keys matching a pattern

Use `SCAN` with a `MATCH` pattern. Never use `KEYS` (it is intentionally blocked).

```bash
# Scan for session keys
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local SCAN 0 MATCH "session:*" COUNT 100
```

The SCAN cursor-based iteration:
1. Start with cursor `0`
2. The first line of output is the next cursor
3. Subsequent lines are matching keys
4. Repeat with the returned cursor until it returns `0`

### Inspect a key

First check its type, then use the appropriate read command:

```bash
# Check type
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local TYPE mykey

# Based on type:
# string -> GET
# hash   -> HGETALL
# list   -> LRANGE mykey 0 -1
# set    -> SMEMBERS mykey
# zset   -> ZRANGE mykey 0 -1
# stream -> XRANGE mykey - + COUNT 10
```

### Check TTL

```bash
# Seconds
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local TTL mykey

# Milliseconds
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local PTTL mykey
```

### Database overview

```bash
# Number of keys
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local DBSIZE

# Server info (memory, clients, stats)
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local INFO
```

### Inspect hash fields

```bash
# All fields and values
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local HGETALL user:123

# Specific field
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local HGET user:123 email

# Just the field names
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local HKEYS user:123
```

### List inspection with bounds

Always use bounded ranges to avoid pulling entire large lists:

```bash
# First 20 items
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local LRANGE queue:jobs 0 19

# List length first, then decide range
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local LLEN queue:jobs
```

### Stream inspection

```bash
# Stream length
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local XLEN mystream

# Last 10 entries
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local XREVRANGE mystream + - COUNT 10

# Stream info
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local XINFO STREAM mystream
```

## Output Control

Output is truncated to **200 lines** by default to keep context compact.

To adjust the limit:

```bash
${CLAUDE_PLUGIN_ROOT}/skills/redis-query/scripts/redis-read.sh .env.local --max-lines 500 INFO
```

When working with commands that can return large results, prefer bounded queries:
- `SCAN` with `COUNT` hint instead of fetching everything
- `LRANGE` with explicit start/stop instead of `0 -1` on unknown-length lists
- `XRANGE`/`XREVRANGE` with `COUNT`
- `ZRANGE` with specific ranges

## Presenting Results

### For simple values

Present directly:

```
**Key**: `session:abc123`
**Type**: string
**TTL**: 3600 seconds
**Value**: {"userId": "42", "role": "admin"}
```

### For hashes

Format as a table:

```
**Key**: `user:123` (hash, 5 fields)

| Field | Value |
|-------|-------|
| name | John Doe |
| email | john@example.com |
| role | admin |
| created | 2026-01-15 |
| active | true |
```

### For scan results

Summarize patterns found:

```
Found 47 keys matching `session:*`:
- 32 with TTL > 1 hour
- 10 with TTL < 5 minutes
- 5 with no expiry set
```

## Error Handling

### Connection refused

If PING fails, do not attempt further commands. Tell the user to establish their tunnel.

### Command blocked

If the script rejects a command, explain that only read-only commands are allowed and suggest the appropriate read-only alternative.

### Empty results

If a key doesn't exist, `GET` returns `(nil)`. Inform the user the key was not found and suggest using `SCAN` to find similar keys.

## Connection Parameters

The script reads these from the specified env file:

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | `localhost` | Redis host address |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | *(none)* | Authentication password |
| `REDIS_CLUSTER` | `false` | Set to `true` for cluster mode |

## Tips

1. **Always check TYPE before reading** — use the right command for the data structure
2. **Use SCAN, not KEYS** — KEYS is blocked because it can freeze large databases
3. **Bound your queries** — use COUNT, LIMIT, and range arguments to control output size
4. **Check TTL** — helps understand if data is stale or about to expire
5. **Start with INFO** — gives a quick overview of memory usage, connected clients, and keyspace stats
6. **Use --max-lines** — increase the limit only when you actually need to see more output
