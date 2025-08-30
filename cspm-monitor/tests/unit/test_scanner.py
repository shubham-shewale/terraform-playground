#!/usr/bin/env python3
"""
Unit tests for CSPM Monitor Scanner Lambda function
"""

import json
import pytest
from unittest.mock import patch, MagicMock
from decimal import Decimal
from datetime import datetime, timezone, timedelta

# Import the Lambda function from the built ZIP file
import sys
import os
import zipfile
import tempfile
import importlib.util

def load_lambda_function(zip_path, module_name):
    """Load Lambda function from ZIP file"""
    with zipfile.ZipFile(zip_path, 'r') as zip_ref:
        # Extract to temporary directory
        with tempfile.TemporaryDirectory() as temp_dir:
            zip_ref.extractall(temp_dir)

            # Import the module
            spec = importlib.util.spec_from_file_location(
                module_name,
                os.path.join(temp_dir, f"{module_name}.py")
            )
            module = importlib.util.module_from_spec(spec)
            sys.modules[module_name] = module
            spec.loader.exec_module(module)
            return module

# Load the Lambda function from ZIP file
lambda_src_dir = os.path.join(os.path.dirname(__file__), '../../lambda-src')
zip_path = os.path.join(lambda_src_dir, 'scanner.zip')

if os.path.exists(zip_path):
    scanner_module = load_lambda_function(zip_path, 'scanner')
    lambda_handler = scanner_module.lambda_handler
    get_ssm_parameter = scanner_module.get_ssm_parameter
    calculate_ttl_timestamp = scanner_module.calculate_ttl_timestamp
    process_finding = scanner_module.process_finding
    send_alert = scanner_module.send_alert
else:
    # Fallback to direct import for development
    sys.path.insert(0, lambda_src_dir)
    from scanner import (
        lambda_handler,
        get_ssm_parameter,
        calculate_ttl_timestamp,
        process_finding,
        send_alert
    )


class TestGetSSMParameter:
    """Test SSM parameter retrieval"""

    def setup_method(self):
        """Setup for each test method"""
        self.mock_ssm = MagicMock()

    def test_get_ssm_parameter_success(self):
        """Test successful SSM parameter retrieval"""
        with patch('boto3.client') as mock_boto3_client:
            mock_boto3_client.return_value = self.mock_ssm
            self.mock_ssm.get_parameter.return_value = {
                'Parameter': {'Value': 'test-table-name'}
            }

            result = get_ssm_parameter('/test/param')
            assert result == 'test-table-name'
            self.mock_ssm.get_parameter.assert_called_once_with(
                Name='/test/param',
                WithDecryption=True
            )

    def test_get_ssm_parameter_error(self):
        """Test SSM parameter retrieval error"""
        with patch('boto3.client') as mock_boto3_client:
            mock_boto3_client.return_value = self.mock_ssm
            from botocore.exceptions import ClientError
            self.mock_ssm.get_parameter.side_effect = ClientError(
                {'Error': {'Code': 'ParameterNotFound'}}, 'GetParameter'
            )

            with pytest.raises(ClientError) as exc_info:
                get_ssm_parameter('/test/param')

            assert exc_info.value.response['Error']['Code'] == 'ParameterNotFound'

    def test_get_ssm_parameter_encrypted_value(self):
        """Test SSM parameter with encrypted value"""
        with patch('boto3.client') as mock_boto3_client:
            mock_boto3_client.return_value = self.mock_ssm
            self.mock_ssm.get_parameter.return_value = {
                'Parameter': {'Value': 'encrypted-secret-value'}
            }

            result = get_ssm_parameter('/secure/database/password')
            assert result == 'encrypted-secret-value'

            # Verify WithDecryption was used
            call_args = self.mock_ssm.get_parameter.call_args
            assert call_args[1]['WithDecryption'] is True


class TestCalculateTTLTimestamp:
    """Test TTL timestamp calculation"""

    def test_calculate_ttl_timestamp_basic(self):
        """Test basic TTL timestamp calculation"""
        # Test with a known future date
        days_from_now = 30

        with patch('scanner.datetime') as mock_datetime:
            # Mock current time as 2024-01-15 10:30:00 UTC
            fixed_time = datetime(2024, 1, 15, 10, 30, 0, tzinfo=timezone.utc)
            mock_datetime.now.return_value = fixed_time
            mock_datetime.replace = MagicMock(return_value=fixed_time.replace(hour=0, minute=0, second=0, microsecond=0))

            result = calculate_ttl_timestamp(days_from_now)

            # Expected: 2024-01-15 00:00:00 + 30 days = 2024-02-14 00:00:00
            expected = datetime(2024, 2, 14, 0, 0, 0, tzinfo=timezone.utc)
            expected_timestamp = int(expected.timestamp())

            assert result == expected_timestamp
            assert isinstance(result, int)
            assert result > 0

    def test_calculate_ttl_timestamp_different_days(self):
        """Test TTL calculation with different day values"""
        test_cases = [7, 30, 90, 365]

        for days in test_cases:
            with patch('scanner.datetime') as mock_datetime:
                fixed_time = datetime(2024, 1, 1, 12, 0, 0, tzinfo=timezone.utc)
                mock_datetime.now.return_value = fixed_time
                mock_datetime.replace = MagicMock(return_value=fixed_time.replace(hour=0, minute=0, second=0, microsecond=0))

                result = calculate_ttl_timestamp(days)

                # Calculate expected result
                expected_date = fixed_time.replace(hour=0, minute=0, second=0, microsecond=0) + timedelta(days=days)
                expected_timestamp = int(expected_date.timestamp())

                assert result == expected_timestamp
                assert result > int(fixed_time.timestamp())  # Should be in the future

    def test_calculate_ttl_timestamp_zero_days(self):
        """Test TTL calculation with zero days (edge case)"""
        with patch('scanner.datetime') as mock_datetime:
            fixed_time = datetime(2024, 1, 1, 12, 0, 0, tzinfo=timezone.utc)
            mock_datetime.now.return_value = fixed_time
            mock_datetime.replace = MagicMock(return_value=fixed_time.replace(hour=0, minute=0, second=0, microsecond=0))

            result = calculate_ttl_timestamp(0)

            # Should be midnight of the same day
            expected = fixed_time.replace(hour=0, minute=0, second=0, microsecond=0)
            expected_timestamp = int(expected.timestamp())

            assert result == expected_timestamp


class TestProcessFinding:
    """Test finding processing"""

    def setup_method(self):
        """Setup for each test method"""
        self.base_finding = {
            'Id': 'test-finding-123',
            'Title': 'Test Security Finding',
            'Description': 'This is a test finding',
            'Severity': {'Label': 'HIGH'},
            'Resources': [{'Type': 'AwsEc2Instance', 'Id': 'i-1234567890abcdef0'}],
            'AwsAccountId': '123456789012',
            'Region': 'us-east-1'
        }

    def test_process_finding_complete(self):
        """Test processing complete finding"""
        result = process_finding(self.base_finding)

        assert result is not None
        assert result['id'] == 'test-finding-123'
        assert result['severity'] == 'HIGH'
        assert result['title'] == 'Test Security Finding'
        assert result['description'] == 'This is a test finding'
        assert result['resource_type'] == 'AwsEc2Instance'
        assert result['resource_id'] == 'i-1234567890abcdef0'
        assert result['account_id'] == '123456789012'
        assert result['region'] == 'us-east-1'

        # Verify generated fields
        assert 'timestamp' in result
        assert 'ttl_timestamp' in result
        assert 'raw_finding' in result

        # Verify timestamp format
        assert isinstance(result['timestamp'], str)
        assert len(result['timestamp']) > 0

        # Verify TTL timestamp is reasonable (future date)
        assert isinstance(result['ttl_timestamp'], int)
        assert result['ttl_timestamp'] > 1600000000  # Some time in 2020

    def test_process_finding_minimal(self):
        """Test processing minimal finding"""
        minimal_finding = {
            'Id': 'minimal-finding',
            'Severity': {'Label': 'MEDIUM'}
        }

        result = process_finding(minimal_finding)

        assert result is not None
        assert result['id'] == 'minimal-finding'
        assert result['severity'] == 'MEDIUM'
        assert result['title'] == ''
        assert result['description'] == ''
        assert result['resource_type'] == ''
        assert result['resource_id'] == ''

    def test_process_finding_missing_resources(self):
        """Test processing finding with missing resources"""
        finding_no_resources = {
            'Id': 'no-resources-finding',
            'Title': 'Finding without resources',
            'Severity': {'Label': 'LOW'}
        }

        result = process_finding(finding_no_resources)

        assert result is not None
        assert result['resource_type'] == ''
        assert result['resource_id'] == ''

    def test_process_finding_multiple_resources(self):
        """Test processing finding with multiple resources"""
        finding_multi_resources = {
            'Id': 'multi-resource-finding',
            'Title': 'Finding with multiple resources',
            'Severity': {'Label': 'HIGH'},
            'Resources': [
                {'Type': 'AwsEc2Instance', 'Id': 'i-123'},
                {'Type': 'AwsS3Bucket', 'Id': 'my-bucket'}
            ]
        }

        result = process_finding(finding_multi_resources)

        # Should use the first resource
        assert result['resource_type'] == 'AwsEc2Instance'
        assert result['resource_id'] == 'i-123'

    def test_process_finding_float_conversion(self):
        """Test float to Decimal conversion"""
        finding_with_floats = {
            'Id': 'float-finding',
            'Severity': {'Label': 'HIGH'},
            'numeric_score': 8.5,
            'confidence': 0.95,
            'count': 42.0
        }

        result = process_finding(finding_with_floats)

        assert isinstance(result['numeric_score'], Decimal)
        assert result['numeric_score'] == Decimal('8.5')

        assert isinstance(result['confidence'], Decimal)
        assert result['confidence'] == Decimal('0.95')

        assert isinstance(result['count'], Decimal)
        assert result['count'] == Decimal('42.0')

    def test_process_finding_different_severities(self):
        """Test processing findings with different severity levels"""
        severities = ['INFORMATIONAL', 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL']

        for severity in severities:
            finding = {
                'Id': f'test-{severity.lower()}',
                'Severity': {'Label': severity}
            }

            result = process_finding(finding)
            assert result['severity'] == severity

    def test_process_finding_error_cases(self):
        """Test processing error handling"""
        error_cases = [
            None,           # None finding
            {},             # Empty finding
            {'Id': None},   # Missing ID
            {'Severity': None},  # Missing severity
            {'Id': '', 'Severity': {}},  # Empty values
        ]

        for invalid_finding in error_cases:
            result = process_finding(invalid_finding)
            assert result is None

    def test_process_finding_raw_finding_storage(self):
        """Test that raw finding is properly stored"""
        result = process_finding(self.base_finding)

        assert 'raw_finding' in result
        raw_finding = result['raw_finding']

        # Should be JSON string
        assert isinstance(raw_finding, str)

        # Should be parseable back to original
        parsed = json.loads(raw_finding)
        assert parsed['Id'] == 'test-finding-123'
        assert parsed['Severity']['Label'] == 'HIGH'


class TestSendAlert:
    """Test alert sending"""

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.sns')
    def test_send_alert_critical(self, mock_sns, mock_get_ssm):
        """Test sending critical alert"""
        mock_get_ssm.return_value = 'arn:aws:sns:us-east-1:123456789012:test-topic'

        send_alert('CRITICAL', 'Test critical finding', 'test-finding-123')

        mock_sns.publish.assert_called_once()
        call_args = mock_sns.publish.call_args
        assert call_args[1]['TopicArn'] == 'arn:aws:sns:us-east-1:123456789012:test-topic'
        assert 'CRITICAL' in call_args[1]['Subject']

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.sns')
    def test_send_alert_high(self, mock_sns, mock_get_ssm):
        """Test sending high severity alert"""
        mock_get_ssm.return_value = 'arn:aws:sns:us-east-1:123456789012:test-topic'

        send_alert('HIGH', 'Test high finding', 'test-finding-456')

        mock_sns.publish.assert_called_once()
        call_args = mock_sns.publish.call_args
        assert 'HIGH' in call_args[1]['Subject']

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.sns')
    def test_send_alert_medium_no_action(self, mock_sns, mock_get_ssm):
        """Test that medium severity doesn't send alert"""
        send_alert('MEDIUM', 'Test medium finding', 'test-finding-789')

        mock_sns.publish.assert_not_called()
        mock_get_ssm.assert_not_called()

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.sns')
    def test_send_alert_error(self, mock_sns, mock_get_ssm):
        """Test alert sending error"""
        mock_get_ssm.side_effect = Exception("SSM error")

        # Should not raise exception, just log error
        send_alert('CRITICAL', 'Test finding', 'test-finding')

        mock_sns.publish.assert_not_called()


class TestLambdaHandler:
    """Test Lambda handler functionality"""

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.dynamodb')
    def test_lambda_handler_direct_security_hub_event(self, mock_dynamodb, mock_get_ssm):
        """Test processing direct Security Hub event"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        event = {
            'source': 'aws.securityhub',
            'detail': {
                'findings': [{
                    'Id': 'test-finding-123',
                    'Title': 'Test Finding',
                    'Severity': {'Label': 'HIGH'}
                }]
            }
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['findings_processed'] == 1
        assert body['findings_stored'] == 1
        mock_table.put_item.assert_called_once()

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.dynamodb')
    def test_lambda_handler_sqs_event(self, mock_dynamodb, mock_get_ssm):
        """Test processing SQS event from EventBridge"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        event = {
            'Records': [{
                'eventSource': 'aws:sqs',
                'body': json.dumps({
                    'detail': {
                        'findings': [{
                            'Id': 'sqs-finding-123',
                            'Severity': {'Label': 'CRITICAL'}
                        }]
                    }
                })
            }]
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['findings_processed'] == 1
        mock_table.put_item.assert_called_once()

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.dynamodb')
    def test_lambda_handler_manual_event(self, mock_dynamodb, mock_get_ssm):
        """Test processing manual/test event"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        event = {
            'findings': [{
                'Id': 'manual-finding-123',
                'Severity': {'Label': 'MEDIUM'}
            }]
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['findings_processed'] == 1
        mock_table.put_item.assert_called_once()

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.dynamodb')
    @patch('scanner.send_alert')
    def test_lambda_handler_critical_finding_alert(self, mock_send_alert, mock_dynamodb, mock_get_ssm):
        """Test that critical findings trigger alerts"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        event = {
            'findings': [{
                'Id': 'critical-finding-123',
                'Title': 'Critical Security Issue',
                'Severity': {'Label': 'CRITICAL'}
            }]
        }

        lambda_handler(event, None)

        mock_send_alert.assert_called_once_with(
            'CRITICAL',
            'Security Finding: Critical Security Issue',
            'critical-finding-123'
        )

    @patch('scanner.get_ssm_parameter')
    @patch('scanner.dynamodb')
    def test_lambda_handler_dynamodb_error(self, mock_dynamodb, mock_get_ssm):
        """Test handling DynamoDB errors"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        from botocore.exceptions import ClientError
        mock_table.put_item.side_effect = ClientError(
            {'Error': {'Code': 'ValidationException'}}, 'PutItem'
        )

        event = {
            'findings': [{
                'Id': 'error-finding-123',
                'Severity': {'Label': 'HIGH'}
            }]
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['findings_processed'] == 1
        assert body['findings_stored'] == 0  # Failed to store

    @patch('scanner.get_ssm_parameter')
    def test_lambda_handler_ssm_error(self, mock_get_ssm):
        """Test handling SSM parameter errors"""
        from botocore.exceptions import ClientError
        mock_get_ssm.side_effect = ClientError(
            {'Error': {'Code': 'ParameterNotFound'}}, 'GetParameter'
        )

        event = {
            'findings': [{
                'Id': 'test-finding',
                'Severity': {'Label': 'HIGH'}
            }]
        }

        with pytest.raises(ClientError):
            lambda_handler(event, None)

    def test_lambda_handler_empty_findings(self):
        """Test handling empty findings list"""
        event = {
            'findings': []
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['findings_processed'] == 0
        assert body['findings_stored'] == 0

    def test_lambda_handler_malformed_finding(self):
        """Test handling malformed findings"""
        event = {
            'findings': [None, {}, {'invalid': 'finding'}]
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['findings_processed'] == 3
        assert body['findings_stored'] == 0  # None of them should be stored due to processing errors


if __name__ == '__main__':
    pytest.main([__file__])