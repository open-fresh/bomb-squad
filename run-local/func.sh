#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="${SCRIPT_DIR}/.."

DOCKER=$(which docker) > /dev/null 2>&1
if [ -z "${DOCKER}" ]; then
  echo "Sorry, you need to install Docker. Exiting."
  exit 1
fi

MK=$(which minikube) >/dev/null 2>&1
if [[ -z ${MK} && "x${0}x" == "xrun-minikube.shx" ]]; then
  echo "Sorry, you need to install minikube. Exiting."
  exit 1
fi

MS=$(which minikube) >/dev/null 2>&1
if [ -z ${MK} ]; then
  echo "Sorry, you need to install minikube. Exiting."
  exit 1
fi

${MK} version | grep v0.25.1 > /dev/null
if [ $? -ne 0 ]; then
  echo "WARNING: You don't seem to be running minikube v0.25.1. Other versions may work,"
  echo "but as of 30/04/2018 you're likely to have a bad time."
fi

KS=$(which ks) > /dev/null 2>&1
if [ -z "${KS}" ]; then
  echo "Sorry, you need to install ksonnet. Exiting."
  exit 1
fi
