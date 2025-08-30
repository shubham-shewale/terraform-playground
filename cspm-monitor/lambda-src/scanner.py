#!/usr/bin/env python3
"""
AWS CSPM Monitor - Security Findings Scanner Lambda Function
Processes Security Hub findings and stores them in DynamoDB
"""

import json
import os
import boto3
import logging
from datetime import datetime, timezone
from decimal import Decimal
from botocore.exceptions import ClientError

# Configure logging
logger = logging.getLogger()
logger.setLevel(logging.INFO)

# Initialize AWS clients
dynamodb = boto3.resource('dynamodb')
securityhub = boto3.client('securityhub')
sns = boto3.client('sns')

# Environment variables
DYNAMODB_TABLE_PARAM = os.environ.get('DYNAMODB_TABLE_PARAM', '/cspm-monitor/dynamodb-table-name')
SNS_TOPIC_ARN_PARAM = os.environ.get('SNS_TOPIC_ARN_PARAM', '/cspm-monitor/sns-topic-arn')
DYNAMODB_TTL_DAYS = int(os.environ.get('DYNAMODB_TTL_DAYS', '90'))

# SSM Parameter Store for configuration
ssm = boto3.client('ssm')

def get_ssm_parameter(name):
    """Retrieve parameter from SSM Parameter Store"""
    try:
        response = ssm.get_parameter(Name=name, WithDecryption=True)
        return response['Parameter']['Value']
    except ClientError as e:
        logger.error(f"Failed to retrieve SSM parameter {name}: {e}")
        raise

def calculate_ttl_timestamp(days_from_now):
    """Calculate TTL timestamp for DynamoDB"""
    now = datetime.now(timezone.utc)
    ttl_datetime = now.replace(hour=0, minute=0, second=0, microsecond=0)  # Midnight UTC
    ttl_datetime = ttl_datetime + timedelta(days=days_from_now)
    return int(ttl_datetime.timestamp())

def process_finding(finding):
    """Process a single Security Hub finding"""
    try:
        # Extract key information
        finding_id = finding.get('Id', '')
        severity = finding.get('Severity', {}).get('Label', 'INFORMATIONAL')
        title = finding.get('Title', '')
        description = finding.get('Description', '')
        resource_type = finding.get('Resources', [{}])[0].get('Type', '')
        resource_id = finding.get('Resources', [{}])[0].get('Id', '')
        account_id = finding.get('AwsAccountId', '')
        region = finding.get('Region', '')

        # Create timestamp
        timestamp = datetime.now(timezone.utc).isoformat()

        # Prepare DynamoDB item
        item = {
            'id': finding_id,
            'severity': severity,
            'timestamp': timestamp,
            'title': title,
            'description': description,
            'resource_type': resource_type,
            'resource_id': resource_id,
            'account_id': account_id,
            'region': region,
            'raw_finding': json.dumps(finding, default=str),
            'ttl_timestamp': calculate_ttl_timestamp(DYNAMODB_TTL_DAYS)
        }

        # Convert any float values to Decimal for DynamoDB
        for key, value in item.items():
            if isinstance(value, float):
                item[key] = Decimal(str(value))

        return item

    except Exception as e:
        logger.error(f"Error processing finding: {e}")
        return None

def send_alert(severity, message, finding_id):
    """Send alert via SNS"""
    try:
        if severity in ['CRITICAL', 'HIGH']:
            topic_arn = get_ssm_parameter(SNS_TOPIC_ARN_PARAM)
            subject = f"CSPM Monitor - {severity} Security Finding"

            sns.publish(
                TopicArn=topic_arn,
                Subject=subject,
                Message=json.dumps({
                    'severity': severity,
                    'message': message,
                    'finding_id': finding_id,
                    'timestamp': datetime.now(timezone.utc).isoformat()
                }, indent=2)
            )
            logger.info(f"Alert sent for {severity} finding: {finding_id}")

    except Exception as e:
        logger.error(f"Failed to send alert: {e}")

def lambda_handler(event, context):
    """Main Lambda handler function"""
    logger.info("CSPM Monitor Scanner Lambda started")
    logger.info(f"Event: {json.dumps(event, indent=2)}")

    try:
        # Get DynamoDB table name from SSM
        table_name = get_ssm_parameter(DYNAMODB_TABLE_PARAM)
        table = dynamodb.Table(table_name)

        # Process findings from Security Hub
        findings_processed = 0
        findings_stored = 0

        # Handle different event sources
        if 'source' in event and event['source'] == 'aws.securityhub':
            # Direct Security Hub event
            findings = event.get('detail', {}).get('findings', [])
        elif 'Records' in event:
            # SQS event (from EventBridge DLQ)
            for record in event['Records']:
                if record.get('eventSource') == 'aws:sqs':
                    sqs_body = json.loads(record['body'])
                    findings = sqs_body.get('detail', {}).get('findings', [])
                    break
            else:
                findings = []
        else:
            # Manual invocation or test
            findings = event.get('findings', [])

        logger.info(f"Processing {len(findings)} findings")

        for finding in findings:
            findings_processed += 1

            # Process the finding
            item = process_finding(finding)
            if not item:
                continue

            try:
                # Store in DynamoDB
                table.put_item(Item=item)
                findings_stored += 1

                # Send alert for high-severity findings
                severity = item.get('severity', 'INFORMATIONAL')
                if severity in ['CRITICAL', 'HIGH']:
                    message = f"Security Finding: {item.get('title', 'Unknown')}"
                    send_alert(severity, message, item.get('id', ''))

                logger.info(f"Stored finding: {item['id']} (Severity: {severity})")

            except ClientError as e:
                logger.error(f"Failed to store finding {item.get('id', 'unknown')}: {e}")
                continue

        # Log summary
        logger.info(f"Processing complete: {findings_processed} processed, {findings_stored} stored")

        return {
            'statusCode': 200,
            'body': json.dumps({
                'message': 'Security findings processed successfully',
                'findings_processed': findings_processed,
                'findings_stored': findings_stored,
                'timestamp': datetime.now(timezone.utc).isoformat()
            })
        }

    except Exception as e:
        logger.error(f"Lambda execution failed: {e}")
        raise

if __name__ == '__main__':
    # For local testing
    test_event = {
        'findings': [{
            'Id': 'test-finding-123',
            'Title': 'Test Security Finding',
            'Description': 'This is a test finding',
            'Severity': {'Label': 'HIGH'},
            'Resources': [{'Type': 'AwsEc2Instance', 'Id': 'i-1234567890abcdef0'}],
            'AwsAccountId': '123456789012',
            'Region': 'us-east-1'
        }]
    }

    result = lambda_handler(test_event, None)
    print(json.dumps(result, indent=2))