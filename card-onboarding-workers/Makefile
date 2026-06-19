.PHONY: all test infra-synth-preprocessor infra-synth-worker infra-synth-all

all: test infra-synth-all

test:
	@echo "Running tests in card-onboarding-file-preprocessor..."
	cd card-onboarding-file-preprocessor && go test ./...
	@echo "Running tests in card-onboarding-worker..."
	cd card-onboarding-worker && go test ./...

infra-synth-preprocessor:
	@echo "Synthesizing File Preprocessor CDK Stack..."
	cd card-onboarding-file-preprocessor && cdk synth

infra-synth-worker:
	@echo "Synthesizing Processor Worker CDK Stack..."
	cd card-onboarding-worker && cdk synth

infra-synth-all: infra-synth-preprocessor infra-synth-worker
