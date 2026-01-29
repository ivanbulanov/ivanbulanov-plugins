#!/usr/bin/env bash
set -euo pipefail

# Redis Read-Only Query Script
# Enforces a strict allowlist of read-only Redis commands.
# Usage: redis-read.sh <env-file> [--max-lines N] <COMMAND> [args...]

readonly DEFAULT_MAX_LINES=200
readonly DEFAULT_HOST="localhost"
readonly DEFAULT_PORT="6379"

# Read-only command allowlist (uppercase)
readonly ALLOWED_COMMANDS=(
  # General
  PING INFO DBSIZE TYPE TTL PTTL EXISTS SCAN RANDOMKEY OBJECT DUMP
  # Strings
  GET MGET STRLEN GETRANGE
  # Hashes
  HGET HGETALL HKEYS HVALS HLEN HMGET HEXISTS HSCAN
  # Lists
  LRANGE LLEN LINDEX
  # Sets
  SMEMBERS SCARD SISMEMBER SRANDMEMBER SSCAN
  # Sorted sets
  ZRANGE ZRANGEBYSCORE ZCARD ZSCORE ZCOUNT ZRANK ZSCAN
  # Streams
  XLEN XRANGE XREVRANGE XINFO
  # Server (read-only)
  CONFIG
)

print_usage() {
  echo "Usage: redis-read.sh <env-file> [--max-lines N] <COMMAND> [args...]"
  echo ""
  echo "Options:"
  echo "  --max-lines N   Maximum output lines (default: ${DEFAULT_MAX_LINES})"
  echo ""
  echo "Allowed commands:"
  echo "  General:      PING INFO DBSIZE TYPE TTL PTTL EXISTS SCAN RANDOMKEY OBJECT DUMP"
  echo "  Strings:      GET MGET STRLEN GETRANGE"
  echo "  Hashes:       HGET HGETALL HKEYS HVALS HLEN HMGET HEXISTS HSCAN"
  echo "  Lists:        LRANGE LLEN LINDEX"
  echo "  Sets:         SMEMBERS SCARD SISMEMBER SRANDMEMBER SSCAN"
  echo "  Sorted sets:  ZRANGE ZRANGEBYSCORE ZCARD ZSCORE ZCOUNT ZRANK ZSCAN"
  echo "  Streams:      XLEN XRANGE XREVRANGE XINFO"
  echo "  Server:       CONFIG GET"
}

die() {
  echo "ERROR: $1" >&2
  exit 1
}

# Validate arguments
if [[ $# -lt 2 ]]; then
  print_usage
  exit 1
fi

env_file="$1"
shift

if [[ ! -f "$env_file" ]]; then
  die "Environment file not found: ${env_file}"
fi

# Parse optional --max-lines
max_lines="${DEFAULT_MAX_LINES}"
if [[ "${1:-}" == "--max-lines" ]]; then
  if [[ $# -lt 2 ]]; then
    die "--max-lines requires a numeric argument"
  fi
  max_lines="$2"
  if ! [[ "$max_lines" =~ ^[0-9]+$ ]] || [[ "$max_lines" -eq 0 ]]; then
    die "--max-lines must be a positive integer, got: ${max_lines}"
  fi
  shift 2
fi

if [[ $# -lt 1 ]]; then
  die "No Redis command specified"
fi

# Parse env file for connection parameters
redis_host="${DEFAULT_HOST}"
redis_port="${DEFAULT_PORT}"
redis_password=""
redis_cluster="false"
matched_vars=0

while IFS= read -r line || [[ -n "$line" ]]; do
  # Strip carriage returns (Windows-style line endings)
  line="${line//$'\r'/}"

  # Skip comments and empty lines
  [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue

  # Strip optional 'export' prefix
  line="${line#export }"
  line="${line#export	}"

  # Strip inline comments (unquoted # preceded by whitespace)
  # Only strip if not inside quotes
  if [[ "$line" =~ ^[[:space:]]*([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*=[[:space:]]*(\"[^\"]*\"|\'[^\']*\'|[^#]*) ]]; then
    key="${BASH_REMATCH[1]}"
    val="${BASH_REMATCH[2]}"

    # Trim trailing whitespace
    val="${val%"${val##*[! 	]}"}"

    # Remove surrounding quotes (mutually exclusive: double OR single)
    if [[ "$val" =~ ^\"(.*)\"$ ]]; then
      val="${BASH_REMATCH[1]}"
    elif [[ "$val" =~ ^\'(.*)\'$ ]]; then
      val="${BASH_REMATCH[1]}"
    fi

    case "$key" in
      REDIS_HOST)     redis_host="$val"; matched_vars=$((matched_vars + 1)) ;;
      REDIS_PORT)     redis_port="$val"; matched_vars=$((matched_vars + 1)) ;;
      REDIS_PASSWORD) redis_password="$val"; matched_vars=$((matched_vars + 1)) ;;
      REDIS_CLUSTER)  redis_cluster="$val"; matched_vars=$((matched_vars + 1)) ;;
    esac
  fi
done < "$env_file"

if [[ "$matched_vars" -eq 0 ]]; then
  echo "Warning: no REDIS_* variables found in ${env_file}. Using defaults (${DEFAULT_HOST}:${DEFAULT_PORT})." >&2
fi

# Validate parsed connection parameters
if ! [[ "$redis_port" =~ ^[0-9]+$ ]]; then
  die "Invalid REDIS_PORT value: '${redis_port}'. Must be a number."
fi

if ! [[ "$redis_host" =~ ^[a-zA-Z0-9._-]+$ ]]; then
  die "Invalid REDIS_HOST value: '${redis_host}'. Must contain only alphanumeric characters, dots, hyphens, and underscores."
fi

# Validate command against allowlist
command_upper="${1^^}"

is_allowed=false
for allowed in "${ALLOWED_COMMANDS[@]}"; do
  if [[ "$command_upper" == "$allowed" ]]; then
    is_allowed=true
    break
  fi
done

if [[ "$is_allowed" == false ]]; then
  echo "BLOCKED: '${1}' is not a read-only command." >&2
  echo "" >&2
  print_usage >&2
  exit 1
fi

# Special case: CONFIG only allows GET subcommand
if [[ "$command_upper" == "CONFIG" ]]; then
  subcommand="${2:-}"
  subcommand_upper="${subcommand^^}"
  if [[ "$subcommand_upper" != "GET" ]]; then
    die "Only 'CONFIG GET' is allowed. '${1} ${subcommand}' is not a read-only operation."
  fi
fi

# Build redis-cli arguments
cli_args=(-h "$redis_host" -p "$redis_port")

if [[ "$redis_cluster" == "true" ]]; then
  cli_args+=(-c)
fi

# Use REDISCLI_AUTH env var instead of -a flag to avoid password exposure in process table
if [[ -n "$redis_password" ]]; then
  export REDISCLI_AUTH="$redis_password"
fi

# Test connectivity
if ! redis-cli "${cli_args[@]}" PING > /dev/null 2>&1; then
  echo "Cannot connect to Redis at ${redis_host}:${redis_port}." >&2
  echo "" >&2
  echo "Please establish your tunnel first. For example:" >&2
  echo "  ssh -L ${redis_port}:<redis-host>:<redis-port> <bastion-host>" >&2
  exit 1
fi

# Execute the command and handle output truncation
output=$(redis-cli "${cli_args[@]}" "$@")

if [[ -z "$output" ]]; then
  echo "$output"
  exit 0
fi

total_lines=$(printf '%s\n' "$output" | wc -l)

if [[ "$total_lines" -gt "$max_lines" ]]; then
  printf '%s\n' "$output" | head -n "$max_lines"
  echo ""
  echo "[Output truncated: showing ${max_lines} of ${total_lines} lines. Use --max-lines N to adjust]"
else
  echo "$output"
fi
