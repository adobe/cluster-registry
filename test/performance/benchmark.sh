#!/usr/bin/env bash

ENDPOINT=$1
NR_OF_SECS=$2
AUTH_TOKEN=$3

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
    statuscode=$(curl -XGET -o response.json --silent --write-out "%{http_code}" -H "Authorization: Bearer ${AUTH_TOKEN}" "${endpoint}")

    output=$(check_status_code "${statuscode}"); rc=$?
    if [[ ${rc} == 1 ]]; then
        # If the request fails no need to run the test
        printf "%s\n\n" "${output}" 1>&2
        sleep 0.1
        return 1
    fi

    # Run the performance tests
    hey -z "${secs}s" -o csv -H "Authorization: Bearer ${token}" "${endpoint}" | heyparser
    printf "\n" # Add an extra line for a better separation

    sleep 0.1 # To avoind racecondition between stderr and stdout in printing to the terminal
    return 0
}

run_benckmarks "${ENDPOINT}/readyz" "${NR_OF_SECS}"

run_benckmarks "${ENDPOINT}/livez" "${NR_OF_SECS}"

if [ -z "${AUTH_TOKEN}" ]; then
    printf "ERROR: Missing script parameter AUTH_TOKEN for the rest of the endpoints.\n" 1>&2
    exit 1
fi

run_benckmarks "${ENDPOINT}/api/v1/clusters" "${NR_OF_SECS}" "${AUTH_TOKEN}"

# Before we can benchmark the last endpoint we need the name of a cluster
printf "Geting cluster name... "
statuscode=$(curl -XGET -o response.json --silent --write-out "%{http_code}" -H "Authorization: Bearer ${AUTH_TOKEN}" "${ENDPOINT}/api/v1/clusters")
output=$(check_status_code "${statuscode}"); rc=$?
if [[ ${rc} == 1 ]]; then
    printf "\nERROR: Failed to get cluster name: %s\n" "${output}" 1>&2
    exit 1
fi
cluster_name=$(jq -r '.items[0].name' response.json)
printf "Done! Got %s.\n" "${cluster_name}"

statuscode=$(curl -XGET -o response.json --silent --write-out "%{http_code}" -H "Authorization: Bearer ${AUTH_TOKEN}" "${ENDPOINT}/api/v1/clusters/${cluster_name}")
output=$(check_status_code "${statuscode}"); rc=$?
if [[ ${rc} == 1 ]]; then
    exit 1
fi

run_benckmarks "${ENDPOINT}/api/v1/clusters/${cluster_name}" "${NR_OF_SECS}" "${AUTH_TOKEN}"
