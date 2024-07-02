#!/usr/bin/env bash
# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u

curl -s https://raw.githubusercontent.com/lluissm/license-header-checker/master/install.sh | bash

./bin/license-header-checker -a -r -i testdata ./hack/license_header.txt . go && [[ -z `git status -s` ]]