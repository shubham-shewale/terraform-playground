# Static Website Production Test Suite

A comprehensive, production-grade test suite for the Static Website Terraform project that validates CloudFront, WAF, S3, and security configurations.

## 🏗️ Test Architecture

The test suite is organized into multiple layers:

```
tests/
├── unit/                 # Unit tests for individual components
├── integration/          # Integration tests for service interactions
├── e2e/                  # End-to-end website functionality tests
├── compliance/           # Security compliance and regulatory tests
├── chaos/                # Chaos engineering for resilience testing
├── performance/          # CDN performance and load testing
├── cost/                 # Cost optimization validation
├── security/             # Security vulnerability scanning
└── fixtures/             # Test data and mock configurations
```

## 🧪 Test Categories

### Chaos Engineering Tests (`chaos/`)
**Purpose**: Validate system resilience and recovery capabilities for CDN infrastructure
**Framework**: Terratest with AWS SDK
**Coverage**:
- CloudFront distribution failure simulation
- S3 origin access disruption testing
- WAF protection failure scenarios
- Certificate validation failure testing
- Origin Shield regional failure simulation

**Key Tests**:
- `TestChaosCloudFrontFailure` - Tests distribution disable/enable scenarios
- `TestChaosS3OriginFailure` - Validates S3 origin access disruptions
- `TestChaosWAFFailure` - Tests WAF rule misconfiguration scenarios
- `TestChaosCertificateFailure` - Simulates SSL certificate issues
- `TestChaosOriginShieldFailure` - Tests Origin Shield regional failures

### Performance & Load Tests (`performance/`)
**Purpose**: Validate CDN performance under various loads and conditions
**Framework**: Terratest with concurrent HTTP testing
**Coverage**:
- CloudFront response time validation
- Cache performance and hit ratios
- Global CDN latency testing
- Compression effectiveness validation
- Security headers performance impact

**Key Tests**:
- `TestCDNPerformanceBaseline` - Establishes performance baselines
- `TestCDNLoadHandling` - Tests concurrent request handling
- `TestCDNCachePerformance` - Validates cache effectiveness
- `TestCDNGlobalPerformance` - Tests global distribution performance
- `TestCDNCompressionPerformance` - Validates compression benefits

### Cost Optimization Tests (`cost/`)
**Purpose**: Ensure cost-effective CloudFront and WAF resource utilization
**Framework**: Terratest with CloudWatch cost monitoring
**Coverage**:
- CloudFront price class optimization
- WAF rule cost efficiency
- S3 storage and request cost monitoring
- Data transfer cost analysis
- Certificate and monitoring cost validation

**Key Tests**:
- `TestCloudFrontCostOptimization` - Validates price class and Origin Shield usage
- `TestWAFCostOptimization` - Monitors WAF request volume and rule efficiency
- `TestS3CostOptimization` - Checks storage lifecycle and encryption costs
- `TestCertificateCostOptimization` - Validates ACM certificate cost efficiency
- `TestDataTransferCostOptimization` - Monitors CloudFront data transfer costs

### Security Vulnerability Scanning (`security/`)
**Purpose**: Comprehensive security assessment of the static website infrastructure
**Framework**: Terratest with AWS security services
**Coverage**:
- S3 bucket security validation
- CloudFront security configuration
- WAF protection effectiveness
- SSL/TLS configuration validation
- Content security and headers verification

**Key Tests**:
- `TestWebsiteVulnerabilityScan` - Comprehensive vulnerability assessment
- `TestCloudFrontSecurityScan` - CloudFront security configuration validation
- `TestWAFSecurityScan` - WAF rule and protection effectiveness
- `TestS3SecurityScan` - S3 bucket security and access control
- `TestCertificateSecurityScan` - SSL/TLS certificate validation

## 🚀 Quick Start

### Prerequisites
```bash
# Install Go 1.21+
go version

# Install Terraform 1.5+
terraform version

# Configure AWS credentials
aws configure
```

### Running Tests

#### Unit Tests
```bash
cd tests
go test ./unit/... -v -timeout 15m
```

#### Integration Tests
```bash
cd tests
go test ./integration/... -v -timeout 40m
```

#### Performance Tests
```bash
cd tests
go test ./performance/... -v -timeout 25m
```

#### Chaos Tests
```bash
cd tests
go test ./chaos/... -v -timeout 35m
```

#### Cost Optimization Tests
```bash
cd tests
go test ./cost/... -v -timeout 20m
```

#### Security Tests
```bash
cd tests
go test ./security/... -v -timeout 20m
```

#### All Tests
```bash
cd tests
go test ./... -v -timeout 60m
```

## 🔧 Configuration

### Test Variables
Tests use environment-specific configurations:

```hcl
# Test environment
domain_name = "test.example.com"
rate_limit  = 2000

# Staging environment
domain_name = "staging.example.com"

# Production environment
domain_name = "www.example.com"
```

### Environment Variables
```bash
# AWS Configuration
export AWS_REGION=us-east-1
export AWS_PROFILE=your-profile

# Test Configuration
export TEST_ENVIRONMENT=test
export TEST_TIMEOUT=60m
export TEST_PARALLEL=4
```

## 📊 Test Coverage

### Infrastructure Components Tested

| Component | Unit | Integration | E2E | Chaos | Performance | Cost | Security |
|-----------|------|-------------|-----|-------|-------------|------|----------|
| CloudFront | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| WAF | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| S3 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Route 53 | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ |
| ACM | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| CloudTrail | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ |

### Security Controls Tested

- ✅ HTTPS enforcement and TLS 1.2+ validation
- ✅ Security headers (CSP, HSTS, X-Frame-Options, etc.)
- ✅ WAF protection (SQL injection, XSS, bot control)
- ✅ S3 bucket security (encryption, access control)
- ✅ Origin Access Control (OAC) configuration
- ✅ Public access prevention
- ✅ Certificate validation and renewal

## 🔒 Security Testing

### Automated Security Scanning
- **Infrastructure as Code**: TFLint, TFSec, Checkov
- **Runtime Security**: AWS Config, CloudTrail validation
- **Web Application Security**: WAF rule effectiveness
- **SSL/TLS Security**: Certificate and cipher validation
- **Content Security**: Headers and CSP validation

### Security Test Categories

1. **Network Security**
   - CloudFront distribution security
   - WAF rule configuration and effectiveness
   - SSL/TLS configuration validation
   - Geographic access restrictions

2. **Content Security**
   - S3 bucket access control
   - Origin Access Control validation
   - Server-side encryption verification
   - Public access block configuration

3. **Application Security**
   - Security headers validation
   - Content Security Policy enforcement
   - XSS and injection protection
   - Clickjacking prevention

## 📈 Performance Testing

### CDN Performance Metrics
- **Response Time**: P50, P95, P99 percentiles
- **Cache Hit Ratio**: Origin vs edge request ratio
- **Global Latency**: Regional performance validation
- **Throughput**: Requests per second capacity
- **Compression Ratio**: Content optimization effectiveness

### Load Testing Scenarios
- **Concurrent Users**: Multi-user access patterns
- **Traffic Spikes**: Sudden load increase simulation
- **Global Distribution**: Worldwide access validation
- **Cache Performance**: Content delivery optimization
- **Security Overhead**: WAF performance impact

## 💰 Cost Optimization

### CloudFront Cost Optimization
- **Price Class Selection**: Cost-effective edge location usage
- **Origin Shield**: Regional caching optimization
- **Cache Behaviors**: Optimal cache configuration
- **Compression**: Bandwidth reduction
- **Request Optimization**: Efficient request handling

### WAF Cost Management
- **Rule Optimization**: Essential rule selection
- **Rate Limiting**: Cost-effective abuse prevention
- **Request Sampling**: Efficient logging
- **Managed Rules**: AWS managed rule optimization

### Monitoring Costs
- **CloudWatch Metrics**: Efficient monitoring configuration
- **Log Retention**: Optimal log lifecycle management
- **Data Transfer**: Cost-effective data routing
- **Storage Optimization**: S3 lifecycle policies

## 🔄 CI/CD Integration

### GitHub Actions Pipeline
The test suite integrates with comprehensive CI/CD:

```yaml
# Pipeline stages
1. Validate - Terraform validation
2. Security Scan - TFLint, TFSec, Checkov
3. Unit Tests - Fast component validation
4. Integration Tests - Service interaction testing
5. Performance Tests - CDN performance validation
6. Chaos Tests - Resilience testing (scheduled)
7. Cost Tests - Cost optimization validation
8. Security Tests - Vulnerability scanning
9. Compliance Tests - Regulatory compliance
10. E2E Tests - Full deployment validation
11. Deploy Staging - Automated staging deployment
12. Deploy Production - Manual approval deployment
```

### Environment Strategy
- **Test**: Automated testing environment
- **Staging**: Pre-production validation
- **Production**: Manual approval required

## 📋 Test Results and Reporting

### Test Execution Matrix

| Test Type | Environment | Frequency | Duration | AWS Resources |
|-----------|-------------|-----------|----------|---------------|
| Unit Tests | Local/CI | Every commit | 5-15 min | None |
| Integration Tests | Test | PR/Main branch | 20-45 min | Created/Destroyed |
| Performance Tests | Test | Main branch | 25-35 min | Created/Destroyed |
| Chaos Tests | Test | Scheduled | 30-40 min | Created/Destroyed |
| Cost Tests | Test | Main branch | 20-30 min | Created/Destroyed |
| Security Tests | Test | Main branch | 20-25 min | Created/Destroyed |
| E2E Tests | Staging | Release | 30-60 min | Created/Destroyed |

### Success Criteria
- ✅ All unit tests pass
- ✅ Integration tests deploy successfully
- ✅ Security scans pass with no high-severity issues
- ✅ Performance benchmarks meet requirements
- ✅ Cost optimization targets achieved
- ✅ Chaos tests validate resilience
- ✅ E2E tests confirm functionality

## 🛠️ Development

### Adding New Tests

1. **Unit Tests**:
```go
func TestNewComponent(t *testing.T) {
    t.Parallel()
    // Test implementation
}
```

2. **Integration Tests**:
```go
func TestComponentIntegration(t *testing.T) {
    t.Parallel()
    terraformOptions := // setup
    defer terraform.Destroy(t, terraformOptions)
    // Integration test logic
}
```

3. **Performance Tests**:
```go
func TestPerformanceMetrics(t *testing.T) {
    t.Parallel()
    // Performance measurement logic
}
```

### Test Data Management
Test fixtures stored in `fixtures/`:
- Mock AWS responses
- Test configurations
- Sample certificates
- Performance test data

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
go clean -modcache
```

3. **Terraform State Conflicts**
```bash
cd ../..
rm -rf .terraform/
terraform init
```

4. **Test Timeouts**
```bash
export TEST_TIMEOUT=60m
go test -timeout $TEST_TIMEOUT
```

### Debug Mode
```bash
export TF_LOG=DEBUG
export TERRATEST_LOG=debug
go test ./unit/cloudfront_test.go -v
```

## 📚 Resources

### Documentation
- [CloudFront Developer Guide](https://docs.aws.amazon.com/AmazonCloudFront/)
- [AWS WAF Developer Guide](https://docs.aws.amazon.com/waf/)
- [S3 Static Website Hosting](https://docs.aws.amazon.com/AmazonS3/latest/userguide/WebsiteHosting.html)
- [Terratest Documentation](https://terratest.gruntwork.io/)

### Security Resources
- [AWS Security Best Practices](https://aws.amazon.com/architecture/security/)
- [OWASP Web Application Security](https://owasp.org/)
- [Content Security Policy](https://content-security-policy.com/)

## 🤝 Contributing

1. Follow established test patterns
2. Include comprehensive documentation
3. Add appropriate timeouts and cleanup
4. Validate security implications
5. Update CI/CD pipeline if needed

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

**Note**: This test suite ensures production-grade quality, security, performance, and cost optimization for static website deployments on AWS.