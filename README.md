# Card Onboarding Platform 2026

An event-driven, resilient card onboarding platform designed to process batch card enrollment CSV files, perform validations, orchestrate downstream core banking operations, and persist status tracking.

---

## 🏗️ Architecture Overview

The system is built on an event-driven architecture using **Go 1.26**, **AWS Serverless (S3, SQS, DLQ, Lambda)**, **DynamoDB**, and **REST APIs (Gin, OpenAPI 3.0)**.

```
Manual CSV Upload -> [S3 Input Bucket]
                            │ (ObjectCreated Event)
                            ▼
           [Preprocessor SQS Queue]
                            │
                            ▼
           [Lambda: CSV File Preprocessor] -> Writes Results to [S3 Output Bucket]
                            │ (Publish Accepted Records)
                            ▼
            [Worker SQS Queue]
                            │
                            ▼
           [Lambda: Card Onboarding Worker]
                            │ (Invoke API Client)
                            ▼
           [ECS/Fargate: Onboard Service] (Orchestrator & State Machine)
            ├── Reads/Writes [DynamoDB Status Table]
            ├── Reads/Writes [DynamoDB Account Table]
            ├── Calls [Customer Management Service] (Mock)
            └── Calls [Account Management Service] (Mock)
```

---

## 📁 Repository Index

The platform is divided into two workspaces/repositories within this project directory:

### 1. ⚙️ [card-onboarding-services]
Contains the backend microservices run on ECS/Fargate:
* **`onboard-service`**: Core orchestrator implementing state machine, transaction resume logic, and DynamoDB storage.
* **`customer-management-service`**: Mock banking registry service with simulated validation and failure endpoints.
* **`account-management-service`**: Mock interest rates service with simulated validation and failure endpoints.

### 2. ⚡ [card-onboarding-workers]
Contains the event-driven serverless workers and deployment scripts:
* **`card-onboarding-file-preprocessor`**: SQS-triggered Lambda downloading, parsing, and validating CSV metadata and structures.
* **`card-onboarding-worker`**: SQS-triggered Lambda carrying out card/email business validations and calling the orchestrator.
* **`CDK Infrastructure`**: Infrastructure as Code stacks written in Go using AWS CDK v2.

---

## 🚀 Quick Start & Local Development

This project is configured as a **Go Workspace** via [go.work]to link all services and worker modules together.

### Root Orchestration Commands
You can run common workflows for the entire monorepo from the root directory using the [Makefile]:

```bash
# Run tests across all modules
make test

# Generate all OpenAPI models and clients
make generate

# Run linter across all modules
make lint
```

For component-specific setup and configuration:
* Read [Services Setup Guide]
* Read [Workers Setup Guide]

---

## 📈 Monitoring & Alerts

Observability is handled via **Datadog**:
* Custom counters and durations check file sizes, validation rates, and latency.
* Monitors automatically trigger alerts on any Dead-letter Queue (DLQ) messages or high service error rates (>5%).
