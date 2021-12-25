#!/bin/bash

cd backend || exit

sudo rmmod uvcvideo
sudo modprobe uvcvideo nodrop=1 timeout=1000 quirks=0x80

export VERBOSE=5
export APIHOST="192.168.1.16"
echo $APIHOST
cmd=./start
$cmd


now=$(date +"%T")
echo "[$now] Running"
