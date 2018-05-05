#!/bin/bash

while :
do
    clear
    go test
    inotifywait -q -e close_write *.go
done
