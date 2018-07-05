#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="${SCRIPT_DIR}/.."
MK_PROFILE="bomb-squad"

source ${SCRIPT_DIR}/func.sh

echo "Deleting ksonnet bits from minikube environment..."
pushd ${SCRIPT_DIR}/ksonnet > /dev/null
kubectl config use-context bomb-squad
${KS} delete --insecure-skip-tls-verify minikube 
git checkout app.yaml
popd > /dev/null
