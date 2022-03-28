#!/usr/bin/env bash

# Environment variables as input:
#   APISERVER_ENDPOINT (required), the url to the api service
#   APISERVER_AUTH_TOKEN (required), the token used to authenticate to the api
#   TOKEN_PATH, use this env variable to set the APISERVER_AUTH_TOKEN from a file
#   PERFORMANCE_TEST_TIME, the time it will take for a benchmark to run on one endpoint
#                          the default is 10
#   LIMIT, the number of clusters on the '/clusters' endpoint. We need a consistent nr
#          of clusters so the result are comparable. The default is 200

if [ -z "${APISERVER_ENDPOINT}" ]; then
    printf "ERROR: Missing env variable APISERVER_ENDPOINT.\n" 1>&2
    exit 1
fi

ENDPOINT="${APISERVER_ENDPOINT}"
NR_OF_SECS="${PERFORMANCE_TEST_TIME:-10}"  # If the env var is not set use the default 10

check_status_code()
{
    local err
    local statuscode=$1

    if (( "${statuscode}" >= 400 )); then
        err=$(jq -r '.errors.body' response.json)
        echo "ERROR: (${statuscode}) ${err}"
        return 1
    fi

    return 0
}

run_benckmarks()
{
    local endpoint=$1
    local secs=$2
    local token=$3

    printf 'Running benchmark on %s for %ss ...\n' "${endpoint}" "${secs}"

    # Check the request for errors or if is possible before running the performance test
    statuscode=$(curl -XGET -o response.json --silent --write-out "%{http_code}" -H "Authorization: Bearer ${token}" "${endpoint}")
    resp_code=$?
    if [[ ${resp_code} != 0 ]]; then # If request fails no need to run the test
        printf "ERROR: Could not contact server.\n\n" 1>&2
        sleep 0.1; return 1
    fi

    output=$(check_status_code "${statuscode}"); resp_code=$?
    if [[ ${resp_code} == 1 ]]; then # If request fails no need to run the test
        printf "%s\n\n" "${output}" 1>&2
        sleep 0.1; return 1
    fi

    # Run the performance tests
    hey -z "${secs}s" -o csv -H "Authorization: Bearer ${token}" "${endpoint}" | heyparser
    printf "\n" # Add an extra line for a better separation

    # The sleep is to avoind racecondition between stderr and stdout
    # and print the output in the right order to the terminal
    sleep 0.1; return 0
}

run_benckmarks "${ENDPOINT}/readyz" "${NR_OF_SECS}"

run_benckmarks "${ENDPOINT}/livez" "${NR_OF_SECS}"

if [ -z "${APISERVER_AUTH_TOKEN}" ]; then
    if [ -z "${TOKEN_PATH}" ]; then
        printf "ERROR: Missing env variable API_AUTH_TOKEN for the rest of the endpoints.\n" 1>&2
        exit 1
    fi
    APISERVER_AUTH_TOKEN=$(cat "${TOKEN_PATH}")
fi

TOKEN="${APISERVER_AUTH_TOKEN}"
LIMIT="${LIMIT:-200}"
OFFSET="${OFFSET:-0}"

run_benckmarks "${ENDPOINT}/api/v1/clusters?limit=${LIMIT}&offset=${OFFSET}" "${NR_OF_SECS}" "${TOKEN}"

# Before we can benchmark the last endpoint we need the name of a cluster
printf "Geting cluster name... "
statuscode=$(curl -XGET -o response.json --silent --write-out "%{http_code}" -H "Authorization: Bearer ${TOKEN}" "${ENDPOINT}/api/v1/clusters?limit=1&offset=0")

resp_code=$?
if [[ ${resp_code} != 0 ]]; then # If request fails no need to run the test
    printf "\nERROR: Could not contact server.\n\n" 1>&2
    exit 1
fi

output=$(check_status_code "${statuscode}"); rc=$?
if [[ ${rc} == 1 ]]; then
    printf "\nERROR: Failed to get cluster name: %s\n\n" "${output}" 1>&2
    exit 1
fi

cluster_name=$(jq -r '.items[0].name' response.json)
printf "Done! Got %s.\n" "${cluster_name}"

statuscode=$(curl -XGET -o response.json --silent --write-out "%{http_code}" -H "Authorization: Bearer ${TOKEN}" "${ENDPOINT}/api/v1/clusters/${cluster_name}")
output=$(check_status_code "${statuscode}"); rc=$?
if [[ ${rc} == 1 ]]; then
    exit 1
fi

run_benckmarks "${ENDPOINT}/api/v1/clusters/${cluster_name}" "${NR_OF_SECS}" "${TOKEN}"
