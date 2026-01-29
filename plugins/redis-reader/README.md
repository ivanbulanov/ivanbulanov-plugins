# Redis Reader

Read-only Redis querying skill for Claude Code. Connects to Redis via `redis-cli` with a strict command allowlist that prevents any data mutation.

## Prerequisites

- `redis-cli` installed and available on PATH
- A `.env` file with Redis connection parameters
- An active connection to the Redis instance (e.g. via SSH tunnel)

## Connection Parameters

Set these in your `.env` file (or any env file you specify):

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | *(none)* | Redis password |
| `REDIS_CLUSTER` | `false` | Enable cluster mode |

## Usage

Ask Claude Code to query Redis. Examples:

- "Check what keys match `session:*` in Redis, use `.env.local`"
- "What's the TTL on key `user:123`?"
- "Show me the hash at `config:app`"
- "How many keys are in the database?"

## Allowed Commands

Only read-only commands are permitted. See the skill documentation for the full list. All write, delete, and administrative commands are blocked.

## Output Control

Output is truncated to 200 lines by default to keep context compact. The `--max-lines` flag can adjust this limit.
