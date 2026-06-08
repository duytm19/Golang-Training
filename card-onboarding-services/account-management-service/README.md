# Account Management Service (Mock)

A mock utility service representing the bank's core savings interest rates calculator.

## 🧭 Responsibilities
* Mocks API endpoint responses for account interest lookups.
* Triggers failures for QA testing based on standard query strings.

## ⚠️ Failure Simulators
* `CUST_FAIL_INTEREST` -> Returns HTTP 500 (Internal Error)
* `CUST_NO_INTEREST` -> Returns HTTP 404 (Not Found)
