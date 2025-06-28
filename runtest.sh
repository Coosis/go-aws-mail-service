#!/bin/sh
export AMQP_URI=amqp://andy:realpassword@localhost:5672/
export QUEUE_NAME=mail
export SEND_TO="mandy@gmail.com"

go run ./test/runtest.go
