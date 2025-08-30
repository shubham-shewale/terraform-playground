# Basic VPC Test Suite

A comprehensive, production-grade test suite for the Basic VPC Terraform project that validates infrastructure security, connectivity, compliance, and functionality.

## 🏗️ Test Architecture

The test suite is organized into multiple layers:

```
tests/
├── unit/                 # Unit tests for individual resources
├── integration/          # Integration tests for resource interactions
├── e2e/                  # End-to-end connectivity tests
├── compliance/           # Security compliance tests
├── fixtures/             # Test data and mock responses
└── scripts/              # Test utilities and helpers
```

## 🧪 Test Types

### Unit Tests (`unit/`)
- **Framework**: Terratest (Go)
- **Coverage**: Individual Terraform resources
- **Execution**: Fast, isolated, no AWS resources created
- **Files**:
  - `vpc_test.go` - VPC, subnets, route tables
  - `ec2_test.go` - EC2 instances, EBS encryption
  - `security_test.go` - Security groups, NACLs
  - `monitoring_test.go` - CloudWatch alarms, dashboards
  - `cloudtrail_test.go` - CloudTrail, S3 buckets
  - `ssm_test.go` - SSM roles, VPC endpoints

### Integration Tests (`integration/`)
- **Framework**: Terratest (Go)
- **Coverage**: Resource interactions and dependencies
- **Execution**: Creates AWS resources, moderate duration
- **Files**:
  - `network_connectivity_test.go` - Network configuration validation
  - `security_integration_test.go` - Security group and IAM integration

### End-to-End Tests (`e2e/`)
- **Framework**: Shell scripts
- **Coverage**: Full infrastructure deployment and connectivity
- **Execution**: Creates complete environment, long duration
- **Files**:
  - `connectivity_test.sh` - Full E2E connectivity validation

### Compliance Tests (`compliance/`)
- **Framework**: InSpec
- **Coverage**: Security best practices and compliance
- **Execution**: Validates deployed resources against policies
- **Files**:
  - `vpc_compliance_test.rb` - Network and VPC compliance
  - `security_compliance_test.rb` - Security controls compliance

## 🚀 Quick Start

### Prerequisites

```bash
# Install Go 1.21+
go version

# Install Terraform 1.5+
terraform version

# Install AWS CLI
aws --version

# Configure AWS credentials
aws configure
```

### Running Tests

#### Unit Tests
```bash
cd tests
go test -v ./unit/... -timeout 30m
```

#### Integration Tests
```bash
cd tests
go test -v ./integration/... -timeout 45m
```

#### End-to-End Tests
```bash
cd tests/e2e
chmod +x connectivity_test.sh
./connectivity_test.sh
```

#### Compliance Tests
```bash
# Install InSpec
gem install inspec

# Run compliance tests
cd tests/compliance
inspec exec . --reporter cli
```

## 🔧 Configuration

### Test Variables

Tests use the following variable configuration:

```hcl
environment = "test"
allowed_http_cidrs = ["203.0.113.0/24"]
allowed_ssh_cidrs = ["203.0.113.0/24"]
vpc_cidr = "10.0.0.0/16"
public_subnet_cidr = "10.0.1.0/24"
private_subnet_cidr = "10.0.2.0/24"
```

### Environment Variables

```bash
# AWS Configuration
export AWS_REGION=us-east-1
export AWS_PROFILE=your-profile

# Test Configuration
export TEST_ENVIRONMENT=test
export TEST_TIMEOUT=45m
```

## 📊 Test Coverage

### Infrastructure Components Tested

| Component | Unit Tests | Integration Tests | E2E Tests | Compliance Tests |
|-----------|------------|-------------------|-----------|------------------|
| VPC | ✅ | ✅ | ✅ | ✅ |
| Subnets | ✅ | ✅ | ✅ | ✅ |
| EC2 Instances | ✅ | ✅ | ✅ | ✅ |
| Security Groups | ✅ | ✅ | ✅ | ✅ |
| NACLs | ✅ | ✅ | ✅ | ✅ |
| Route Tables | ✅ | ✅ | ✅ | ✅ |
| Internet Gateway | ✅ | ✅ | ✅ | ✅ |
| NAT Gateway | ✅ | ✅ | ✅ | ✅ |
| CloudWatch | ✅ | ✅ | ✅ | ✅ |
| CloudTrail | ✅ | ✅ | ✅ | ✅ |
| SSM | ✅ | ✅ | ✅ | ✅ |
| VPC Endpoints | ✅ | ✅ | ✅ | ✅ |
| S3 Buckets | ✅ | ✅ | ✅ | ✅ |
| IAM Roles | ✅ | ✅ | ✅ | ✅ |

### Security Controls Tested

- ✅ Encryption at rest (EBS, S3)
- ✅ Network security (SG + NACL)
- ✅ Access control (IAM, least privilege)
- ✅ Monitoring and alerting
- ✅ Audit logging (CloudTrail)
- ✅ Secure communication (VPC endpoints)

## 🔒 Security Testing

### Automated Security Scans

The test suite includes automated security scanning:

```bash
# TFLint for Terraform best practices
tflint --config .tflint.hcl

# TFSec for security vulnerabilities
tfsec .

# InSpec for compliance validation
inspec exec compliance/
```

### Security Test Categories

1. **Infrastructure Security**
   - Encryption validation
   - Network segmentation
   - Access control verification

2. **Compliance Validation**
   - CIS Benchmarks
   - AWS Well-Architected Framework
   - Security best practices

3. **Vulnerability Assessment**
   - Configuration drift detection
   - Security group analysis
   - IAM permission validation

## 📈 Performance Testing

### Test Execution Times

| Test Type | Duration | Frequency |
|-----------|----------|-----------|
| Unit Tests | 5-15 minutes | Every commit |
| Integration Tests | 20-45 minutes | PR validation |
| E2E Tests | 30-60 minutes | Release validation |
| Compliance Tests | 10-20 minutes | Daily |

### Resource Cleanup

All tests include automatic resource cleanup:

```go
defer terraform.Destroy(t, terraformOptions)
```

## 🔄 CI/CD Integration

### GitHub Actions Workflow

The test suite integrates with GitHub Actions:

```yaml
# .github/workflows/ci.yml
name: Basic VPC CI/CD Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  validate:
    # Terraform validation
  security-scan:
    # TFLint, TFSec
  unit-tests:
    # Fast unit tests
  integration-tests:
    # Resource integration tests
  compliance-tests:
    # Security compliance
  e2e-tests:
    # Full environment tests
  deploy:
    # Deployment to staging
```

### Test Environments

- **Unit Tests**: Local/mock environment
- **Integration Tests**: Isolated AWS account
- **E2E Tests**: Staging environment
- **Production**: Manual promotion

## 📋 Test Results and Reporting

### Test Reports

Tests generate comprehensive reports:

```bash
# JUnit XML reports
go test -v ./... -coverprofile=coverage.out

# HTML coverage reports
go tool cover -html=coverage.out -o coverage.html

# JSON test results
go test -v ./... -json > test-results.json
```

### Metrics Collected

- Test execution time
- Test pass/fail rates
- Code coverage percentage
- Security scan results
- Performance benchmarks

## 🛠️ Development

### Adding New Tests

1. **Unit Tests**:
   ```go
   func TestNewResource(t *testing.T) {
       t.Parallel()
       // Test implementation
   }
   ```

2. **Integration Tests**:
   ```go
   func TestResourceIntegration(t *testing.T) {
       t.Parallel()
       // Integration test implementation
   }
   ```

3. **Compliance Tests**:
   ```ruby
   control 'new-control-1.0' do
     impact 1.0
     title 'New Control Title'
     # Test implementation
   end
   ```

### Test Data Management

Test fixtures are stored in `fixtures/`:

- `test_variables.tfvars` - Terraform variables
- `mock_aws_responses.json` - Mock AWS API responses
- `test_data/` - Additional test data files

## 🚨 Troubleshooting

### Common Issues

1. **AWS Credentials**
   ```bash
   aws configure
   aws sts get-caller-identity
   ```

2. **Go Module Issues**
   ```bash
   cd tests
   go mod tidy
   go mod download
   ```

3. **Terraform State**
   ```bash
   cd ../../
   rm -rf .terraform/
   terraform init
   ```

4. **Test Timeouts**
   ```bash
   export TEST_TIMEOUT=60m
   go test -timeout $TEST_TIMEOUT
   ```

### Debug Mode

Enable debug logging:

```bash
export TF_LOG=DEBUG
export TERRATEST_LOG=debug
go test -v ./unit/vpc_test.go
```

## 📚 Resources

### Documentation
- [Terratest Documentation](https://terratest.gruntwork.io/)
- [InSpec Documentation](https://docs.chef.io/inspec/)
- [Terraform Testing Best Practices](https://www.terraform.io/docs/language/tests/index.html)

### Related Projects
- [AWS Testing Library](https://github.com/aws/aws-testing-library)
- [Terraform Compliance](https://terraform-compliance.com/)
- [Checkov](https://checkov.io/)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

### Code Standards

- Use descriptive test names
- Include test documentation
- Follow Go naming conventions
- Add appropriate timeouts
- Include cleanup in defer statements

## 📄 License

This test suite is licensed under the MIT License. See LICENSE file for details.

## 🆘 Support

For support and questions:

- Create an issue in the repository
- Check existing documentation
- Review test logs for error details
- Contact the infrastructure team

---

**Note**: This test suite is designed for production use and includes comprehensive validation of security, compliance, and functionality requirements.