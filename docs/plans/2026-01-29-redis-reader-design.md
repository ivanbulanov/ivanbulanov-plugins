# Redis Reader Plugin Design

## Purpose

A Claude Code plugin with a skill that allows read-only querying of Redis instances. The user establishes an SSH tunnel to the Redis instance, and Claude Code connects through it using `redis-cli`.

## Connection

Connection parameters are read from a user-specified `.env` file in the current directory:

- `REDIS_HOST` — defaults to `localhost`
- `REDIS_PORT` — defaults to `6379`
- `REDIS_PASSWORD` — optional
- `REDIS_CLUSTER` — defaults to `false`; when `true`, enables cluster mode (`-c` flag)

Before any query, the script verifies connectivity with `PING`. On failure, it stops and asks the user to establish their tunnel.

## Allowlisted Commands (Read-Only)

### General
`PING`, `INFO`, `DBSIZE`, `TYPE`, `TTL`, `PTTL`, `EXISTS`, `SCAN`, `RANDOMKEY`, `OBJECT`, `DUMP`

### Strings
`GET`, `MGET`, `STRLEN`, `GETRANGE`

### Hashes
`HGET`, `HGETALL`, `HKEYS`, `HVALS`, `HLEN`, `HMGET`, `HEXISTS`, `HSCAN`

### Lists
`LRANGE`, `LLEN`, `LINDEX`

### Sets
`SMEMBERS`, `SCARD`, `SISMEMBER`, `SRANDMEMBER`, `SSCAN`

### Sorted Sets
`ZRANGE`, `ZRANGEBYSCORE`, `ZCARD`, `ZSCORE`, `ZCOUNT`, `ZRANK`, `ZSCAN`

### Streams
`XLEN`, `XRANGE`, `XREVRANGE`, `XINFO`

### Server
`CONFIG GET`

**Explicitly excluded**: `KEYS` (blocks on large DBs), all write/mutate/admin commands.

## Script: `redis-read.sh`

```
redis-read.sh <env-file> [--max-lines N] <COMMAND> [args...]
```

1. Parse env file for connection params
2. Validate command against allowlist (reject with helpful error listing allowed commands)
3. Test connectivity with `PING` (exit with tunnel setup message on failure)
4. Execute via `redis-cli`
5. Truncate output to 200 lines by default (configurable with `--max-lines`)

## Output Control

- Default limit: 200 lines
- Truncation message: `"[Output truncated: showing N of M lines. Use --max-lines N to adjust]"`
- SKILL.md instructs Claude to use bounded queries for large-result commands

## Plugin Structure

```
plugins/redis-reader/
├── .claude-plugin/
│   └── plugin.json
├── README.md
└── skills/
    └── redis-query/
        ├── SKILL.md
        └── scripts/
            └── redis-read.sh
```
