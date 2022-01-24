#!/bin/bash

BEHOST="192.168.0.30:8080"
n=0

until [ "$n" -ge 5 ]
do
  url="http://$BEHOST/report"
  echo "attempt $n - $url"
  curl "$url" && break
  n=$((n+1))
  sleep 15
done

