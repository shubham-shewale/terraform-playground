#!/usr/bin/env python3
"""
Unit tests for CSPM Monitor Archiver Lambda function
"""

import json
import gzip
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
zip_path = os.path.join(lambda_src_dir, 'archiver.zip')

if os.path.exists(zip_path):
    archiver_module = load_lambda_function(zip_path, 'archiver')
    lambda_handler = archiver_module.lambda_handler
    get_ssm_parameter = archiver_module.get_ssm_parameter
    get_expired_findings = archiver_module.get_expired_findings
    archive_findings_to_s3 = archiver_module.archive_findings_to_s3
    delete_archived_findings = archiver_module.delete_archived_findings
else:
    # Fallback to direct import for development
    sys.path.insert(0, lambda_src_dir)
    from archiver import (
        lambda_handler,
        get_ssm_parameter,
        get_expired_findings,
        archive_findings_to_s3,
        delete_archived_findings
    )


class TestGetSSMParameter:
    """Test SSM parameter retrieval"""

    @patch('archiver.ssm')
    def test_get_ssm_parameter_success(self, mock_ssm):
        """Test successful SSM parameter retrieval"""
        mock_ssm.get_parameter.return_value = {
            'Parameter': {'Value': 'test-value'}
        }

        result = get_ssm_parameter('/test/param')
        assert result == 'test-value'

    @patch('archiver.ssm')
    def test_get_ssm_parameter_error(self, mock_ssm):
        """Test SSM parameter retrieval error"""
        from botocore.exceptions import ClientError
        mock_ssm.get_parameter.side_effect = ClientError(
            {'Error': {'Code': 'ParameterNotFound'}}, 'GetParameter'
        )

        with pytest.raises(ClientError):
            get_ssm_parameter('/test/param')


class TestGetExpiredFindings:
    """Test expired findings retrieval"""

    def test_get_expired_findings_success(self):
        """Test successful expired findings retrieval"""
        mock_table = MagicMock()

        # Mock scan response with expired findings
        mock_table.scan.return_value = {
            'Items': [
                {'id': 'expired-1', 'ttl_timestamp': 1600000000},  # Expired
                {'id': 'expired-2', 'ttl_timestamp': 1600000001},  # Expired
            ],
            'LastEvaluatedKey': None
        }

        cutoff_timestamp = 1600000002  # Current time

        result = get_expired_findings(mock_table, cutoff_timestamp)

        assert len(result) == 2
        assert result[0]['id'] == 'expired-1'
        assert result[1]['id'] == 'expired-2'
        mock_table.scan.assert_called_once()

    def test_get_expired_findings_pagination(self):
        """Test expired findings retrieval with pagination"""
        mock_table = MagicMock()

        # Mock paginated response
        mock_table.scan.side_effect = [
            {
                'Items': [{'id': 'expired-1', 'ttl_timestamp': 1600000000}],
                'LastEvaluatedKey': 'key1'
            },
            {
                'Items': [{'id': 'expired-2', 'ttl_timestamp': 1600000001}],
                'LastEvaluatedKey': None
            }
        ]

        cutoff_timestamp = 1600000002

        result = get_expired_findings(mock_table, cutoff_timestamp)

        assert len(result) == 2
        assert mock_table.scan.call_count == 2

    def test_get_expired_findings_no_expired(self):
        """Test when no findings are expired"""
        mock_table = MagicMock()

        mock_table.scan.return_value = {
            'Items': [
                {'id': 'active-1', 'ttl_timestamp': 1700000000},  # Future
            ],
            'LastEvaluatedKey': None
        }

        cutoff_timestamp = 1600000000  # Past

        result = get_expired_findings(mock_table, cutoff_timestamp)

        assert len(result) == 1
        assert result[0]['id'] == 'active-1'

    @patch('archiver.logger')
    def test_get_expired_findings_error(self, mock_logger):
        """Test error handling in expired findings retrieval"""
        mock_table = MagicMock()

        from botocore.exceptions import ClientError
        mock_table.scan.side_effect = ClientError(
            {'Error': {'Code': 'ValidationException'}}, 'Scan'
        )

        with pytest.raises(ClientError):
            get_expired_findings(mock_table, 1600000000)


class TestArchiveFindingsToS3:
    """Test S3 archiving functionality"""

    @patch('archiver.s3')
    @patch('archiver.datetime')
    def test_archive_findings_to_s3_success(self, mock_datetime, mock_s3):
        """Test successful S3 archiving"""
        # Mock current time
        mock_now = MagicMock()
        mock_now.now.return_value = datetime(2024, 1, 1, 12, 0, 0, tzinfo=timezone.utc)
        mock_datetime.now = mock_now.now

        findings = [
            {'id': 'test-1', 'severity': 'HIGH', 'score': Decimal('8.5')},
            {'id': 'test-2', 'severity': 'MEDIUM', 'score': Decimal('6.0')}
        ]

        bucket_name = 'test-archive-bucket'

        result = archive_findings_to_s3(findings, bucket_name)

        assert result == 2
        mock_s3.put_object.assert_called_once()

        # Verify the call arguments
        call_args = mock_s3.put_object.call_args
        assert call_args[1]['Bucket'] == bucket_name
        assert call_args[1]['ContentType'] == 'application/json'
        assert call_args[1]['ContentEncoding'] == 'gzip'
        assert call_args[1]['ServerSideEncryption'] == 'AES256'

        # Verify compressed data
        compressed_data = call_args[1]['Body']
        decompressed_data = gzip.decompress(compressed_data).decode('utf-8')
        archive_json = json.loads(decompressed_data)

        assert archive_json['metadata']['total_findings'] == 2
        assert len(archive_json['findings']) == 2
        assert archive_json['findings'][0]['score'] == 8.5  # Decimal converted to float
        assert isinstance(archive_json['findings'][0]['score'], float)

    @patch('archiver.s3')
    def test_archive_findings_to_s3_no_bucket(self, mock_s3):
        """Test archiving without S3 bucket configured"""
        findings = [{'id': 'test-1', 'severity': 'HIGH'}]

        result = archive_findings_to_s3(findings, '')

        assert result == 0
        mock_s3.put_object.assert_not_called()

    @patch('archiver.s3')
    def test_archive_findings_to_s3_error(self, mock_s3):
        """Test S3 archiving error"""
        from botocore.exceptions import ClientError
        mock_s3.put_object.side_effect = ClientError(
            {'Error': {'Code': 'NoSuchBucket'}}, 'PutObject'
        )

        findings = [{'id': 'test-1', 'severity': 'HIGH'}]

        with pytest.raises(ClientError):
            archive_findings_to_s3(findings, 'invalid-bucket')


class TestDeleteArchivedFindings:
    """Test DynamoDB deletion functionality"""

    def test_delete_archived_findings_success(self):
        """Test successful deletion of archived findings"""
        mock_table = MagicMock()

        findings = [
            {'id': 'test-1'},
            {'id': 'test-2'},
            {'id': 'test-3'},
        ]

        result = delete_archived_findings(mock_table, findings)

        assert result == 3
        assert mock_table.batch_writer.call_count == 1

        # Verify batch writer was used correctly
        batch_writer_mock = mock_table.batch_writer.return_value.__enter__.return_value
        assert batch_writer_mock.delete_item.call_count == 3

    def test_delete_archived_findings_multiple_batches(self):
        """Test deletion with multiple batches (25+ items)"""
        mock_table = MagicMock()

        # Create 30 findings to test batching
        findings = [{'id': f'test-{i}'} for i in range(30)]

        result = delete_archived_findings(mock_table, findings)

        assert result == 30
        assert mock_table.batch_writer.call_count == 2  # 25 + 5 = 2 batches

    def test_delete_archived_findings_error(self):
        """Test deletion error handling"""
        mock_table = MagicMock()

        from botocore.exceptions import ClientError
        mock_table.batch_writer.side_effect = ClientError(
            {'Error': {'Code': 'ValidationException'}}, 'BatchWriteItem'
        )

        findings = [{'id': 'test-1'}]

        with pytest.raises(ClientError):
            delete_archived_findings(mock_table, findings)


class TestLambdaHandler:
    """Test Lambda handler functionality"""

    @patch('archiver.get_ssm_parameter')
    @patch('archiver.dynamodb')
    @patch('archiver.get_expired_findings')
    @patch('archiver.archive_findings_to_s3')
    @patch('archiver.delete_archived_findings')
    def test_lambda_handler_successful_archival(self, mock_delete, mock_archive, mock_get_expired,
                                               mock_dynamodb, mock_get_ssm):
        """Test successful archival process"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        expired_findings = [
            {'id': 'expired-1', 'severity': 'HIGH'},
            {'id': 'expired-2', 'severity': 'MEDIUM'}
        ]
        mock_get_expired.return_value = expired_findings
        mock_archive.return_value = 2
        mock_delete.return_value = 2

        event = {'source': 'aws.events'}

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['findings_processed'] == 2
        assert body['findings_archived'] == 2
        assert body['findings_deleted'] == 2

    @patch('archiver.get_ssm_parameter')
    @patch('archiver.dynamodb')
    @patch('archiver.get_expired_findings')
    def test_lambda_handler_no_expired_findings(self, mock_get_expired, mock_dynamodb, mock_get_ssm):
        """Test when no findings are expired"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        mock_get_expired.return_value = []

        event = {'source': 'aws.events'}

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['findings_processed'] == 0
        assert body['findings_archived'] == 0
        assert body['findings_deleted'] == 0

    @patch('archiver.get_ssm_parameter')
    @patch('archiver.dynamodb')
    @patch('archiver.get_expired_findings')
    @patch('archiver.archive_findings_to_s3')
    def test_lambda_handler_archival_failure(self, mock_archive, mock_get_expired,
                                           mock_dynamodb, mock_get_ssm):
        """Test archival failure handling"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        expired_findings = [{'id': 'expired-1'}]
        mock_get_expired.return_value = expired_findings
        mock_archive.return_value = 0  # Archival failed

        event = {'source': 'aws.events'}

        result = lambda_handler(event, None)

        assert result['statusCode'] == 500
        body = json.loads(result['body'])
        assert body['findings_processed'] == 1
        assert body['findings_archived'] == 0
        assert body['findings_deleted'] == 0
        assert 'Failed to archive all findings' in body['error']

    @patch('archiver.get_ssm_parameter')
    @patch('archiver.dynamodb')
    @patch('archiver.get_expired_findings')
    @patch('archiver.archive_findings_to_s3')
    @patch('archiver.delete_archived_findings')
    def test_lambda_handler_partial_deletion_failure(self, mock_delete, mock_archive, mock_get_expired,
                                                    mock_dynamodb, mock_get_ssm):
        """Test partial deletion failure handling"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        expired_findings = [{'id': 'expired-1'}, {'id': 'expired-2'}]
        mock_get_expired.return_value = expired_findings
        mock_archive.return_value = 2  # All archived successfully
        mock_delete.return_value = 1   # Only 1 deleted

        event = {'source': 'aws.events'}

        result = lambda_handler(event, None)

        assert result['statusCode'] == 500
        body = json.loads(result['body'])
        assert body['findings_processed'] == 2
        assert body['findings_archived'] == 2
        assert body['findings_deleted'] == 1
        assert 'Failed to delete all archived findings' in body['error']

    @patch('archiver.get_ssm_parameter')
    def test_lambda_handler_ssm_error(self, mock_get_ssm):
        """Test SSM parameter error handling"""
        from botocore.exceptions import ClientError
        mock_get_ssm.side_effect = ClientError(
            {'Error': {'Code': 'ParameterNotFound'}}, 'GetParameter'
        )

        event = {'source': 'aws.events'}

        result = lambda_handler(event, None)

        assert result['statusCode'] == 500
        body = json.loads(result['body'])
        assert 'Archival failed' in body['message']

    @patch('archiver.get_ssm_parameter')
    @patch('archiver.dynamodb')
    @patch('archiver.get_expired_findings')
    def test_lambda_handler_scan_error(self, mock_get_expired, mock_dynamodb, mock_get_ssm):
        """Test DynamoDB scan error handling"""
        mock_get_ssm.return_value = 'test-table'
        mock_table = MagicMock()
        mock_dynamodb.Table.return_value = mock_table

        from botocore.exceptions import ClientError
        mock_get_expired.side_effect = ClientError(
            {'Error': {'Code': 'ValidationException'}}, 'Scan'
        )

        event = {'source': 'aws.events'}

        result = lambda_handler(event, None)

        assert result['statusCode'] == 500
        body = json.loads(result['body'])
        assert 'Archival failed' in body['message']


if __name__ == '__main__':
    pytest.main([__file__])