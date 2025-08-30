#!/usr/bin/env python3
"""
AWS CSPM Monitor - API Lambda Function
Provides REST API endpoints for querying security findings
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

# Environment variables
DYNAMODB_TABLE_PARAM = os.environ.get('DYNAMODB_TABLE_PARAM', '/cspm-monitor/dynamodb-table-name')

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

def get_table():
    """Get DynamoDB table resource"""
    table_name = get_ssm_parameter(DYNAMODB_TABLE_PARAM)
    return dynamodb.Table(table_name)

def query_findings_by_severity(severity=None, limit=100):
    """Query findings by severity using GSI"""
    try:
        table = get_table()

        if severity:
            # Query specific severity
            response = table.query(
                IndexName='SeverityTimestampIndex',
                KeyConditionExpression=boto3.dynamodb.conditions.Key('severity').eq(severity),
                ScanIndexForward=False,  # Most recent first
                Limit=limit
            )
        else:
            # Scan all findings (less efficient, use with caution)
            response = table.scan(
                Limit=limit,
                FilterExpression=boto3.dynamodb.conditions.Attr('severity').exists()
            )

        items = response.get('Items', [])

        # Convert DynamoDB Decimal types to float for JSON serialization
        for item in items:
            for key, value in item.items():
                if isinstance(value, Decimal):
                    item[key] = float(value)

        return items

    except ClientError as e:
        logger.error(f"Failed to query findings: {e}")
        raise

def get_finding_by_id(finding_id):
    """Get a specific finding by ID"""
    try:
        table = get_table()
        response = table.get_item(Key={'id': finding_id})

        if 'Item' in response:
            item = response['Item']
            # Convert Decimal types
            for key, value in item.items():
                if isinstance(value, Decimal):
                    item[key] = float(value)
            return item
        else:
            return None

    except ClientError as e:
        logger.error(f"Failed to get finding {finding_id}: {e}")
        raise

def get_findings_summary():
    """Get summary statistics of findings using efficient query"""
    try:
        table = get_table()

        # Use GSI to get severity distribution efficiently
        severity_counts = {}
        total_findings = 0

        # Query each severity using the GSI
        severities = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFORMATIONAL']

        for severity in severities:
            try:
                response = table.query(
                    IndexName='SeverityTimestampIndex',
                    KeyConditionExpression=boto3.dynamodb.conditions.Key('severity').eq(severity),
                    Select='COUNT'
                )
                count = response.get('Count', 0)
                if count > 0:
                    severity_counts[severity] = count
                    total_findings += count
            except ClientError as e:
                logger.warning(f"Failed to query severity {severity}: {e}")
                continue

        # If no GSI results, fall back to limited scan (last resort)
        if total_findings == 0:
            logger.warning("GSI queries failed, falling back to limited scan")
            response = table.scan(
                Limit=1000,  # Limit to prevent excessive costs
                ProjectionExpression='severity'
            )
            items = response.get('Items', [])
            total_findings = len(items)

            for item in items:
                severity = item.get('severity', 'UNKNOWN')
                severity_counts[severity] = severity_counts.get(severity, 0) + 1

        return {
            'total_findings': total_findings,
            'severity_breakdown': severity_counts,
            'last_updated': datetime.now(timezone.utc).isoformat()
        }

    except ClientError as e:
        logger.error(f"Failed to get findings summary: {e}")
        raise

def create_response(status_code, body, cors=True):
    """Create API Gateway response"""
    response = {
        'statusCode': status_code,
        'headers': {
            'Content-Type': 'application/json',
            'Access-Control-Allow-Origin': '*' if cors else None,
            'Access-Control-Allow-Headers': 'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token' if cors else None,
            'Access-Control-Allow-Methods': 'GET,POST,OPTIONS' if cors else None,
        },
        'body': json.dumps(body, default=str)
    }

    # Remove None values
    response['headers'] = {k: v for k, v in response['headers'].items() if v is not None}

    return response

def lambda_handler(event, context):
    """Main Lambda handler function"""
    logger.info("CSPM Monitor API Lambda started")
    logger.info(f"Event: {json.dumps(event, indent=2)}")

    try:
        # Extract request information
        http_method = event.get('httpMethod', '')
        path = event.get('path', '')
        query_params = event.get('queryStringParameters', {}) or {}
        path_params = event.get('pathParameters', {}) or {}

        logger.info(f"Request: {http_method} {path}")

        # Handle CORS preflight
        if http_method == 'OPTIONS':
            return create_response(200, {'message': 'CORS preflight successful'})

        # Route requests
        if http_method == 'GET':
            if path.endswith('/findings'):
                # Validate and sanitize input parameters
                severity = query_params.get('severity')
                limit_param = query_params.get('limit', '100')

                # Validate severity
                if severity and severity not in ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFORMATIONAL']:
                    return create_response(400, {
                        'success': False,
                        'error': 'Invalid severity. Must be one of: CRITICAL, HIGH, MEDIUM, LOW, INFORMATIONAL',
                        'timestamp': datetime.now(timezone.utc).isoformat()
                    })

                # Validate and sanitize limit
                try:
                    limit = int(limit_param)
                    if limit < 1 or limit > 1000:
                        raise ValueError("Limit out of range")
                except (ValueError, TypeError):
                    return create_response(400, {
                        'success': False,
                        'error': 'Invalid limit. Must be a number between 1 and 1000',
                        'timestamp': datetime.now(timezone.utc).isoformat()
                    })

                if 'id' in query_params:
                    # Get specific finding
                    finding_id = query_params['id']

                    # Validate finding ID format (basic validation)
                    if not finding_id or len(finding_id) > 256:
                        return create_response(400, {
                            'success': False,
                            'error': 'Invalid finding ID format',
                            'timestamp': datetime.now(timezone.utc).isoformat()
                        })

                    finding = get_finding_by_id(finding_id)
                    if finding:
                        return create_response(200, {
                            'success': True,
                            'data': finding,
                            'timestamp': datetime.now(timezone.utc).isoformat()
                        })
                    else:
                        return create_response(404, {
                            'success': False,
                            'error': 'Finding not found',
                            'timestamp': datetime.now(timezone.utc).isoformat()
                        })

                else:
                    # Get findings list
                    findings = query_findings_by_severity(severity, limit)
                    return create_response(200, {
                        'success': True,
                        'data': findings,
                        'count': len(findings),
                        'timestamp': datetime.now(timezone.utc).isoformat()
                    })

            elif path.endswith('/summary'):
                # Get findings summary
                summary = get_findings_summary()
                return create_response(200, {
                    'success': True,
                    'data': summary,
                    'timestamp': datetime.now(timezone.utc).isoformat()
                })

            elif path.endswith('/health'):
                # Health check endpoint
                return create_response(200, {
                    'status': 'healthy',
                    'service': 'cspm-monitor-api',
                    'timestamp': datetime.now(timezone.utc).isoformat(),
                    'version': '1.0.0'
                })

        # Method not allowed
        return create_response(405, {
            'success': False,
            'error': 'Method not allowed',
            'timestamp': datetime.now(timezone.utc).isoformat()
        })

    except ValueError as e:
        logger.error(f"Validation error: {e}")
        return create_response(400, {
            'success': False,
            'error': 'Invalid request parameters',
            'timestamp': datetime.now(timezone.utc).isoformat()
        })

    except ClientError as e:
        logger.error(f"AWS error: {e}")
        return create_response(500, {
            'success': False,
            'error': 'Internal server error',
            'timestamp': datetime.now(timezone.utc).isoformat()
        })

    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        return create_response(500, {
            'success': False,
            'error': 'Internal server error',
            'timestamp': datetime.now(timezone.utc).isoformat()
        })

if __name__ == '__main__':
    # For local testing
    test_event = {
        'httpMethod': 'GET',
        'path': '/findings',
        'queryStringParameters': {
            'limit': '10'
        }
    }

    result = lambda_handler(test_event, None)
    print(json.dumps(result, indent=2))