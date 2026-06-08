# Card Onboarding Workers

This repository contains the event-driven Lambda workers and AWS CDK v2 infrastructure code for the Card Onboarding Platform.

---

## ⚡ Lambda Workers Included

1. **`card-onboarding-file-preprocessor`**
   * **Responsibility**: Triggers from S3 `ObjectCreated` notifications routed through SQS. Downloads CSV batches, runs structure validation, outputs logs back to S3, and forwards single row items to the onboarding queue.
   * **Queue**: `card-onboarding-file-preprocessor-sqs-{env}`
   * **DLQ**: `card-onboarding-file-preprocessor-dlq-{env}`
   * **Retry Rules**: SQS retries message processing on any download/network failure up to 3 times before forwarding it to the DLQ.
2. **`card-onboarding-worker`**
   * **Responsibility**: Listens to accepted records. Conducts card-type and email business validation. Initiates transactions on `onboard-service` using generated clients.
   * **Queue**: `card-onboarding-worker-sqs-{env}`
   * **DLQ**: `card-onboarding-worker-dlq-{env}`
   * **Retry Rules**: 
     * **Business failure**: Marks message processed successfully (no retry, no DLQ).
     * **Technical failure (HTTP 5xx / timeouts)**: Fails Lambda, triggering SQS to retry up to 3 times before sending the item to the DLQ.

---

## ⚙️ Configuration Variables

| Variable | Description | Default |
|:---|:---|:---|
| `ENV` | Deployment environment (`dev`, `staging`, `prod`) | `dev` |
| `MAX_FILE_SIZE_BYTES` | Maximum CSV upload file size allowed | `10485760` (10MB) |
| `ONBOARD_SERVICE_URL` | Endpoint of the orchestrator API | `http://localhost:8080` |
| `S3_OUTPUT_BUCKET` | Destination bucket name for verification results | `card-onboarding-output-bucket-{env}` |

---

## 🚀 Commands

### Build Code
```bash
# Build Lambda executables
go build -o bin/preprocessor ./card-onboarding-file-preprocessor/cmd/card-onboarding-file-preprocessor/main.go
go build -o bin/worker ./card-onboarding-worker/cmd/card-onboarding-worker/main.go
```

### AWS CDK Synthesis
Synthesize CloudFormation templates:
```bash
# Synth preprocessor stack
cd card-onboarding-file-preprocessor && cdk synth

# Synth worker stack
cd card-onboarding-worker && cdk synth
```

### Run Locally
Local debugging requires mock triggers or local events:
```bash
# Run preprocessor unit tests with mocks
go test ./card-onboarding-file-preprocessor/...

# Run worker unit tests with mocks
go test ./card-onboarding-worker/...
```

### Test Suites
```bash
# Run all unit tests
go test ./...

# Run integration smoke tests
cd smoke-test && go test -v
```

### Deployment
Tripping deployments using the CDK:
```bash
# Deploy Preprocessor stack
cd card-onboarding-file-preprocessor && cdk deploy

# Deploy Worker stack
cd card-onboarding-worker && cdk deploy
```
