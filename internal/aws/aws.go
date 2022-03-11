package aws

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/ivan-bokov/dstack-test/internal/stacktrace"
)

var (
	defaultFlushInterval = 5 * time.Second
)

type Logger struct {
	cwl           *cloudwatchlogs.Client
	flushInterval time.Duration
	seqToken      *string
	logGroup      string
	logStream     string
}

type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	LogGroup        string
	LogStream       string
	FlushInterval   time.Duration
}

type creds struct {
	keyID     string
	secretKey string
}

func (c creds) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     c.keyID,
		SecretAccessKey: c.secretKey,
	}, nil
}

func New(config *Config) (*Logger, error) {
	client := cloudwatchlogs.NewFromConfig(aws.Config{
		Region: config.Region,
		Credentials: creds{
			keyID:     config.AccessKeyID,
			secretKey: config.SecretAccessKey,
		},
		RetryMaxAttempts: 10,
		RetryMode:        aws.RetryModeStandard,
		ClientLogMode:    aws.LogRequestEventMessage | aws.LogResponseEventMessage,
	})

	return &Logger{
		cwl:           client,
		logGroup:      config.LogGroup,
		logStream:     config.LogStream,
		flushInterval: config.FlushInterval,
	}, nil
}

func (l *Logger) checkStreamExists(ctx context.Context) error {
	resp, err := l.cwl.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(l.logGroup),
	})
	if err != nil {
		return stacktrace.Wrap(err)
	}
	for _, logStream := range resp.LogStreams {
		if *logStream.LogStreamName == l.logStream {
			l.seqToken = logStream.UploadSequenceToken
			return nil
		}
	}
	_, err = l.cwl.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(l.logGroup),
		LogStreamName: aws.String(l.logStream),
	})

	return nil
}

func (l *Logger) checkGroupExists(ctx context.Context) error {
	resp, err := l.cwl.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{})
	if err != nil {
		return stacktrace.Wrap(err)
	}

	for _, logGroup := range resp.LogGroups {
		if *logGroup.LogGroupName == l.logGroup {
			return nil
		}
	}

	_, err = l.cwl.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(l.logGroup),
	})
	if err != nil {
		return stacktrace.Wrap(err)
	}
	return nil
}

func (l *Logger) publishButch(ctx context.Context, logEvents []types.InputLogEvent) {
	if len(logEvents) == 0 {
		return
	}
	var err error
	input := cloudwatchlogs.PutLogEventsInput{
		LogEvents:     logEvents,
		LogGroupName:  aws.String(l.logGroup),
		LogStreamName: aws.String(l.logStream),
		SequenceToken: l.seqToken,
	}
	resp, err := l.cwl.PutLogEvents(ctx, &input)
	if err != nil {
		log.Println("[ERROR] unable to put logs to cloudwatch", err)
	}
	if resp.NextSequenceToken != nil {
		l.seqToken = resp.NextSequenceToken
	}

}

var newTicker = func(freq time.Duration) *time.Ticker {
	return time.NewTicker(freq)
}

func (l *Logger) Write(ctx context.Context, logs chan string) {
	var err error
	var logEvents []types.InputLogEvent

	if err = l.checkGroupExists(ctx); err != nil {
		log.Println("[ERROR] unable to check group", err)
		return
	}
	if err = l.checkStreamExists(ctx); err != nil {
		log.Println("[ERROR] unable to check stream", err)
		return
	}
	if l.flushInterval <= 0 {
		l.flushInterval = defaultFlushInterval
	}
	ticker := newTicker(l.flushInterval)

	for {
		select {
		case <-ctx.Done():
			l.publishButch(context.Background(), logEvents)
			logEvents = logEvents[:0]
			return
		case <-ticker.C:
			l.publishButch(ctx, logEvents)
			logEvents = logEvents[:0]
		case event, ok := <-logs:
			if !ok {
				l.publishButch(ctx, logEvents)
				logEvents = logEvents[:0]
				log.Println("[ERROR] log channel closed")
				return
			}
			logEvents = append(logEvents, types.InputLogEvent{
				Message:   aws.String(event),
				Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
			})
		}
	}
}
