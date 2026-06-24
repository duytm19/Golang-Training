package stack

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type WorkerStackProps struct {
	awscdk.StackProps
	EnvName string
}

func NewWorkerStack(scope constructs.Construct, id string, props *WorkerStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, jsii.String(id), &sprops)

	envName := "dev"
	if props != nil && props.EnvName != "" {
		envName = props.EnvName
	}

	// 1. SQS Queue & DLQ
	dlq := awssqs.NewQueue(stack, jsii.String("WorkerDlq"), &awssqs.QueueProps{
		QueueName:       jsii.String(fmt.Sprintf("card-onboarding-worker-dlq-%s", envName)),
		RetentionPeriod: awscdk.Duration_Days(jsii.Number(4)),
	})

	workerQueue := awssqs.NewQueue(stack, jsii.String("WorkerQueue"), &awssqs.QueueProps{
		QueueName:         jsii.String(fmt.Sprintf("card-onboarding-worker-sqs-%s", envName)),
		VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(60)),
		RetentionPeriod:   awscdk.Duration_Days(jsii.Number(4)),
		DeadLetterQueue: &awssqs.DeadLetterQueue{
			MaxReceiveCount: jsii.Number(3),
			Queue:           dlq,
		},
	})

	// 2. Lambda Function
	onboardServiceUrl := fmt.Sprintf("http://onboard-service.%s.local:8080", envName)

	workerLambda := awslambda.NewFunction(stack, jsii.String("WorkerLambda"), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("card-onboarding-worker-%s", envName)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Handler:      jsii.String("bootstrap"),
		Code:         awslambda.Code_FromAsset(jsii.String("./dist"), nil),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
		Environment: &map[string]*string{
			"ONBOARD_SERVICE_URL": jsii.String(onboardServiceUrl),
			"TIMEOUT_SECONDS":     jsii.String("10"),
		},
	})

	// 3. IAM Permissions & Event Source Mapping
	workerQueue.GrantConsumeMessages(workerLambda)

	awslambda.NewEventSourceMapping(stack, jsii.String("WorkerQueueTrigger"), &awslambda.EventSourceMappingProps{
		Target:         workerLambda,
		EventSourceArn: workerQueue.QueueArn(),
	})

	return stack
}
