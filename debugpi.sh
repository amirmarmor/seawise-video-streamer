#!/bin/bash

sudo rmmod uvcvideo
sudo modprobe uvcvideo nodrop=1 timeout=10000 quirks=0x80

export VERBOSE=5
export BEHOST="192.168.10.8" #"192.168.1.11"
echo "connecting to " + $BEHOST
cmd=/home/pi/seawise-video-streamer/start
$cmd


now=$(date +"%T")
echo "[$now] Running"
