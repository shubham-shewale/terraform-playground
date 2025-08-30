import boto3
import json
import os
from datetime import datetime

dynamodb = boto3.resource('dynamodb')
table = dynamodb.Table(os.environ['DYNAMODB_TABLE'])
sns = boto3.client('sns')
sns_topic_arn = os.environ['SNS_TOPIC_ARN']
securityhub = boto3.client('securityhub')

def lambda_handler(event, context):
    findings = []

    # Check if this is triggered by Security Hub event
    if 'source' in event and event['source'] == 'aws.securityhub':
        # Process Security Hub findings from EventBridge
        if 'detail' in event:
            detail = event['detail']
            if 'findings' in detail:
                for finding in detail['findings']:
                    processed_finding = process_security_hub_finding(finding)
                    if processed_finding:
                        findings.append(processed_finding)
    else:
        # Fallback: Get all findings from Security Hub
        try:
            response = securityhub.get_findings()
            for finding in response['Findings']:
                processed_finding = process_security_hub_finding(finding)
                if processed_finding:
                    findings.append(processed_finding)
        except Exception as e:
            print(f"Error getting Security Hub findings: {e}")
            return {'statusCode': 500, 'body': json.dumps({'error': str(e)})}

    # Store findings in DynamoDB
    stored_count = 0
    for finding in findings:
        try:
            table.put_item(Item=finding)
            stored_count += 1

            # Send alert for high/critical severity findings
            if finding.get('severity', '').upper() in ['HIGH', 'CRITICAL']:
                sns.publish(
                    TopicArn=sns_topic_arn,
                    Message=json.dumps(finding),
                    Subject=f"{finding.get('severity', 'Unknown')} Severity Security Finding"
                )
        except Exception as e:
            print(f"Error storing finding: {e}")

    return {
        'statusCode': 200,
        'body': json.dumps({
            'findings_processed': len(findings),
            'findings_stored': stored_count
        })
    }

def process_security_hub_finding(sh_finding):
    """Process a Security Hub finding into our standardized format"""
    try:
        # Extract resource information
        resources = sh_finding.get('Resources', [])
        resource_id = resources[0].get('Id', 'Unknown') if resources else 'Unknown'
        resource_type = resources[0].get('Type', 'Unknown') if resources else 'Unknown'

        # Map Security Hub severity to our format
        severity_mapping = {
            'CRITICAL': 'Critical',
            'HIGH': 'High',
            'MEDIUM': 'Medium',
            'LOW': 'Low',
            'INFORMATIONAL': 'Info'
        }

        severity_label = sh_finding.get('Severity', {}).get('Label', 'UNKNOWN')
        severity = severity_mapping.get(severity_label, 'Unknown')

        # Create standardized finding
        finding = {
            'id': sh_finding.get('Id', f"sh-{resource_id}"),
            'resource_type': resource_type,
            'resource_id': resource_id,
            'title': sh_finding.get('Title', 'Security Finding'),
            'description': sh_finding.get('Description', ''),
            'severity': severity,
            'compliance_status': sh_finding.get('Compliance', {}).get('Status', 'UNKNOWN'),
            'remediation': sh_finding.get('Remediation', {}).get('Recommendation', {}).get('Text', ''),
            'source': 'Security Hub',
            'timestamp': sh_finding.get('CreatedAt', datetime.utcnow().isoformat()),
            'updated_at': sh_finding.get('UpdatedAt', datetime.utcnow().isoformat()),
            'aws_account_id': sh_finding.get('AwsAccountId', ''),
            'region': sh_finding.get('Region', ''),
            'types': sh_finding.get('Types', []),
            'product_arn': sh_finding.get('ProductArn', ''),
            'schema_version': sh_finding.get('SchemaVersion', '')
        }

        return finding

    except Exception as e:
        print(f"Error processing Security Hub finding: {e}")
        return None