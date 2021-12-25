#!/bin/bash

cd backend || exit

sudo rmmod uvcvideo
sudo modprobe uvcvideo nodrop=1 timeout=1000 quirks=0x80

export VERBOSE=0
export APIHOST="seawisely.com"
echo $APIHOST
cmd=./start
$cmd &


now=$(date +"%T")
echo "[$now] Running"
