#!/bin/sh
set -eu

# Check that the required environment variables are set
: "${GOOSE_DBSTRING:?GOOSE_DBSTRING is required}"

export GOOSE_DRIVER=postgres

exec goose -dir /migrations "$@"