#!/bin/bash

set -e

echo ""
echo "--- PPRROF START ---"
echo ""

cd $(dirname "${BASH_SOURCE[0]}")
function cleanup {
    echo "--- PRINT processID DONE ---"
    echo $(jobs -rp)
}
trap cleanup EXIT

mkdir -p bin
$(pkill -9 pprof-test-server || printf "")

function gobench {
    echo "--- $1 ---"
    if [ "$3" != "" ]; then
        go build -o $2 $3
    fi
    GOMAXPROCS=1 $2 --port $4 --loops 1 &

    sleep 10
    echo "*** 50 connections, 10 seconds, 6 byte packets"
    nl=$'\r\n'
    tcpkali --workers 1 -c 50 -T 20s -m "PING{$nl}" 127.0.0.1:$4
    echo "--- DONE ---"
    echo ""
}

gobench "GEV"  bin/pprof-test-server ../example/echo/echo.go 50001

#sleep 120
