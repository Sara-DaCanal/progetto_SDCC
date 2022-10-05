#!/bin/sh

DC_FILE="./docker-compose.yml"
ENV_FILE="./.env"
PROJ_NAME=


echo "[+] Stop mutual-exclusion group components ..."

# -v flag is needed to remove unnamed volumes created by binding
docker-compose -f ${DC_FILE} --env-file ${ENV_FILE} down -v

# Autogenerated environment file is deleted after shutdown
rm -f ${ENV_FILE}
echo "[+] Autogenerated environment file has been successfully deleted, bye bye"