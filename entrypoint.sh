#!/bin/bash

# Check for required environment variables
: "${DB_DRIVER:=postgres}" # Default driver to 'postgres'
: "${DB_HOST:?Error: DB_HOST environment variable must be set}"
: "${DB_PORT:?Error: DB_PORT environment variable must be set}"
: "${DB_USERNAME:=postgres}"  # Default to 'postgres' if not set
: "${DB_NAME:=postgres}"  # Default to 'postgres' if not set
: "${TIMEOUT_SECONDS:=30}"  # Default timeout for PostgreSQL connection

# Logging function with timestamp
log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check connection using nc
check_connection() {
  nc -z -w 2 "$DB_HOST" "$DB_PORT" > /dev/null 2>&1
  return $?
}

log "Checking connection to PostgreSQL at $DB_HOST:$DB_PORT..."

# Wait for connection with timeout
start_time=$(date +%s)
while true; do
  if check_connection; then
    log "Connection to PostgreSQL successful!"
    break
  else
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    if [ "$elapsed" -ge "$TIMEOUT_SECONDS" ]; then
      log "Error: Failed to connect to PostgreSQL within $TIMEOUT_SECONDS seconds."
      exit 1
    fi
    log "Failed to connect. Retrying in 0.5 seconds..."
    sleep 0.5
  fi
done

log "Applying goose migrations..."

# Run migrations using uv to manage dependencies
if ! goose up; then
  log "Error: Failed to apply goose migrations."
  exit 1
fi
log "Goose migrations applied successfully!"

pwd
ls -r

log "Starting the application..."
exec "$@"
