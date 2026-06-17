package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/org/card-onboarding-workers/card-onboarding-file-preprocessor/internal/infra/stack"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	envName := os.Getenv("ENV_NAME")
	if envName == "" {
		envName = "dev"
	}

	stackProps := &stack.PreprocessorStackProps{
		StackProps: awscdk.StackProps{
			Env: &awscdk.Environment{
				Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
				Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
			},
		},
		EnvName: envName,
	}

	stackName := "CardOnboardingPreprocessorStack-" + envName
	stack.NewPreprocessorStack(app, stackName, stackProps)

	app.Synth(nil)
}
