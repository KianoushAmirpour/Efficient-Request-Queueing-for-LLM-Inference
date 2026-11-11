#!/bin/sh
set -eu

: "${GOOSE_DBSTRING:?GOOSE_DBSTRING is required}"

export GOOSE_DRIVER=postgres

exec goose -dir /migrations "$@"