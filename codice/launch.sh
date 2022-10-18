#!/bin/bash

usage() {
    echo "Usage: ${0} [OPTIONS]"
    echo "Options:\n"
    echo  "\tFLAG   VALUES                    DESCRIPTION"
    echo  "\t-n     <number-of-peers>         Number of peers to spawn"
    echo  "\t-a     [auth | token | quorum]   Algorithm to use"
    echo  "\t-v                               Verbose modality"
    echo  "\t-d     [slow | medium | fast]    Level of net congestion" 
}

SIZE=3
PROVIDED_SIZE=0
VERBOSE=0
DC_FILE="./docker-compose.yml"
ENV_FILE="./.env"
LOG_DIR="./logs"
ALG=""
DELAY="fast"

# Parse command line options
while getopts "hn:a:vd:" opt; do
    case ${opt} in
        h ) 
            usage
            exit 0
            ;;
        n )
            SIZE=$OPTARG
            PROVIDED_SIZE=1   
            ;;
            
        v )
            VERBOSE=1
            ;;
        a )
            ALG=${OPTARG}
            ;;
        d )
            DELAY=${OPTARG}
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
    exit 1
fi

if [ "${DELAY}" != "fast" ] && [ "${DELAY}" != "medium" ] && [ "${DELAY}" != "slow" ]; then
    echo "[!] Invalid net congestion parameter, using default (fast)"
    DELAY="fast"
fi

echo  "[+] SUMMARY"

if [ "${PROVIDED_SIZE}" == "0" ]; then
    echo  "\t[*] Number of peers...........: ${SIZE} (default value)"
else
    echo  "\t[*] Number of peers...........: ${SIZE}"
fi
 
if [ "$VERBOSE" == "1" ]; then
    echo  "\t[*] Verbose output on log.....: ENABLED"
else 
    echo  "\t[*] Verbose output on log.....: DISABLED"
fi

echo  "\t[*] Type of service...........: ${ALG}"
echo  "\t[*] Environment variables file: ${ENV_FILE}"
echo  "\t[*] YAML file.................: ${DC_FILE}"
echo  "\t[*] Delay.....................: ${DELAY}"

# Load environment file
echo "VERBOSE=${VERBOSE}" > ${ENV_FILE}
echo "ALG=\"${ALG}\"" >> ${ENV_FILE}
echo "N=${SIZE}" >> ${ENV_FILE}
echo "DELAY=\"${DELAY}\"" >> ${ENV_FILE}
echo "[+] Created environment file for docker-compose"

# Run containers
echo "[+] Startup ${SIZE} peers, sequencer and register services ..."
rm -d -r ${LOG_DIR}
mkdir ${LOG_DIR}
docker compose build
docker compose up --scale peer_s=${SIZE}
