#!/bin/bash

usage() {
    echo "Usage: ${0} [OPTIONS]"
    echo "Options:\n"
    echo  "\tFLAG   VALUES                    DESCRIPTION"
    echo  "\t-n     [one | many(5)]           Number of peers to spawn"
    echo  "\t-a     [auth | token | quorum]   Algorithm to use"
    echo  "\t-d     [slow | medium | fast]    Level of net congestion"
}

SIZE=""
ACTUALSIZE=1
ALG=""
DELAY=""

while getopts "hn:a:d:" opt; do
    case ${opt} in
        h ) 
            usage
            exit 0
            ;;
        n )
            SIZE=${OPTARG}
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

if [ ${SIZE} == "many" ] && [ ${DELAY} != "slow" ]; then
    ACTUALSIZE=5
else 
    if [ ${SIZE} == "many" ] && [ ${DELAY} == "slow" ]; then
        ACTUALSIZE=3
    fi
fi

sh launch.sh -n $ACTUALSIZE -a ${ALG} -v -d ${DELAY} &
sleep 60
sh down.sh
echo "\n\n"
cd test
export DELAY="${DELAY}"
go test