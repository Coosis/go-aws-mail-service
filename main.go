package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"

	. "github.com/Coosis/go-mail-service/model"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	workername := os.Getenv("WORKER_NAME")
	if workername == "" {
		panic("please specify it with $WORKER_NAME environment variable")
	}
	pid := strconv.Itoa(os.Getpid())
	uid := strconv.Itoa(os.Getuid())
	consumerName := workername + "-" + pid + "-" + uid

	uri := os.Getenv("AMQP_URI")
	if uri == "" {
		panic("AMQP URI is required, please specify it with $AMQP_URI environment variable")
	}

	queue := os.Getenv("QUEUE_NAME")
	if queue == "" {
		panic("Queue name is required, please specify it with $QUEUE_NAME environment variable")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

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

	msgs, err := ch.Consume(
		q.Name,
		consumerName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic("failed to register a consumer, " + err.Error())
	}

	go func() {
		for msg := range msgs {
			println("Received message:", string(msg.Body))
			to := string(msg.Body)
			job, err := UnmarshalMailJob(msg.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to unmarshal mail job: %s\n", err.Error())
				continue
			}
			if err := sendMail(ctx, job); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send email to %s: %s\n", to, err.Error())
			} else {
				fmt.Printf("Email sent successfully to %s\n", to)
			}
		}
	}()

	<-sig
}

func sendMail(ctx context.Context, job *MailJob) error {
	accessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKeyId == "" {
		return fmt.Errorf("$AWS_ACCESS_KEY_ID is required, please set it in the environment")
	}
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		return fmt.Errorf("$AWS_SECRET_ACCESS_KEY is required, please set it in the environment")
	}
	sendFrom := os.Getenv("SEND_FROM")
	if sendFrom == "" {
		return fmt.Errorf("$SEND_FROM is required, please set it in the environment")
	}

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(
			credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     accessKeyId,
					SecretAccessKey: secretAccessKey,
				},
			},
		),
	)
	if err != nil {
		return fmt.Errorf("unable to load SDK config, %s", err.Error())
	}

	client := ses.NewFromConfig(cfg)
	out, err := client.ListIdentities(ctx, &ses.ListIdentitiesInput{
		IdentityType: types.IdentityTypeEmailAddress,
	})
	if err != nil {
		return fmt.Errorf("failed to list identities, %s", err.Error())
	}

	for _, id := range out.Identities {
		println("Identity:", id)
	}

	_, err = client.SendEmail(ctx, &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{job.To},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data:    aws.String(job.Subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &types.Body{
				Text: &types.Content{
					Data:    aws.String(job.Message),
					Charset: aws.String("UTF-8"),
				},
			},
		},
		Source: aws.String(sendFrom),
	})
	if err != nil {
		return fmt.Errorf("failed to send email, %s", err.Error())
	}
	return nil
}
