package main

import (
	"os"
	"context"
	"time"

	. "github.com/Coosis/go-mail-service/model"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	uri := os.Getenv("AMQP_URI")
	if uri == "" {
		panic("AMQP URI is required, please specify it with $AMQP_URI environment variable")
	}

	queue := os.Getenv("QUEUE_NAME")
	if queue == "" {
		panic("Queue name is required, please specify it with $QUEUE_NAME environment variable")
	}

	send_to := os.Getenv("SEND_TO")
	if send_to == "" {
		panic("Send to address is required, please specify it with $SEND_TO environment variable")
	}

	conn, err := amqp.Dial(uri)
	if err != nil {
		panic("failed to connect to RabbitMQ, " + err.Error())
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic("failed to open a channel, " + err.Error())
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic("failed to declare a queue, " + err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testJob := &MailJob{
		To:      send_to,
		Subject: "Test Email",
		Message: "This is a test email from the Go Mail Service.",
	}
	data, err := testJob.Marshal()
	if err != nil {
		panic("failed to marshal mail job, " + err.Error())
	}

	err = ch.PublishWithContext(ctx,
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
	if err != nil {
		panic("failed to publish a message, " + err.Error())
	}
}
