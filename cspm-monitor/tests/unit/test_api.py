#!/usr/bin/env python3
"""
Unit tests for CSPM Monitor API Lambda function
"""

import json
import time
import pytest
from unittest.mock import patch, MagicMock
from decimal import Decimal
from datetime import datetime, timezone

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
zip_path = os.path.join(lambda_src_dir, 'api.zip')

if os.path.exists(zip_path):
    api_module = load_lambda_function(zip_path, 'api')
    lambda_handler = api_module.lambda_handler
    get_ssm_parameter = api_module.get_ssm_parameter
    get_table = api_module.get_table
    query_findings_by_severity = api_module.query_findings_by_severity
    get_finding_by_id = api_module.get_finding_by_id
    get_findings_summary = api_module.get_findings_summary
    create_response = api_module.create_response
else:
    # Fallback to direct import for development
    sys.path.insert(0, lambda_src_dir)
    from api import (
        lambda_handler,
        get_ssm_parameter,
        get_table,
        query_findings_by_severity,
        get_finding_by_id,
        get_findings_summary,
        create_response
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

            # Reload the module to pick up the mock
            import importlib
            if 'api' in sys.modules:
                importlib.reload(sys.modules['api'])

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

    def test_get_ssm_parameter_with_encryption(self):
        """Test SSM parameter retrieval with encryption"""
        with patch('boto3.client') as mock_boto3_client:
            mock_boto3_client.return_value = self.mock_ssm
            self.mock_ssm.get_parameter.return_value = {
                'Parameter': {'Value': 'encrypted-value'}
            }

            result = get_ssm_parameter('/secure/param')
            assert result == 'encrypted-value'

            # Verify WithDecryption was used
            call_args = self.mock_ssm.get_parameter.call_args
            assert call_args[1]['WithDecryption'] is True


class TestGetTable:
    """Test DynamoDB table retrieval"""

    def setup_method(self):
        """Setup for each test method"""
        self.mock_dynamodb = MagicMock()
        self.mock_table = MagicMock()

    def test_get_table_success(self):
        """Test successful table retrieval"""
        with patch('boto3.resource') as mock_boto3_resource, \
             patch('api.get_ssm_parameter') as mock_get_ssm:

            mock_boto3_resource.return_value = self.mock_dynamodb
            mock_get_ssm.return_value = 'test-table'
            self.mock_dynamodb.Table.return_value = self.mock_table

            result = get_table()
            assert result == self.mock_table
            self.mock_dynamodb.Table.assert_called_once_with('test-table')

    def test_get_table_with_different_name(self):
        """Test table retrieval with different table name"""
        with patch('boto3.resource') as mock_boto3_resource, \
             patch('api.get_ssm_parameter') as mock_get_ssm:

            mock_boto3_resource.return_value = self.mock_dynamodb
            mock_get_ssm.return_value = 'production-findings'
            self.mock_dynamodb.Table.return_value = self.mock_table

            result = get_table()
            assert result == self.mock_table
            self.mock_dynamodb.Table.assert_called_once_with('production-findings')


class TestQueryFindingsBySeverity:
    """Test findings query functionality"""

    def setup_method(self):
        """Setup for each test method"""
        self.mock_table = MagicMock()

    def test_query_findings_by_severity_all(self):
        """Test querying all findings"""
        with patch('api.get_table') as mock_get_table, \
             patch('boto3.dynamodb.conditions') as mock_conditions:

            mock_get_table.return_value = self.mock_table
            mock_conditions.Attr.return_value.exists.return_value = True

            self.mock_table.scan.return_value = {
                'Items': [
                    {'id': '1', 'severity': 'HIGH', 'title': 'Test Finding 1'},
                    {'id': '2', 'severity': 'MEDIUM', 'title': 'Test Finding 2'}
                ]
            }

            result = query_findings_by_severity(None, 100)

            assert len(result) == 2
            assert result[0]['id'] == '1'
            assert result[0]['severity'] == 'HIGH'
            assert result[1]['id'] == '2'
            assert result[1]['severity'] == 'MEDIUM'
            self.mock_table.scan.assert_called_once()

    def test_query_findings_by_severity_specific(self):
        """Test querying findings by specific severity"""
        with patch('api.get_table') as mock_get_table, \
             patch('boto3.dynamodb.conditions') as mock_conditions:

            mock_get_table.return_value = self.mock_table
            mock_key_condition = MagicMock()
            mock_conditions.Key.return_value.eq.return_value = mock_key_condition

            self.mock_table.query.return_value = {
                'Items': [
                    {'id': '1', 'severity': 'HIGH', 'title': 'High Severity Finding'}
                ]
            }

            result = query_findings_by_severity('HIGH', 50)

            assert len(result) == 1
            assert result[0]['severity'] == 'HIGH'
            assert result[0]['title'] == 'High Severity Finding'
            self.mock_table.query.assert_called_once()

    def test_query_findings_by_severity_empty_result(self):
        """Test querying with no results"""
        with patch('api.get_table') as mock_get_table, \
             patch('boto3.dynamodb.conditions') as mock_conditions:

            mock_get_table.return_value = self.mock_table
            mock_conditions.Attr.return_value.exists.return_value = True

            self.mock_table.scan.return_value = {'Items': []}

            result = query_findings_by_severity(None, 100)

            assert len(result) == 0
            assert isinstance(result, list)

    def test_query_findings_decimal_conversion(self):
        """Test decimal to float conversion"""
        with patch('api.get_table') as mock_get_table, \
             patch('boto3.dynamodb.conditions') as mock_conditions:

            mock_get_table.return_value = self.mock_table
            mock_conditions.Attr.return_value.exists.return_value = True

            # Test with multiple decimal values
            test_item = {
                'id': 'test-123',
                'severity': 'HIGH',
                'score': Decimal('8.5'),
                'confidence': Decimal('0.95'),
                'count': Decimal('42')
            }

            self.mock_table.scan.return_value = {'Items': [test_item]}

            result = query_findings_by_severity(None, 100)

            assert len(result) == 1
            item = result[0]

            # Verify all Decimal values are converted to float
            assert item['score'] == 8.5
            assert isinstance(item['score'], float)

            assert item['confidence'] == 0.95
            assert isinstance(item['confidence'], float)

            assert item['count'] == 42.0  # Integer decimals become float
            assert isinstance(item['count'], float)

            # Verify non-decimal fields are unchanged
            assert item['id'] == 'test-123'
            assert item['severity'] == 'HIGH'


class TestGetFindingById:
    """Test getting finding by ID"""

    @patch('api.get_table')
    def test_get_finding_by_id_found(self, mock_get_table):
        """Test finding exists"""
        mock_table = MagicMock()
        mock_get_table.return_value = mock_table

        mock_table.get_item.return_value = {
            'Item': {'id': 'test-123', 'severity': 'HIGH', 'title': 'Test Finding'}
        }

        result = get_finding_by_id('test-123')

        assert result['id'] == 'test-123'
        assert result['severity'] == 'HIGH'

    @patch('api.get_table')
    def test_get_finding_by_id_not_found(self, mock_get_table):
        """Test finding does not exist"""
        mock_table = MagicMock()
        mock_get_table.return_value = mock_table

        mock_table.get_item.return_value = {}

        result = get_finding_by_id('test-123')

        assert result is None


class TestGetFindingsSummary:
    """Test findings summary functionality"""

    @patch('api.get_table')
    def test_get_findings_summary_success(self, mock_get_table):
        """Test successful summary generation"""
        mock_table = MagicMock()
        mock_get_table.return_value = mock_table

        # Mock GSI queries
        mock_table.query.side_effect = [
            {'Count': 5},  # CRITICAL
            {'Count': 10}, # HIGH
            {'Count': 20}, # MEDIUM
            {'Count': 15}, # LOW
            {'Count': 8}   # INFORMATIONAL
        ]

        result = get_findings_summary()

        assert result['total_findings'] == 58
        assert result['severity_breakdown']['CRITICAL'] == 5
        assert result['severity_breakdown']['HIGH'] == 10
        assert 'last_updated' in result


class TestCreateResponse:
    """Test response creation"""

    def test_create_response_success(self):
        """Test successful response creation"""
        result = create_response(200, {'message': 'success'})

        assert result['statusCode'] == 200
        assert result['headers']['Content-Type'] == 'application/json'
        assert json.loads(result['body']) == {'message': 'success'}

    def test_create_response_cors(self):
        """Test CORS headers"""
        result = create_response(200, {'message': 'success'}, cors=True)

        assert 'Access-Control-Allow-Origin' in result['headers']
        assert result['headers']['Access-Control-Allow-Origin'] == '*'

    def test_create_response_no_cors(self):
        """Test without CORS headers"""
        result = create_response(200, {'message': 'success'}, cors=False)

        assert 'Access-Control-Allow-Origin' not in result['headers']


class TestLambdaHandler:
    """Test Lambda handler functionality"""

    def test_lambda_handler_cold_start_simulation(self):
        """Test Lambda cold start behavior"""
        # Simulate cold start by testing first invocation
        event = {
            'httpMethod': 'GET',
            'path': '/health'
        }

        # Mock context for cold start
        context = MagicMock()
        context.get_remaining_time_in_millis.return_value = 30000
        context.memory_limit_in_mb = 256
        context.aws_request_id = 'test-request-id'

        result = lambda_handler(event, context)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['status'] == 'healthy'

    def test_lambda_handler_memory_pressure(self):
        """Test Lambda behavior under memory pressure"""
        # Create large dataset to simulate memory usage
        large_findings = [{'id': f'test-{i}', 'severity': 'HIGH'} for i in range(1000)]

        with patch('api.query_findings_by_severity') as mock_query:
            mock_query.return_value = large_findings

            event = {
                'httpMethod': 'GET',
                'path': '/findings',
                'queryStringParameters': {'limit': '1000'}
            }

            context = MagicMock()
            context.get_remaining_time_in_millis.return_value = 25000  # Low time remaining

            result = lambda_handler(event, context)

            assert result['statusCode'] == 200
            body = json.loads(result['body'])
            assert len(body['data']) == 1000

    def test_lambda_handler_timeout_simulation(self):
        """Test Lambda timeout handling"""
        with patch('api.query_findings_by_severity') as mock_query:
            # Mock a function that would normally take too long
            mock_query.return_value = []

            event = {
                'httpMethod': 'GET',
                'path': '/findings'
            }

            context = MagicMock()
            context.get_remaining_time_in_millis.return_value = 500  # Very low time remaining

            # This should complete quickly despite low timeout
            import time
            start_time = time.time()
            result = lambda_handler(event, context)
            end_time = time.time()

            # Should complete in less than 100ms even with low timeout
            assert (end_time - start_time) < 0.1
            assert result['statusCode'] == 200

    def test_lambda_handler_environment_variables(self):
        """Test Lambda environment variable handling"""
        # Test with different environment configurations
        with patch.dict(os.environ, {
            'DYNAMODB_TABLE_PARAM': '/test/custom-table',
            'CUSTOM_ENV_VAR': 'test-value'
        }):
            event = {
                'httpMethod': 'GET',
                'path': '/health'
            }

            result = lambda_handler(event, None)
            assert result['statusCode'] == 200

    def test_lambda_handler_vpc_configuration(self):
        """Test Lambda VPC configuration handling"""
        # This would test VPC-specific behavior in a real deployment
        event = {
            'httpMethod': 'GET',
            'path': '/health'
        }

        context = MagicMock()
        context.client_context = None  # No client context in VPC

        result = lambda_handler(event, context)
        assert result['statusCode'] == 200

    def test_lambda_handler_health_check(self):
        """Test health check endpoint"""
        event = {
            'httpMethod': 'GET',
            'path': '/health'
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['status'] == 'healthy'
        assert body['service'] == 'cspm-monitor-api'

    def test_lambda_handler_options(self):
        """Test CORS preflight"""
        event = {
            'httpMethod': 'OPTIONS',
            'path': '/findings'
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        assert 'Access-Control-Allow-Origin' in result['headers']

    @patch('api.query_findings_by_severity')
    def test_lambda_handler_get_findings(self, mock_query):
        """Test getting findings list"""
        mock_query.return_value = [
            {'id': '1', 'severity': 'HIGH'},
            {'id': '2', 'severity': 'MEDIUM'}
        ]

        event = {
            'httpMethod': 'GET',
            'path': '/findings',
            'queryStringParameters': {'limit': '10'}
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['success'] is True
        assert len(body['data']) == 2
        assert body['count'] == 2

    @patch('api.get_finding_by_id')
    def test_lambda_handler_get_finding_by_id(self, mock_get_finding):
        """Test getting specific finding"""
        mock_get_finding.return_value = {
            'id': 'test-123',
            'severity': 'HIGH',
            'title': 'Test Finding'
        }

        event = {
            'httpMethod': 'GET',
            'path': '/findings',
            'queryStringParameters': {'id': 'test-123'}
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['success'] is True
        assert body['data']['id'] == 'test-123'

    def test_lambda_handler_invalid_severity(self):
        """Test invalid severity parameter"""
        event = {
            'httpMethod': 'GET',
            'path': '/findings',
            'queryStringParameters': {'severity': 'INVALID'}
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 400
        body = json.loads(result['body'])
        assert body['success'] is False
        assert 'Invalid severity' in body['error']

    def test_lambda_handler_invalid_limit(self):
        """Test invalid limit parameter"""
        event = {
            'httpMethod': 'GET',
            'path': '/findings',
            'queryStringParameters': {'limit': 'invalid'}
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 400
        body = json.loads(result['body'])
        assert body['success'] is False
        assert 'Invalid limit' in body['error']

    def test_lambda_handler_method_not_allowed(self):
        """Test unsupported HTTP method"""
        event = {
            'httpMethod': 'POST',
            'path': '/findings'
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 405
        body = json.loads(result['body'])
        assert body['success'] is False
        assert body['error'] == 'Method not allowed'

    @patch('api.get_findings_summary')
    def test_lambda_handler_get_summary(self, mock_summary):
        """Test getting findings summary"""
        mock_summary.return_value = {
            'total_findings': 100,
            'severity_breakdown': {'HIGH': 10, 'MEDIUM': 20},
            'last_updated': '2024-01-01T00:00:00'
        }

        event = {
            'httpMethod': 'GET',
            'path': '/summary'
        }

        result = lambda_handler(event, None)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['success'] is True
        assert body['data']['total_findings'] == 100

    def test_lambda_handler_malformed_json(self):
        """Test handling of malformed JSON in request"""
        # This tests how the handler deals with invalid input
        event = {
            'httpMethod': 'GET',
            'path': '/findings',
            'body': 'invalid json {'
        }

        result = lambda_handler(event, None)
        # Should handle gracefully without crashing
        assert isinstance(result, dict)
        assert 'statusCode' in result

    def test_lambda_handler_large_payload(self):
        """Test handling of large request payloads"""
        # Create a very large query parameter
        large_param = 'x' * 10000  # 10KB parameter

        event = {
            'httpMethod': 'GET',
            'path': '/findings',
            'queryStringParameters': {'id': large_param}
        }

        result = lambda_handler(event, None)
        # Should handle large inputs appropriately
        assert result['statusCode'] in [200, 400]  # Either success or validation error

    def test_lambda_handler_special_characters(self):
        """Test handling of special characters in input"""
        special_chars = "!@#$%^&*()_+-=[]{}|;:,.<>?"

        event = {
            'httpMethod': 'GET',
            'path': '/findings',
            'queryStringParameters': {'severity': special_chars}
        }

        result = lambda_handler(event, None)
        # Should validate input properly
        assert result['statusCode'] == 400
        body = json.loads(result['body'])
        assert 'Invalid severity' in body['error']

    def test_lambda_handler_sql_injection_attempt(self):
        """Test protection against SQL injection attempts"""
        injection_attempts = [
            "'; DROP TABLE findings; --",
            "' OR '1'='1",
            "<script>alert('xss')</script>",
            "../../../etc/passwd"
        ]

        for injection in injection_attempts:
            event = {
                'httpMethod': 'GET',
                'path': '/findings',
                'queryStringParameters': {'id': injection}
            }

            result = lambda_handler(event, None)
            # Should either reject or sanitize the input
            assert result['statusCode'] in [200, 400, 404]

    def test_lambda_handler_rate_limiting_simulation(self):
        """Test behavior under simulated rate limiting"""
        # Simulate multiple rapid requests
        event = {
            'httpMethod': 'GET',
            'path': '/findings'
        }

        results = []
        for i in range(10):
            result = lambda_handler(event, None)
            results.append(result['statusCode'])

        # All requests should succeed (rate limiting would be handled by API Gateway)
        assert all(status == 200 for status in results)

    def test_lambda_handler_concurrent_access(self):
        """Test concurrent access to shared resources"""
        import threading
        import queue

        event = {
            'httpMethod': 'GET',
            'path': '/health'
        }

        results_queue = queue.Queue()
        errors_queue = queue.Queue()

        def make_request():
            try:
                result = lambda_handler(event, None)
                results_queue.put(result['statusCode'])
            except Exception as e:
                errors_queue.put(str(e))

        # Create multiple threads
        threads = []
        num_threads = 3  # Reduced for better reliability

        for i in range(num_threads):
            thread = threading.Thread(target=make_request)
            threads.append(thread)
            thread.start()

        # Wait for all threads with timeout
        for thread in threads:
            thread.join(timeout=5.0)  # 5 second timeout
            if thread.is_alive():
                # Thread didn't complete in time
                errors_queue.put("Thread timeout")

        # Collect results
        results = []
        while not results_queue.empty():
            results.append(results_queue.get())

        errors = []
        while not errors_queue.empty():
            errors.append(errors_queue.get())

        # All requests should succeed
        assert len(results) == num_threads, f"Expected {num_threads} results, got {len(results)}"
        assert all(status == 200 for status in results), f"Not all requests succeeded: {results}"
        assert len(errors) == 0, f"Errors occurred: {errors}"

    def test_lambda_handler_request_context(self):
        """Test Lambda request context handling"""
        event = {
            'httpMethod': 'GET',
            'path': '/health',
            'requestContext': {
                'requestId': 'test-request-123',
                'apiId': 'test-api',
                'stage': 'prod'
            }
        }

        context = MagicMock()
        context.aws_request_id = 'lambda-request-456'
        context.memory_limit_in_mb = 256
        context.get_remaining_time_in_millis.return_value = 30000

        result = lambda_handler(event, context)

        assert result['statusCode'] == 200
        body = json.loads(result['body'])
        assert body['status'] == 'healthy'
        # Verify context information is available
        assert context.aws_request_id == 'lambda-request-456'

    def test_lambda_handler_network_timeout_simulation(self):
        """Test handling of network timeouts"""
        with patch('api.get_table') as mock_get_table:
            mock_table = MagicMock()
            mock_get_table.return_value = mock_table

            # Simulate network timeout
            from botocore.exceptions import ClientError
            mock_table.scan.side_effect = ClientError(
                {'Error': {'Code': 'TimeoutException'}}, 'Scan'
            )

            event = {
                'httpMethod': 'GET',
                'path': '/findings'
            }

            result = lambda_handler(event, None)
            # Should handle timeout gracefully
            assert result['statusCode'] == 500
            body = json.loads(result['body'])
            assert 'Internal server error' in body['error']

    def test_lambda_handler_disk_space_simulation(self):
        """Test behavior when disk space is low"""
        # This would be relevant for Lambda functions with /tmp usage
        event = {
            'httpMethod': 'GET',
            'path': '/health'
        }

        # Mock disk space check instead of actually creating large files
        import shutil

        with patch('shutil.disk_usage') as mock_disk_usage:
            # Simulate very low disk space (100MB free)
            mock_disk_usage.return_value = (100 * 1024 * 1024, 50 * 1024 * 1024, 100 * 1024 * 1024)

            result = lambda_handler(event, None)
            # Should still function despite low disk space
            assert result['statusCode'] == 200
            body = json.loads(result['body'])
            assert body['status'] == 'healthy'

    def test_lambda_handler_high_memory_usage(self):
        """Test Lambda behavior with high memory usage"""
        # Create a large in-memory dataset
        large_dataset = [{'id': f'test-{i}', 'data': 'x' * 1000} for i in range(1000)]

        with patch('api.query_findings_by_severity') as mock_query:
            mock_query.return_value = large_dataset

            event = {
                'httpMethod': 'GET',
                'path': '/findings',
                'queryStringParameters': {'limit': '1000'}
            }

            context = MagicMock()
            context.memory_limit_in_mb = 256

            result = lambda_handler(event, context)

            # Should handle large datasets properly
            assert result['statusCode'] == 200
            body = json.loads(result['body'])
            assert len(body['data']) == 1000
            assert body['count'] == 1000

    def test_lambda_handler_unicode_handling(self):
        """Test handling of Unicode characters"""
        unicode_strings = [
            "测试字符串",  # Chinese
            "🚀🔥💯",     # Emojis
            "café",       # Accented characters
            "русский",    # Cyrillic
        ]

        for unicode_str in unicode_strings:
            event = {
                'httpMethod': 'GET',
                'path': '/findings',
                'queryStringParameters': {'id': unicode_str}
            }

            result = lambda_handler(event, None)
            # Should handle Unicode properly
            assert result['statusCode'] in [200, 400, 404]
            # Response should be valid JSON
            body = json.loads(result['body'])
            assert isinstance(body, dict)


if __name__ == '__main__':
    pytest.main([__file__])