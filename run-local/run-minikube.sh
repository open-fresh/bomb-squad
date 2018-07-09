#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="${SCRIPT_DIR}/.."
SHORT_SHA="$(git rev-parse --short HEAD)"
MK_PROFILE="bomb-squad"

source ${SCRIPT_DIR}/func.sh

function getBSImage() {
  echo $(${DOCKER} images --filter=reference='bomb-squad:'${SHORT_SHA} --format="{{ .Repository }}:{{ .Tag }}" | sort | uniq)
}

function getSSImage() {
  echo $(${DOCKER} images --filter=reference='ss:latest' --format="{{ .Repository }}:{{ .Tag }}" | sort | uniq)
}

function buildBS() {
  pushd ${PROJECT_DIR} > /dev/null
  ${MK} profile ${MK_PROFILE}
  eval "$(${MK} docker-env)"
  set -e
  #make clean
  make
  set -e
  popd > /dev/null
}

function buildSS() {
  pushd ${SCRIPT_DIR}/statspitter > /dev/null
  ${MK} profile ${MK_PROFILE}
  eval "$(${MK} docker-env)"
  set -e
  GOOS=linux GOARCH=amd64 go build -o ss
  docker build -t ss:latest .
  set -e
  popd > /dev/null
}

echo "Checking for ${MK_PROFILE} minikube profile..."
${MK} profile ${MK_PROFILE}
${MK} status
if [ $? -ne 0 ]; then
  echo "${MK_PROFILE} environment not found, creating ..."
  ${MK} profile ${MK_PROFILE}
  ${MK} start --cpus 4 --memory 8192 --kubernetes-version v1.9.4 --profile ${MK_PROFILE} --log_dir ${SCRIPT_DIR}/logs
fi

eval "$(${MK} docker-env -p ${MK_PROFILE})"
echo "${MK_PROFILE} minikube profile setup, continuing..."

buildBS
buildSS

echo "Applying ksonnet bits to minikube environment..."
pushd ${SCRIPT_DIR}/ksonnet > /dev/null
kubectl config use-context bomb-squad
ks param set bomb-squad imageTag ${SHORT_SHA} --env=minikube
${KS} apply --insecure-skip-tls-verify minikube 
popd > /dev/null

#kubectl delete pod -l app=bomb-squad 2> /dev/null
docker rm -f $(docker ps | grep /bin/bs | awk '{ print $1 }')
