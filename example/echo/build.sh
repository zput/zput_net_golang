#!/bin/bash

set -e

#curdir=`dirname $(readlink -f $0)`
#cd $curdir
appname=echo_d

CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o ${appname} echo.go
