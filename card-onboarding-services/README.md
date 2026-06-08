# Card Onboarding Services

This repository houses the core microservices for the Card Onboarding Platform. These services run on AWS ECS/Fargate and handle API orchestration, customer registration, and account details.

---

## 🛠️ Microservices Included

1. **`onboard-service`**
   * **Responsibility**: Orchestrates onboarding transactions. Operates a state machine to write step logs to DynamoDB and handles resume procedures on error.
   * **DynamoDB Usage**:
     * `onboard-service-request-status` (Workflow states tracking)
     * `onboard-service-account-details` (Processed bank details records)
2. **`customer-management-service`**
   * **Responsibility**: Mock banking customer registration ledger. Supports failure simulation.
3. **`account-management-service`**
   * **Responsibility**: Mock interest rates service. Supports failure simulation.

---

## 🌐 API Specifications & Endpoints

### 1. `onboard-service`
* **Swagger Specs**: `onboard-service/swagger-internal.yaml`
* **Generated Client SDK**: `onboard-service/pkg/onboard`
* **Endpoints**:
  * `POST /internal/cards/onboard` (Orchestrates single onboarding job)
  * `GET /internal/cards/{customerId}/status` (Retrieves onboarding workflow state)
  * `GET /health` (System health-check)

### 2. `customer-management-service` (Mock)
* **Swagger Specs**: `customer-management-service/swagger-internal.yaml`
* **Generated Client SDK**: `customer-management-service/pkg/customer`
* **Endpoints**:
  * `POST /internal/customers/register`
  * `GET /internal/customers/{customerId}`
  * `GET /health`

### 3. `account-management-service` (Mock)
* **Swagger Specs**: `account-management-service/swagger-internal.yaml`
* **Generated Client SDK**: `account-management-service/pkg/account`
* **Endpoints**:
  * `GET /internal/accounts/{customerId}/interest-details`
  * `GET /health`

---

## ⚙️ Configuration Variables

The following environment variables configure the service runtime:

| Variable | Description | Default |
|:---|:---|:---|
| `PORT` | Port for the HTTP server to listen on | `8080` |
| `ENV` | Environment identifier (`dev`, `staging`, `prod`) | `dev` |
| `DYNAMODB_TABLE_STATUS` | Status tracking table name | `onboard-service-request-status` |
| `DYNAMODB_TABLE_DETAILS` | Account details table name | `onboard-service-account-details` |
| `CUSTOMER_SERVICE_URL` | Base URL for the Customer Management Service | `http://localhost:8081` |
| `ACCOUNT_SERVICE_URL` | Base URL for the Account Management Service | `http://localhost:8082` |

---

## 🚀 Commands

### Local Code Generation
API interfaces and Go clients are generated automatically using `oapi-codegen`:
```bash
# Generate clients and server stubs for all services
make generate

# Validate swagger files
make swagger-validate

# Verify generated code matches specs (Fails if out of date)
make generate-check
```

### Run Locally
Start each service using standard Go runs:
```bash
# Run Onboard Service
go run onboard-service/main.go

# Run Customer Mock Service
go run customer-management-service/main.go

# Run Account Mock Service
go run account-management-service/main.go
```

### Build Docker Images
```bash
# Build service containers
docker build -t onboard-service:latest ./onboard-service
docker build -t customer-management-service:latest ./customer-management-service
docker build -t account-management-service:latest ./account-management-service
```

### Test Suites
```bash
# Run all unit tests
make test

# Run code coverage report
make coverage

# Run smoke integration tests
make smoke-test
```

### Deployment
To build and deploy the container services to AWS:
```bash
make deploy
```
