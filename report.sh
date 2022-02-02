#!/bin/bash

host="192.168.10.8:8080"
sn="$(cat /proc/cpuinfo | grep Serial | cut -d ' ' -f 2)"
url="http://$host/report"
ip="$(/sbin/ip -o -4 addr list wlan0 | awk '{print $4}' | cut -d)"

n=0

until [ "$n" -ge 5 ]
do
  echo "attempt $n - $url"
  curl -X POST -H "Content-Type: application/json" -d '{"sn": "'$sn'", "ip": "'$ip'"}' "$url" && break
  n=$((n+1))
  sleep 15
done

