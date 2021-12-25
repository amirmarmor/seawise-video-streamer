#!/bin/bash

cd backend
go build -o start
now=$(date +"%T")
echo "[$now] build done"

