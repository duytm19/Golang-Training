# Card Onboarding File Preprocessor

An SQS-triggered Lambda worker that acts as the entrance gate for onboarding batch card data.

## 🧭 Responsibilities
* Parses incoming SQS S3 notifications.
* Validates CSV metadata: size limits, file readability, extension checking.
* Parses rows and confirms structural soundness (exactly 6 columns matching standard headers).
* Uploads dynamic processing status result `.csv` back to S3.
* Publishes valid records to SQS for downstream worker ingestion.
