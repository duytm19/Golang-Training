package stack

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3notifications"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type PreprocessorStackProps struct {
	awscdk.StackProps
	EnvName string
}

func NewPreprocessorStack(scope constructs.Construct, id string, props *PreprocessorStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, jsii.String(id), &sprops)

	envName := "dev"
	if props != nil && props.EnvName != "" {
		envName = props.EnvName
	}

	// 1. S3 Buckets
	inputBucket := awss3.NewBucket(stack, jsii.String("InputBucket"), &awss3.BucketProps{
		BucketName:        jsii.String(fmt.Sprintf("card-onboarding-input-bucket-%s", envName)),
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		AutoDeleteObjects: jsii.Bool(true),
	})

	outputBucket := awss3.NewBucket(stack, jsii.String("OutputBucket"), &awss3.BucketProps{
		BucketName:        jsii.String(fmt.Sprintf("card-onboarding-output-bucket-%s", envName)),
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		AutoDeleteObjects: jsii.Bool(true),
	})

	// 2. SQS Queue & DLQ
	dlq := awssqs.NewQueue(stack, jsii.String("PreprocessorDlq"), &awssqs.QueueProps{
		QueueName:       jsii.String(fmt.Sprintf("card-onboarding-file-preprocessor-dlq-%s", envName)),
		RetentionPeriod: awscdk.Duration_Days(jsii.Number(4)),
	})

	preprocessorQueue := awssqs.NewQueue(stack, jsii.String("PreprocessorQueue"), &awssqs.QueueProps{
		QueueName:         jsii.String(fmt.Sprintf("card-onboarding-file-preprocessor-sqs-%s", envName)),
		VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(60)),
		RetentionPeriod:   awscdk.Duration_Days(jsii.Number(4)),
		DeadLetterQueue: &awssqs.DeadLetterQueue{
			MaxReceiveCount: jsii.Number(3),
			Queue:           dlq,
		},
	})

	// 3. S3 Event Notification to SQS Queue
	inputBucket.AddEventNotification(
		awss3.EventType_OBJECT_CREATED,
		awss3notifications.NewSqsDestination(preprocessorQueue),
		&awss3.NotificationKeyFilter{
			Suffix: jsii.String(".csv"),
		},
	)

	// 4. Lambda Function
	workerQueueUrl := fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/card-onboarding-worker-sqs-%s", *stack.Region(), *stack.Account(), envName)

	preprocessorLambda := awslambda.NewFunction(stack, jsii.String("PreprocessorLambda"), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("card-onboarding-file-preprocessor-%s", envName)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Handler:      jsii.String("bootstrap"),
		Code:         awslambda.Code_FromAsset(jsii.String("./dist"), nil),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
		Environment: &map[string]*string{
			"AWS_REGION":     stack.Region(),
			"OUTPUT_BUCKET":  outputBucket.BucketName(),
			"WORKER_SQS_URL": jsii.String(workerQueueUrl),
		},
	})

	// 5. IAM Permissions (Least Privilege)
	inputBucket.GrantRead(preprocessorLambda, nil)
	outputBucket.GrantWrite(preprocessorLambda, nil, jsii.Strings("processed/*"))

	// Lambda consumes events from Preprocessor Queue
	awslambda.NewEventSourceMapping(stack, jsii.String("PreprocessorQueueTrigger"), &awslambda.EventSourceMappingProps{
		Target:         preprocessorLambda,
		EventSourceArn: preprocessorQueue.QueueArn(),
	})

	// Lambda needs SendMessage to Worker SQS Queue
	workerQueueArn := fmt.Sprintf("arn:aws:sqs:%s:%s:card-onboarding-worker-sqs-%s", *stack.Region(), *stack.Account(), envName)
	preprocessorLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("sqs:SendMessage"),
		Resources: jsii.Strings(workerQueueArn),
	}))

	return stack
}
