# Customer Management Service (Mock)

A mock utility service representing the bank's core customer ledger.

## 🧭 Responsibilities
* Mocks API endpoint responses for customer lookup and enrollment.
* Triggers failures for QA testing based on standard query strings.

## ⚠️ Failure Simulators
* `CUST_FAIL_REGISTER` -> Returns HTTP 500 (Internal Error)
* `CUST_BAD_REQUEST` -> Returns HTTP 400 (Bad Request)
