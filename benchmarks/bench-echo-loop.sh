#!/bin/bash

set -e

echo ""
echo "--- BENCH ECHO START ---"
echo ""

cd $(dirname "${BASH_SOURCE[0]}")
function cleanup {
    echo "--- BENCH ECHO DONE ---"
    kill -9 $(jobs -rp)
    wait $(jobs -rp) 2>/dev/null
}
trap cleanup EXIT

mkdir -p bin
$(pkill -9 zput_net_golang-echo-server || printf "")

function gobench {
    echo "--- $1 ---"
    if [ "$3" != "" ]; then
        go build -o $2 $3
    fi
    GOMAXPROCS=1 $2 --port $4 --loops $5 &

    sleep 1
    echo "*** 50 connections, 10 seconds, 6 byte packets"
    nl=$'\r\n'
    tcpkali --workers 1 -c 50 -T 60s -m "PING{$nl}" 127.0.0.1:$4
    echo "--- DONE ---"
    sleep 5
    echo ""
}

gobench "ZPUT-NET-GOLANG"  bin/zput_net_golang-echo-server ../example/echo/echo.go 58111 1
gobench "ZPUT-NET-GOLANG"  bin/zput_net_golang-echo-server ../example/echo/echo.go 58112 2
gobench "ZPUT-NET-GOLANG"  bin/zput_net_golang-echo-server ../example/echo/echo.go 58113 3
gobench "ZPUT-NET-GOLANG"  bin/zput_net_golang-echo-server ../example/echo/echo.go 58114 4
gobench "ZPUT-NET-GOLANG"  bin/zput_net_golang-echo-server ../example/echo/echo.go 58115 5
gobench "ZPUT-NET-GOLANG"  bin/zput_net_golang-echo-server ../example/echo/echo.go 58116 6


#Processes: 380 total, 2 running, 29 stuck, 349 sleeping, 2317 threads                                                     20:25:21
#Load Avg: 2.83, 2.51, 2.59  CPU usage: 7.48% user, 7.97% sys, 84.54% idle    SharedLibs: 178M resident, 60M data, 35M linkedit.
#MemRegions: 63322 total, 1272M resident, 62M private, 852M shared. PhysMem: 6054M used (1749M wired), 2137M unused.
#VM: 1946G vsize, 1371M framework vsize, 248923(0) swapins, 491070(0) swapouts.
#Networks: packets: 78406024/753G in, 78338758/753G out. Disks: 401438/5892M read, 74757/4680M written.


