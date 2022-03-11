package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/ivan-bokov/dstack-test/internal/aws"
	"github.com/ivan-bokov/dstack-test/internal/docker"
)

func main() {

	dockerImage := kingpin.Flag("docker-image", "A name of a Docker image").Required().String()
	bashCommand := kingpin.Flag("bash-command", "A bash command (to run inside this Docker image)").Required().String()
	cloudwatchGroup := kingpin.Flag("cloudwatch-group", "A name of an AWS CloudWatch group").Required().String()
	cloudwatchStream := kingpin.Flag("cloudwatch-stream", "A name of an AWS CloudWatch stream").Required().String()
	awsAccessKeyID := kingpin.Flag("aws-access-key-id", "").Required().String()
	awsSecretAccessKey := kingpin.Flag("aws-secret-access-key", "").Required().String()
	awsRegion := kingpin.Flag("aws-region", "").Required().String()

	kingpin.Parse()
	sigCh, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	dock, err := docker.New(*dockerImage, *bashCommand)
	if err != nil {
		log.Println("[ERROR] unable to create Docker client", err)
		os.Exit(2)
	}
	err = dock.Run(sigCh)
	if err != nil {
		log.Println("[ERROR] unable to create container", err)
		cancel()
		os.Exit(2)
	}
	defer dock.Close()
	logger, err := aws.New(&aws.Config{
		AccessKeyID:     *awsAccessKeyID,
		SecretAccessKey: *awsSecretAccessKey,
		Region:          *awsRegion,
		LogGroup:        *cloudwatchGroup,
		LogStream:       *cloudwatchStream,
		FlushInterval:   3 * time.Second,
	})
	if err != nil {
		log.Println("[ERROR] unable to create AWS Logger", err)
		cancel()
		os.Exit(2)
	}
	go logger.Write(sigCh, dock.Logs())
	dock.Wait(sigCh)
	cancel()

}
