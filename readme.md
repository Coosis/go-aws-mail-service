# go-mail-service
A simple mail service written in Go using aws sdk v2 and rabbitmq aqmp for 
message jobs.

# !! NOTE !!
The `Compose.yaml` file is not for the email worker, it's for the 
rabbitmq service. You can use it to spin up a rabbitmq service for 
development purposes.

# Usage
Check out the `run.sh` and `runtest.sh`.
