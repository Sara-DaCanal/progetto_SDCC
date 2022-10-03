#!/bin/bash

usage() {
    echo "Usage: ${0} [OPTIONS]"
    echo "Options:"
    echo -e "\tFLAG   VALUES                    DESCRIPTION"
    echo -e "\t-n     <number-of-peers>         Number of peers to spawn"
    echo -e "\t-a     [auth | token | quorum]   Algorithm to use"
    echo -e "\t-v                               Verbose modality"
}

SIZE=3
PROVIDED_SIZE=0
VERBOSE=0
DC_FILE="./docker-compose.yml"
ENV_FILE="./.env"
LOG_DIR="./logs"
ALG=""

# Parse command line options
while getopts ":h:n:a:v" opt; do
    case ${opt} in
        h ) 
            usage
            exit 0
            ;;
        n )
            PROVIDED_SIZE=1
            SIZE=$OPTARG
            ;;
        v )
            VERBOSE=1
            ;;
        a )
            ALG=${OPTARG}
            ;;
        ? )
            usage
            exit 1 
    esac
done
shift $((OPTIND -1))

# Dump of configurations 
if [ "${ALG}" != "auth" ] && [ "${ALG}" != "quorum" ] && [ "${ALG}" != "token" ]; then
    echo "[!] Select a valid type of algorithm: auth | quorum | token"
    #exit 1
fi

echo -e "[+] SUMMARY"

if [ "${PROVIDED_SIZE}" == "0" ]; then
    echo -e "\t[*] Number of peers...........: ${SIZE} (default value)"
else
    echo -e "\t[*] Number of peers...........: ${SIZE}"
fi
 
if [ "$VERBOSE" == "1" ]; then
    echo -e "\t[*] Verbose output on log.....: ENABLED"
else 
    echo -e "\t[*] Verbose output on log.....: DISABLED"
fi

echo -e "\t[*] Type of service...........: ${ALG}"
echo -e "\t[*] Environment variables file: ${ENV_FILE}"
echo -e "\t[*] YAML file.................: ${DC_FILE}"

# Load environment file
echo "VERBOSE=${VERBOSE}" > ${ENV_FILE}
echo "ALG=\"${ALG}\"" >> ${ENV_FILE}
echo "N=${SIZE}" >> ${ENV_FILE}
echo "[+] Created environment file for docker-compose"

# Run containers
echo -e "[+] Startup ${SIZE} peers, sequencer and register services ..."
rm -d -r ${LOG_DIR}
mkdir ${LOG_DIR}
docker compose build
docker compose up --scale peer_s=${SIZE}
