#!/bin/sh

echo "hello"

#*****************************************************************
#************************ Building binary ******************
#*****************************************************************

go version
# echo $GOCACHE
# export GOCACHE=cache
env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main main.go

zip stripe-webhook.zip main

aws lambda update-function-code \
    --function-name  stripe-webhook \
    --zip-file fileb://./stripe-webhook.zip

