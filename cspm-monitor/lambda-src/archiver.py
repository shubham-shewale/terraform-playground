#!/usr/bin/env python3
"""
AWS CSPM Monitor - Data Archiver Lambda Function
Archives expired security findings from DynamoDB to S3
"""

import json
import os
import boto3
import logging
import gzip
from datetime import datetime, timezone, timedelta
from decimal import Decimal
from botocore.exceptions import ClientError

# Configure logging
logger = logging.getLogger()
logger.setLevel(logging.INFO)

# Initialize AWS clients
dynamodb = boto3.resource('dynamodb')
s3 = boto3.client('s3')

# Environment variables
DYNAMODB_TABLE_PARAM = os.environ.get('DYNAMODB_TABLE_PARAM', '/cspm-monitor/dynamodb-table-name')
S3_ARCHIVE_BUCKET = os.environ.get('S3_ARCHIVE_BUCKET', '')
RETENTION_DAYS = int(os.environ.get('RETENTION_DAYS', '90'))

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

def get_expired_findings(table, cutoff_timestamp):
    """Query for findings that have expired based on TTL"""
    try:
        # Scan for items with TTL timestamp less than cutoff
        response = table.scan(
            FilterExpression=boto3.dynamodb.conditions.Attr('ttl_timestamp').lt(cutoff_timestamp)
        )

        items = response.get('Items', [])

        # Handle pagination if there are more items
        while 'LastEvaluatedKey' in response:
            response = table.scan(
                FilterExpression=boto3.dynamodb.conditions.Attr('ttl_timestamp').lt(cutoff_timestamp),
                ExclusiveStartKey=response['LastEvaluatedKey']
            )
            items.extend(response.get('Items', []))

        logger.info(f"Found {len(items)} expired findings")
        return items

    except ClientError as e:
        logger.error(f"Failed to query expired findings: {e}")
        raise

def archive_findings_to_s3(findings, bucket_name):
    """Archive findings to S3 with compression"""
    try:
        if not bucket_name:
            logger.warning("No S3 archive bucket configured, skipping archival")
            return 0

        # Create archive filename with timestamp
        timestamp = datetime.now(timezone.utc).strftime('%Y%m%d_%H%M%S')
        key = f"security-findings-archive-{timestamp}.json.gz"

        # Convert Decimal types to float for JSON serialization
        serializable_findings = []
        for finding in findings:
            serializable_finding = {}
            for key, value in finding.items():
                if isinstance(value, Decimal):
                    serializable_finding[key] = float(value)
                else:
                    serializable_finding[key] = value
            serializable_findings.append(serializable_finding)

        # Create archive data
        archive_data = {
            'metadata': {
                'archived_at': datetime.now(timezone.utc).isoformat(),
                'total_findings': len(serializable_findings),
                'retention_days': RETENTION_DAYS
            },
            'findings': serializable_findings
        }

        # Compress and upload to S3
        json_data = json.dumps(archive_data, indent=2, default=str)
        compressed_data = gzip.compress(json_data.encode('utf-8'))

        s3.put_object(
            Bucket=bucket_name,
            Key=key,
            Body=compressed_data,
            ContentType='application/json',
            ContentEncoding='gzip',
            Metadata={
                'archived-at': datetime.now(timezone.utc).isoformat(),
                'finding-count': str(len(serializable_findings)),
                'retention-days': str(RETENTION_DAYS)
            },
            ServerSideEncryption='AES256'
        )

        logger.info(f"Archived {len(serializable_findings)} findings to s3://{bucket_name}/{key}")
        return len(serializable_findings)

    except ClientError as e:
        logger.error(f"Failed to archive findings to S3: {e}")
        raise

def delete_archived_findings(table, findings):
    """Delete archived findings from DynamoDB"""
    try:
        deleted_count = 0

        # Batch delete in groups of 25 (DynamoDB limit)
        for i in range(0, len(findings), 25):
            batch = findings[i:i+25]

            with table.batch_writer() as writer:
                for finding in batch:
                    writer.delete_item(Key={'id': finding['id']})
                    deleted_count += 1

        logger.info(f"Deleted {deleted_count} findings from DynamoDB")
        return deleted_count

    except ClientError as e:
        logger.error(f"Failed to delete archived findings: {e}")
        raise

def lambda_handler(event, context):
    """Main Lambda handler function"""
    logger.info("CSPM Monitor Archiver Lambda started")
    logger.info(f"Event: {json.dumps(event, indent=2)}")

    try:
        # Get DynamoDB table
        table_name = get_ssm_parameter(DYNAMODB_TABLE_PARAM)
        table = dynamodb.Table(table_name)

        # Calculate cutoff timestamp (current time minus retention period)
        cutoff_datetime = datetime.now(timezone.utc) - timedelta(days=RETENTION_DAYS)
        cutoff_timestamp = int(cutoff_datetime.timestamp())

        logger.info(f"Archiving findings older than {cutoff_datetime.isoformat()}")

        # Get expired findings
        expired_findings = get_expired_findings(table, cutoff_timestamp)

        if not expired_findings:
            logger.info("No expired findings to archive")
            return {
                'statusCode': 200,
                'body': json.dumps({
                    'message': 'No expired findings to archive',
                    'findings_processed': 0,
                    'findings_archived': 0,
                    'findings_deleted': 0,
                    'timestamp': datetime.now(timezone.utc).isoformat()
                })
            }

        # Archive to S3 first
        archived_count = archive_findings_to_s3(expired_findings, S3_ARCHIVE_BUCKET)

        # Only delete from DynamoDB if archival was successful
        if archived_count == len(expired_findings):
            # All findings archived successfully, safe to delete
            deleted_count = delete_archived_findings(table, expired_findings)

            if deleted_count == len(expired_findings):
                logger.info(f"Archival complete: {len(expired_findings)} processed, {archived_count} archived, {deleted_count} deleted")
                return {
                    'statusCode': 200,
                    'body': json.dumps({
                        'message': 'Security findings archived successfully',
                        'findings_processed': len(expired_findings),
                        'findings_archived': archived_count,
                        'findings_deleted': deleted_count,
                        'cutoff_timestamp': cutoff_datetime.isoformat(),
                        'timestamp': datetime.now(timezone.utc).isoformat()
                    })
                }
            else:
                logger.error(f"Partial deletion failure: {deleted_count}/{len(expired_findings)} findings deleted")
                # Attempt to restore archived data or alert for manual intervention
                return {
                    'statusCode': 500,
                    'body': json.dumps({
                        'message': 'Partial archival failure - manual intervention required',
                        'findings_processed': len(expired_findings),
                        'findings_archived': archived_count,
                        'findings_deleted': deleted_count,
                        'error': 'Failed to delete all archived findings from DynamoDB',
                        'timestamp': datetime.now(timezone.utc).isoformat()
                    })
                }
        else:
            logger.error(f"Archival failure: {archived_count}/{len(expired_findings)} findings archived")
            return {
                'statusCode': 500,
                'body': json.dumps({
                    'message': 'Archival failed - no findings deleted to prevent data loss',
                    'findings_processed': len(expired_findings),
                    'findings_archived': archived_count,
                    'findings_deleted': 0,
                    'error': 'Failed to archive all findings to S3',
                    'timestamp': datetime.now(timezone.utc).isoformat()
                })
            }

    except Exception as e:
        logger.error(f"Lambda execution failed: {e}")
        return {
            'statusCode': 500,
            'body': json.dumps({
                'message': 'Archival failed',
                'error': str(e),
                'timestamp': datetime.now(timezone.utc).isoformat()
            })
        }

if __name__ == '__main__':
    # For local testing
    test_event = {
        'source': 'aws.events',
        'detail-type': 'Scheduled Event',
        'detail': {}
    }

    result = lambda_handler(test_event, None)
    print(json.dumps(result, indent=2))