#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.."; pwd)"

echo 'Loading local environment variables...'
source ${ROOT_DIR}/.env.local

echo 'Create dynamodb schema...'
aws dynamodb delete-table --table-name ${DB_TABLE_NAME} --endpoint-url $DB_ENDPOINT > /dev/null 2>&1 || true
aws dynamodb create-table --cli-input-json file://${ROOT_DIR}/database/schema.json --endpoint-url $DB_ENDPOINT > /dev/null

echo 'Populate database with dummy data..'
go run ${ROOT_DIR}/database/import.go --input-file ${ROOT_DIR}/database/dummy-data.yaml
