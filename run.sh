#!/bin/sh
export AWS_REGION=us-east-2
export AWS_ACCESS_KEY_ID=AAAAAAAAAAAAAAAAAAAA
export AWS_SECRET_ACCESS_KEY=BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB

export AMQP_URI=amqp://andy:realpassword@localhost:5672/
export QUEUE_NAME=mail
export SEND_FROM="andy@gmail.com"
export WORKER_NAME="andys-worker"
go run .
