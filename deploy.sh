#! /usr/bin/env bash
# Generic deployment script

DEPLOY_CONFIG_FILE="./deploy_config.sh"

#set -x
set -e

if [ -f $DEPLOY_CONFIG_FILE ]; then
    echo "Loading Config File [$DEPLOY_CONFIG_FILE]"
    source $DEPLOY_CONFIG_FILE
else
    echo "Deployment Config File [$DEPLOY_CONFIG_FILE] Missing. Can't Continue."
    exit -1
fi

LATEST_BUILD=$(ls -t builds/rbl_* | head -1)
LATEST_BUILD_NAME=$(basename $LATEST_BUILD)
scp -i $DEPLOY_USER_CERT -P $DEPLOY_PORT $LATEST_BUILD ${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_BUILD_TARGET}/
ssh -i $DEPLOY_USER_CERT -p $DEPLOY_PORT ${DEPLOY_USER}@${DEPLOY_HOST} "ln -sf '${DEPLOY_BUILD_TARGET}/${LATEST_BUILD_NAME}' ${DEPLOY_LINK_TARGET}"
