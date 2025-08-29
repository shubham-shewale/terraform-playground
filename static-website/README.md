# Static Website on S3 + CloudFront + WAF (AWS)

This Terraform configuration provisions a comprehensive, secure static website infrastructure with enterprise-grade security controls, global content delivery, and comprehensive monitoring.

## 🏗️ Architecture Overview

The configuration creates a secure static website stack with:
- **Content Security**: Private S3 bucket with Origin Access Control (OAC)
- **Global Delivery**: CloudFront distribution with HTTPS and security headers
- **Web Protection**: AWS WAF with managed rules and rate limiting
- **Monitoring & Logging**: Comprehensive logging and security monitoring
- **DNS Management**: Route 53 integration for custom domains

## 📦 What Gets Created

### Content Storage
- **Private S3 Bucket** with website configuration
- **Origin Access Control** (OAC) for secure CloudFront access
- **Versioning and encryption** for content protection
- **Public access blocks** and strict bucket policies

### Content Delivery
- **CloudFront Distribution** with global edge locations
- **HTTPS Enforcement** with ACM SSL certificates
- **Security Headers** injected via response headers policy
- **Compression and optimization** for performance

### Security Protection
- **AWS WAF Web ACL** with comprehensive rule sets
- **Rate Limiting** to prevent abuse
- **Managed Rules** for common attack patterns
- **Logging Pipeline** via Kinesis Firehose to S3

### DNS & Access
- **Route 53 Alias** record for custom domain
- **Certificate Validation** via DNS for ACM
- **Access Control** through CloudFront OAC

### Monitoring & Logging
- **CloudTrail** for API call auditing
- **WAF Logging** to dedicated S3 buckets
- **CloudFront Access Logs** for request analysis
- **Security Monitoring** and alerting

> **Note**: Remote state is configured in `backend.tf` to an S3 bucket. Ensure it exists or adjust before running.

### Prerequisites
- Terraform v1.3+
- AWS credentials configured
- A public hosted zone in Route 53 matching `var.domain_name`
- Ownership/control of the domain in Route 53 to validate ACM

### Quick start
```bash
cd static-website

# Optionally customize domain in a tfvars file
cat > static-website.auto.tfvars <<'EOF'
domain_name = "example.com" # must exist in Route 53 hosted zones
EOF

# Initialize (ensure S3 backend in backend.tf exists or update it)
terraform init

# Review
terraform plan -out tfplan

# Apply
terraform apply tfplan
```

### After apply
- Wait for ACM validation to complete (DNS record is created automatically). CloudFront will deploy once the certificate is validated.
- Fetch outputs:
  - `cloudfront_domain` – access the site at `https://<cloudfront_domain>` while DNS propagates
  - `s3_bucket_name` – your website bucket
- Once Route 53 alias is live, access at `https://<domain_name>`.

### Inputs
- `domain_name` (string) – Domain for the website (must be in Route 53). No default; set via tfvars or environment.

### Outputs
- `cloudfront_domain` – CloudFront distribution domain
- `s3_bucket_name` – Website S3 bucket name

## 🛡️ Security Controls

### Content Protection
- **Private S3 Bucket** with strict access controls
- **Origin Access Control** (OAC) for CloudFront-only access
- **Server-Side Encryption** (SSE-S3) for all objects
- **Versioning** enabled for content protection
- **Public Access Blocks** preventing unauthorized access

### Network Security
- **HTTPS Enforcement** with TLS 1.2+ minimum
- **Security Headers** injected via CloudFront response policy
- **Origin Shield** for improved cache efficiency and security
- **Geographic Restrictions** configurable for content access

### Web Application Firewall
- **AWS WAF Web ACL** with comprehensive protection:
  - **AWSManagedRulesCommonRuleSet**: Common web exploits
  - **AWSManagedRulesSQLiRuleSet**: SQL injection protection
  - **AWSManagedRulesKnownBadInputsRuleSet**: Malformed requests
  - **AWSManagedRulesBotControlRuleSet**: Bot traffic management
  - **AWSManagedRulesAnonymousIpList**: Anonymous IP blocking
- **Rate Limiting**: Configurable request limits per IP
- **Logging Pipeline**: WAF logs to S3 via Kinesis Firehose

### Access Control
- **Bucket Policies** allowing only CloudFront access
- **TLS-Only Access** enforced on log buckets
- **Encrypted Object Writes** for all log data
- **Certificate Validation** via DNS for ACM

### Monitoring & Auditing
- **CloudTrail Integration** for API call auditing
- **WAF Request Logging** for security analysis
- **CloudFront Access Logs** for request analysis
- **Security Event Monitoring** and alerting

### Architecture and flow
1. Users resolve `var.domain_name` → Route 53 alias to CloudFront
2. CloudFront serves from S3 origin using Origin Access Control (no public S3)
3. WAF filters malicious traffic; CloudFront adds security headers and enforces TLS 1.2+
4. Access logs and WAF logs flow to dedicated S3 buckets

### Remote state
Defined in `backend.tf`:
```hcl
backend "s3" {
  bucket = "my-terraform-state-bucket-381492134996"
  key    = "terraform-playground-static-website.tfstate"
  region = "us-east-1"
}
```

### Destroy
```bash
terraform destroy
```

## 🔒 Security Checklist

### Pre-Deployment
- [x] **Domain Ownership**: Verify domain exists in Route 53
- [x] **Certificate Validation**: Ensure DNS records can be created
- [x] **WAF Configuration**: Rate limiting and rules configured
- [x] **Security Headers**: Response headers policy configured
- [x] **Encryption**: S3 SSE and HTTPS enforcement enabled
- [x] **Access Control**: OAC and bucket policies configured

### Post-Deployment Verification
- [ ] Verify CloudFront distribution is deployed
- [ ] Confirm ACM certificate is validated and active
- [ ] Test HTTPS access to the website
- [ ] Validate security headers are present
- [ ] Check WAF is blocking malicious requests
- [ ] Verify S3 bucket is not publicly accessible

## ⚠️ Security Considerations

### Content Security
- **CSP Headers**: Strict Content Security Policy implemented
- **XSS Protection**: Multiple layers of XSS prevention
- **Frame Options**: Clickjacking protection enabled
- **HSTS**: HTTP Strict Transport Security configured

### Access Control
- **Origin Access Control**: CloudFront-only S3 access
- **Bucket Policies**: Least-privilege access rules
- **Public Access Blocks**: All public access prevented
- **TLS Enforcement**: HTTPS-only access required

### Monitoring & Response
- **WAF Logging**: Comprehensive request logging
- **CloudTrail**: API call auditing enabled
- **Access Logs**: CloudFront request analysis
- **Security Monitoring**: Automated threat detection

### Performance & Security Balance
- **Edge Locations**: Global content delivery with security
- **Caching**: Optimized caching with security headers
- **Compression**: GZIP compression for performance
- **Origin Shield**: Improved cache hit ratios

## 📊 Cost Considerations

### CloudFront Costs
- **Data Transfer**: ~$0.085/GB for first 10TB
- **Requests**: ~$0.0075 per 10,000 HTTPS requests
- **Origin Shield**: Additional ~$0.030/GB for enhanced caching

### WAF Costs
- **Web ACL**: ~$5.00 per month
- **Rules**: ~$1.00 per rule per month
- **Requests**: ~$0.60 per 1 million requests

### S3 Costs
- **Storage**: ~$0.023/GB for standard storage
- **Requests**: ~$0.0004 per 1,000 GET requests
- **Data Transfer**: ~$0.09/GB to CloudFront

### Monitoring Costs
- **CloudTrail**: ~$2.00 per 100,000 events
- **CloudWatch Logs**: ~$0.50/GB ingested
- **Kinesis Firehose**: ~$0.029/GB processed

## 🚀 Best Practices

### Security
1. **Regular Updates**: Keep WAF rules and managed rule sets updated
2. **Monitor Logs**: Regularly review WAF and CloudFront access logs
3. **Certificate Renewal**: Monitor ACM certificate expiration
4. **Access Review**: Regularly audit who has access to the S3 bucket

### Performance
1. **Cache Optimization**: Configure appropriate cache behaviors
2. **Compression**: Enable compression for text-based content
3. **Edge Locations**: Utilize CloudFront's global network
4. **Monitoring**: Track cache hit ratios and performance metrics

### Operations
1. **Backup Strategy**: Implement content backup procedures
2. **Disaster Recovery**: Plan for content distribution failures
3. **Version Control**: Use S3 versioning for content rollback
4. **Automation**: Implement CI/CD for content deployment

## 📋 Troubleshooting

### Common Issues
- **Certificate Pending**: Wait for DNS propagation and ACM validation
- **Access Denied**: Check OAC configuration and bucket policies
- **Slow Loading**: Review cache behaviors and origin configuration
- **WAF Blocking**: Check WAF rules and rate limiting settings

### Monitoring
- **CloudWatch Metrics**: Monitor 4xx/5xx error rates
- **WAF Metrics**: Track blocked requests and rule matches
- **CloudFront Metrics**: Monitor cache hit ratios and latency
- **Access Logs**: Analyze request patterns and performance

## 🔄 Updates & Maintenance

### Regular Tasks
1. **Security Updates**: Review and update WAF managed rules
2. **Certificate Management**: Monitor SSL certificate expiration
3. **Log Analysis**: Review access logs for security threats
4. **Performance Monitoring**: Track CloudFront performance metrics

### Content Management
1. **Deployment Process**: Implement secure content deployment
2. **Version Control**: Use S3 versioning for content management
3. **Backup Strategy**: Regular content backups and testing
4. **Access Control**: Maintain least-privilege access principles

## 📞 Support & Resources

### AWS Documentation
- [CloudFront Developer Guide](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/)
- [AWS WAF Developer Guide](https://docs.aws.amazon.com/waf/latest/developerguide/)
- [S3 Static Website Hosting](https://docs.aws.amazon.com/AmazonS3/latest/userguide/WebsiteHosting.html)

### Security Resources
- [AWS Security Best Practices](https://aws.amazon.com/architecture/security-identity-compliance/)
- [OWASP Web Application Security](https://owasp.org/www-project-top-ten/)
- [Content Security Policy](https://content-security-policy.com/)

### Diagram
![static-website](static-website.png)