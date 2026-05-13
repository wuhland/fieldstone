#!/bin/sh
set -e

# Generate the MD5 auth hash PgBouncer expects: md5 + md5(password || username)
PGHASH="md5$(printf '%s%s' "${POSTGRES_PASSWORD}" "${POSTGRES_USER}" | md5sum | cut -d' ' -f1)"

mkdir -p /etc/pgbouncer

# Each service connects to a named PgBouncer "database" that maps to the
# real PostgreSQL database with the correct search_path override.
cat > /etc/pgbouncer/pgbouncer.ini <<EOF
[databases]
identity = host=${POSTGRES_HOST:-postgres} port=5432 dbname=${POSTGRES_DB} options="-c search_path=identity,public"
permits  = host=${POSTGRES_HOST:-postgres} port=5432 dbname=${POSTGRES_DB} options="-c search_path=permits,public"
requests = host=${POSTGRES_HOST:-postgres} port=5432 dbname=${POSTGRES_DB} options="-c search_path=requests,public"
records  = host=${POSTGRES_HOST:-postgres} port=5432 dbname=${POSTGRES_DB} options="-c search_path=records,public"
audit    = host=${POSTGRES_HOST:-postgres} port=5432 dbname=${POSTGRES_DB} options="-c search_path=audit,public"
webhooks = host=${POSTGRES_HOST:-postgres} port=5432 dbname=${POSTGRES_DB} options="-c search_path=webhooks,public"

[pgbouncer]
listen_addr        = 0.0.0.0
listen_port        = 5432
auth_type          = md5
auth_file          = /etc/pgbouncer/userlist.txt
pool_mode          = transaction
max_client_conn    = 1000
default_pool_size  = 20
min_pool_size      = 2
reserve_pool_size  = 5
server_idle_timeout = 600
log_connections    = 0
log_disconnections = 0
EOF

# Write credentials file
cat > /etc/pgbouncer/userlist.txt <<EOF
"${POSTGRES_USER}" "${PGHASH}"
EOF

exec pgbouncer /etc/pgbouncer/pgbouncer.ini
