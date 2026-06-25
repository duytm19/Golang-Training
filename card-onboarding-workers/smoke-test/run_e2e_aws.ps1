# run_e2e_aws.ps1
# E2E Test execution script for AWS Dev environment

$inputBucket = "card-onboarding-input-bucket-dev"
$outputBucket = "card-onboarding-output-bucket-dev"
$statusTable = "onboard-service-request-status"
$detailsTable = "onboard-service-account-details"
$cluster = "card-onboarding-cluster"
$service = "onboard-service-service-5ru5qbza"
$lambdaName = "card-onboarding-worker-dev"

Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "Card Onboarding Platform - E2E Testing Script (AWS)" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan

# Check AWS CLI
$awsCheck = Get-Command aws -ErrorAction SilentlyContinue
if (-not $awsCheck) {
    Write-Error "AWS CLI is not installed or not in PATH. Please install it first."
    exit 1
}

# Auto-detect running task IP and update Lambda if needed
Write-Host "Detecting running ECS task for onboard-service..."
$taskArn = (aws ecs list-tasks --cluster $cluster --service-name $service --query "taskArns[0]" --output text).Trim()
if ($taskArn -eq "None" -or -not $taskArn) {
    Write-Error "No running ECS tasks found for service $service in cluster $cluster"
    exit 1
}
Write-Host "Found Task ARN: $taskArn"

$eni = (aws ecs describe-tasks --cluster $cluster --tasks $taskArn --query "tasks[0].attachments[1].details[?name=='networkInterfaceId'].value" --output text).Trim()
if ($eni -eq "None" -or -not $eni) {
    # Try index 0 if ServiceConnect was first or not present
    $eni = (aws ecs describe-tasks --cluster $cluster --tasks $taskArn --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" --output text).Trim()
}

if (-not $eni -or $eni -eq "None") {
    Write-Error "Could not retrieve ENI for task"
    exit 1
}
Write-Host "Found Network Interface: $eni"

$publicIp = (aws ec2 describe-network-interfaces --network-interface-ids $eni --query "NetworkInterfaces[0].Association.PublicIp" --output text).Trim()
if (-not $publicIp -or $publicIp -eq "None") {
    Write-Error "Could not retrieve Public IP for ENI $eni"
    exit 1
}
Write-Host "Detected Task Public IP: $publicIp" -ForegroundColor Green

Write-Host "Updating Lambda worker configuration with URL http://${publicIp}:8080 ..."
aws lambda update-function-configuration --function-name $lambdaName --environment "Variables={ONBOARD_SERVICE_URL=http://${publicIp}:8080,TIMEOUT_SECONDS=10}" | Out-Null
Write-Host "Waiting 5 seconds for Lambda update to propagate..."
Start-Sleep -Seconds 5

# Check if data directory exists
$dataDir = Join-Path $PSScriptRoot "data"
if (-not (Test-Path $dataDir)) {
    Write-Error "Data directory not found at $dataDir"
    exit 1
}

$files = @("happy_path.csv", "structural_error.csv", "business_error.csv", "resumption_error.csv")

foreach ($file in $files) {
    $filePath = Join-Path $dataDir $file
    if (-not (Test-Path $filePath)) {
        Write-Warning "File $file not found in $dataDir, skipping."
        continue
    }

    Write-Host ""
    Write-Host "--------------------------------------------------" -ForegroundColor Yellow
    Write-Host "Testing File: $file" -ForegroundColor Yellow
    Write-Host "--------------------------------------------------" -ForegroundColor Yellow

    # Upload file
    Write-Host "Uploading $file to s3://$inputBucket/$file ..."
    aws s3 cp $filePath "s3://$inputBucket/$file"
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to upload $file to S3"
        continue
    }

    # Wait for S3 event processing
    Write-Host "Waiting 8 seconds for S3 event trigger, Lambda execution, and ECS orchestration..."
    Start-Sleep -Seconds 8

    # Check Output Bucket
    Write-Host "Checking S3 Output Bucket for preprocess result..." -ForegroundColor Green
    aws s3 ls "s3://$outputBucket/processed/" --recursive | Where-Object { $_ -like "*$file*" }
    
    # Query DynamoDB entries if this is not a structural error file
    if ($file -ne "structural_error.csv") {
        Write-Host "Scanning DynamoDB status table for records..." -ForegroundColor Green
        
        # Use single quotes with backslash escaping to pass raw JSON properly to AWS CLI under PowerShell
        $jsonVal = '{\":f\": {\"S\": \"' + $file + '\"}}'
        aws dynamodb scan --table-name $statusTable --filter-expression "sourceFile = :f" --expression-attribute-values $jsonVal --query "Items[*].{CustomerId:customerId.S, OverallStatus:overallStatus.S, CustomerReg:customerRegistrationStatus.S, InterestRate:interestDetailsStatus.S, ErrorMsg:errorMessage.S}" --output table
    } else {
        Write-Host "This was a structural error test. No DynamoDB records should be created." -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "E2E Run completed. Please check the logs/output above." -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
