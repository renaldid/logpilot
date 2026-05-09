#!/usr/bin/env bash
# Streams fake log entries to $1 (log directory) for the VHS demo.
# Each service writes to its own file; logpilot watches all three.
set -euo pipefail

LOGDIR="${1:-/tmp/logpilot-demo}"
mkdir -p "$LOGDIR"

# --- pre-seed with some history so the UI isn't empty at startup ---
cat >> "$LOGDIR/api.log" <<'EOF'
{"time":"2026-05-09T08:00:01Z","level":"INFO","msg":"server listening","port":8080}
{"time":"2026-05-09T08:00:02Z","level":"INFO","msg":"connected to database","host":"postgres:5432"}
{"time":"2026-05-09T08:00:03Z","level":"DEBUG","msg":"config loaded","env":"production"}
{"time":"2026-05-09T08:00:04Z","level":"INFO","msg":"GET /health 200 1ms"}
{"time":"2026-05-09T08:00:05Z","level":"INFO","msg":"GET /api/users 200 12ms"}
EOF

cat >> "$LOGDIR/worker.log" <<'EOF'
{"time":"2026-05-09T08:00:01Z","level":"INFO","msg":"worker started","concurrency":4}
{"time":"2026-05-09T08:00:02Z","level":"INFO","msg":"queue connected","queue":"jobs"}
{"time":"2026-05-09T08:00:03Z","level":"INFO","msg":"job processed","id":"a1b2c3","duration":"45ms"}
{"time":"2026-05-09T08:00:04Z","level":"WARN","msg":"queue depth high","depth":120}
EOF

cat >> "$LOGDIR/db.log" <<'EOF'
2026-05-09 08:00:01 INFO  database system is ready to accept connections
2026-05-09 08:00:02 INFO  autovacuum launcher started
2026-05-09 08:00:03 INFO  checkpoint starting: immediate
2026-05-09 08:00:04 INFO  checkpoint complete: wrote 134 buffers (0.8%)
EOF

# --- live stream ---
API_MSGS=(
  '{"level":"INFO","msg":"POST /api/orders 201 23ms"}'
  '{"level":"INFO","msg":"GET /api/products 200 8ms"}'
  '{"level":"WARN","msg":"rate limit approaching","client":"10.0.0.5"}'
  '{"level":"INFO","msg":"GET /api/users/42 200 5ms"}'
  '{"level":"ERROR","msg":"payment gateway timeout","order_id":"ord-991"}'
  '{"level":"INFO","msg":"POST /api/auth/login 200 31ms"}'
  '{"level":"DEBUG","msg":"cache miss","key":"user:42:profile"}'
  '{"level":"INFO","msg":"GET /api/health 200 1ms"}'
)
WORKER_MSGS=(
  '{"level":"INFO","msg":"job queued","type":"send_email","id":"e8f9"}'
  '{"level":"INFO","msg":"job processed","id":"e8f9","duration":"120ms"}'
  '{"level":"WARN","msg":"retry attempt 1","job":"img-resize","reason":"timeout"}'
  '{"level":"INFO","msg":"job processed","id":"img-001","duration":"890ms"}'
  '{"level":"ERROR","msg":"job failed","id":"pay-007","error":"connection refused"}'
  '{"level":"INFO","msg":"queue drained"}'
)
DB_MSGS=(
  "2026-05-09 08:00:10 INFO  connection received: host=api"
  "2026-05-09 08:00:11 INFO  statement: SELECT * FROM users WHERE id=42"
  "2026-05-09 08:00:12 WARN  slow query detected: 1234ms"
  "2026-05-09 08:00:13 INFO  statement: INSERT INTO orders VALUES (...)"
  "2026-05-09 08:00:14 ERROR could not connect to replica"
)

i=0
while true; do
  NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # api
  MSG="${API_MSGS[$((i % ${#API_MSGS[@]}))]}"
  echo "${MSG/\{/\{\"time\":\"$NOW\",}" >> "$LOGDIR/api.log"

  sleep 0.4

  # worker (every other tick)
  if (( i % 2 == 0 )); then
    WMSG="${WORKER_MSGS[$((i / 2 % ${#WORKER_MSGS[@]}))]}"
    echo "${WMSG/\{/\{\"time\":\"$NOW\",}" >> "$LOGDIR/worker.log"
  fi

  # db (every 3rd tick)
  if (( i % 3 == 0 )); then
    echo "${DB_MSGS[$((i / 3 % ${#DB_MSGS[@]}))]}" >> "$LOGDIR/db.log"
  fi

  (( i++ )) || true
done
