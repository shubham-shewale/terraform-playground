#!/usr/bin/env python3
"""
Test Data Generation Script for CSPM Monitor
Generates realistic test data for unit and integration tests
"""

import json
import random
import uuid
from datetime import datetime, timezone, timedelta
from decimal import Decimal

def generate_security_findings(count=100):
    """Generate realistic security findings for testing"""

    severities = ['INFORMATIONAL', 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL']
    severity_weights = [0.4, 0.3, 0.2, 0.08, 0.02]  # Realistic distribution

    resources = [
        'aws:ec2:instance',
        'aws:s3:bucket',
        'aws:iam:user',
        'aws:iam:role',
        'aws:rds:db-instance',
        'aws:lambda:function',
        'aws:cloudtrail:trail',
        'aws:cloudwatch:log-group'
    ]

    titles = [
        'Security Group allows unrestricted SSH access',
        'S3 bucket is publicly readable',
        'IAM user has overly permissive policies',
        'RDS instance is not encrypted',
        'Lambda function has excessive timeout',
        'CloudTrail logging is disabled',
        'Root account has access keys',
        'Security group allows unrestricted RDP access',
        'S3 bucket lacks versioning',
        'IAM role has wildcard permissions'
    ]

    descriptions = [
        'This security group allows SSH access from 0.0.0.0/0, which is a security risk.',
        'The S3 bucket is configured with public read access, potentially exposing sensitive data.',
        'The IAM user has policies that grant more permissions than necessary for their role.',
        'The RDS database instance is not encrypted at rest, leaving data vulnerable.',
        'The Lambda function timeout is set too high, increasing cost and attack surface.',
        'CloudTrail logging is disabled, preventing audit trail generation.',
        'The root account has active access keys, which should be avoided.',
        'This security group allows RDP access from any IP address.',
        'The S3 bucket does not have versioning enabled, risking data loss.',
        'The IAM role has wildcard (*) permissions, which is overly permissive.'
    ]

    findings = []

    for i in range(count):
        # Generate timestamp within last 30 days
        days_ago = random.randint(0, 30)
        hours_ago = random.randint(0, 23)
        minutes_ago = random.randint(0, 59)

        timestamp = datetime.now(timezone.utc) - timedelta(
            days=days_ago,
            hours=hours_ago,
            minutes=minutes_ago
        )

        # Select severity based on weights
        severity = random.choices(severities, weights=severity_weights)[0]

        finding = {
            'Id': f'test-finding-{i:04d}-{uuid.uuid4().hex[:8]}',
            'Title': random.choice(titles),
            'Description': random.choice(descriptions),
            'Severity': {
                'Label': severity
            },
            'Resources': [{
                'Type': random.choice(resources),
                'Id': f'arn:aws:{random.choice(resources).split(":")[1]}:us-east-1:123456789012:resource/{uuid.uuid4().hex[:12]}'
            }],
            'AwsAccountId': '123456789012',
            'Region': random.choice(['us-east-1', 'us-west-2', 'eu-west-1', 'ap-southeast-1']),
            'CreatedAt': timestamp.isoformat(),
            'UpdatedAt': timestamp.isoformat(),
            'Compliance': {
                'Status': random.choice(['PASSED', 'FAILED', 'NOT_AVAILABLE'])
            },
            'Workflow': {
                'Status': random.choice(['NEW', 'ASSIGNED', 'IN_PROGRESS', 'RESOLVED'])
            }
        }

        findings.append(finding)

    return findings

def generate_dynamodb_items(findings):
    """Convert findings to DynamoDB items"""

    items = []

    for finding in findings:
        # Calculate TTL timestamp (90 days from now)
        ttl_timestamp = int((datetime.now(timezone.utc) + timedelta(days=90)).timestamp())

        item = {
            'id': finding['Id'],
            'severity': finding['Severity']['Label'],
            'timestamp': finding['CreatedAt'],
            'title': finding['Title'],
            'description': finding['Description'],
            'resource_type': finding['Resources'][0]['Type'],
            'resource_id': finding['Resources'][0]['Id'],
            'account_id': finding['AwsAccountId'],
            'region': finding['Region'],
            'raw_finding': json.dumps(finding, default=str),
            'ttl_timestamp': ttl_timestamp,
            'compliance_status': finding.get('Compliance', {}).get('Status', 'UNKNOWN'),
            'workflow_status': finding.get('Workflow', {}).get('Status', 'NEW')
        }

        # Convert any float values to Decimal for DynamoDB
        for key, value in item.items():
            if isinstance(value, float):
                item[key] = Decimal(str(value))

        items.append(item)

    return items

def generate_test_events():
    """Generate test events for Lambda testing"""

    events = {
        'security_hub_event': {
            'source': 'aws.securityhub',
            'detail': {
                'findings': generate_security_findings(5)
            }
        },

        'sqs_event': {
            'Records': [{
                'eventSource': 'aws:sqs',
                'body': json.dumps({
                    'detail': {
                        'findings': generate_security_findings(3)
                    }
                })
            }]
        },

        'manual_event': {
            'findings': generate_security_findings(2)
        },

        'empty_event': {
            'findings': []
        },

        'malformed_event': {
            'invalid': 'data'
        }
    }

    return events

def generate_api_test_cases():
    """Generate test cases for API testing"""

    test_cases = {
        'valid_requests': [
            {
                'method': 'GET',
                'path': '/health',
                'expected_status': 200
            },
            {
                'method': 'GET',
                'path': '/findings',
                'query_params': {'limit': '10'},
                'expected_status': 200
            },
            {
                'method': 'GET',
                'path': '/findings',
                'query_params': {'severity': 'HIGH'},
                'expected_status': 200
            },
            {
                'method': 'GET',
                'path': '/summary',
                'expected_status': 200
            }
        ],

        'invalid_requests': [
            {
                'method': 'GET',
                'path': '/findings',
                'query_params': {'severity': 'INVALID'},
                'expected_status': 400
            },
            {
                'method': 'GET',
                'path': '/findings',
                'query_params': {'limit': '1001'},
                'expected_status': 400
            },
            {
                'method': 'POST',
                'path': '/findings',
                'expected_status': 405
            }
        ],

        'edge_cases': [
            {
                'method': 'GET',
                'path': '/findings',
                'query_params': {'id': 'non-existent-id'},
                'expected_status': 404
            },
            {
                'method': 'GET',
                'path': '/findings',
                'query_params': {'limit': '0'},
                'expected_status': 400
            }
        ]
    }

    return test_cases

def main():
    """Main function to generate and save test data"""

    print("Generating test data for CSPM Monitor...")

    # Generate security findings
    print("Generating security findings...")
    findings = generate_security_findings(200)

    # Generate DynamoDB items
    print("Converting to DynamoDB format...")
    dynamodb_items = generate_dynamodb_items(findings)

    # Generate test events
    print("Generating test events...")
    test_events = generate_test_events()

    # Generate API test cases
    print("Generating API test cases...")
    api_test_cases = generate_api_test_cases()

    # Save to files
    output_dir = 'tests/testdata'
    import os
    os.makedirs(output_dir, exist_ok=True)

    # Save findings
    with open(f'{output_dir}/security_findings.json', 'w') as f:
        json.dump(findings, f, indent=2, default=str)

    # Save DynamoDB items
    with open(f'{output_dir}/dynamodb_items.json', 'w') as f:
        json.dump(dynamodb_items, f, indent=2, default=str)

    # Save test events
    with open(f'{output_dir}/test_events.json', 'w') as f:
        json.dump(test_events, f, indent=2, default=str)

    # Save API test cases
    with open(f'{output_dir}/api_test_cases.json', 'w') as f:
        json.dump(api_test_cases, f, indent=2, default=str)

    print("Test data generated successfully!")
    print(f"Files saved to: {output_dir}/")
    print(f"- {len(findings)} security findings")
    print(f"- {len(dynamodb_items)} DynamoDB items")
    print(f"- {len(test_events)} test events")
    print(f"- {sum(len(cases) for cases in api_test_cases.values())} API test cases")

if __name__ == '__main__':
    main()