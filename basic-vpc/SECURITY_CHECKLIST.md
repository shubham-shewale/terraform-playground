# Basic VPC Security Checklist

## Pre-Deployment Security Review

### Network Configuration
- [x] VPC CIDR ranges are non-overlapping and appropriate for the environment
- [x] Public and private subnets are properly segmented
- [x] Network ACLs are configured with least-privilege rules
- [x] Security groups follow principle of least privilege
- [x] No security groups allow unrestricted access (0.0.0.0/0)
- [x] VPC Flow Logs are enabled for network monitoring
- [x] Route tables are configured for secure traffic flow

### Access Control
- [x] IAM roles follow principle of least privilege
- [x] SSM roles are properly scoped for EC2 access
- [x] No hardcoded credentials in Terraform files
- [x] SSH access is restricted to specific IP ranges
- [x] Root login is disabled on EC2 instances
- [x] Key-based authentication is enforced

### Encryption & Data Protection
- [x] All EBS volumes are encrypted at rest
- [x] S3 buckets have server-side encryption enabled
- [x] CloudTrail logs are stored in encrypted S3 buckets
- [x] TLS 1.2+ is enforced for all HTTPS connections
- [x] VPC endpoints are used for secure service communication

### Monitoring & Logging
- [x] CloudTrail is enabled for API call auditing
- [x] VPC Flow Logs are configured and monitored
- [x] CloudWatch alarms are set up for security events
- [x] SNS notifications are configured for alerts
- [x] Log retention policies are appropriate
- [x] CloudWatch dashboard is configured for monitoring

## Deployment Verification

### Network Security
- [ ] VPC and subnets are created correctly
- [ ] Security groups are applied to instances
- [ ] Network ACLs are functioning as expected
- [ ] VPC endpoints are operational
- [ ] NAT Gateway allows private subnet egress

### Instance Security
- [ ] EC2 instances are running with encrypted volumes
- [ ] User data scripts executed successfully
- [ ] Apache is configured with security headers
- [ ] Fail2ban is installed and running
- [ ] SSH is restricted to key-based authentication

### Access & Connectivity
- [ ] SSM Session Manager works for private instance
- [ ] Public instance is accessible from allowed IPs
- [ ] Private instance communication works internally
- [ ] IAM roles are attached correctly

### Monitoring Setup
- [ ] CloudTrail is logging API calls
- [ ] VPC Flow Logs are capturing traffic
- [ ] CloudWatch alarms are active
- [ ] SNS topics are receiving notifications
- [ ] CloudWatch dashboard displays metrics

## Post-Deployment Security Testing

### Penetration Testing
- [ ] Security group rules are validated
- [ ] Network ACL rules are tested
- [ ] SSH access is restricted properly
- [ ] Web application security headers are present

### Vulnerability Assessment
- [ ] EC2 instances are scanned for vulnerabilities
- [ ] Operating system is up to date
- [ ] Apache configuration is secure
- [ ] SSL/TLS configurations are validated

### Access Review
- [ ] IAM permissions are reviewed
- [ ] SSH keys are rotated regularly
- [ ] Access logs are monitored
- [ ] Least privilege is maintained

## Ongoing Security Maintenance

### Weekly Tasks
- [ ] Review CloudWatch alarms and logs
- [ ] Check for security updates on EC2 instances
- [ ] Monitor VPC Flow Logs for suspicious activity
- [ ] Verify backup and recovery procedures
- [ ] Review SNS notifications and alerts

### Monthly Tasks
- [ ] Conduct security group review
- [ ] Audit IAM permissions and access
- [ ] Update security documentation
- [ ] Review monitoring and alerting rules
- [ ] Validate encryption is working correctly

### Quarterly Tasks
- [ ] Perform comprehensive security assessment
- [ ] Update incident response procedures
- [ ] Review and update security policies
- [ ] Conduct security awareness training

## Security Metrics & KPIs

### Monitoring Metrics
- Number of security group violations
- VPC Flow Log analysis results
- CloudTrail API call patterns
- SSH login attempt monitoring
- SSL/TLS handshake failures

### Compliance Metrics
- Security control implementation status
- Vulnerability remediation time
- Incident response time
- Access review completion rate

## Emergency Contacts & Escalation

### Security Team
- Security Lead: [Contact Information]
- Infrastructure Security: [Contact Information]
- Incident Response: [Contact Information]

### AWS Support
- AWS Support Plan: [Plan Level]
- TAM Contact: [Contact Information]
- Escalation Path: [Escalation Procedures]

## Notes & Recommendations

### Security Best Practices Implemented
1. **Defense in Depth**: Multiple security layers (SG + NACL)
2. **Least Privilege**: Minimal required permissions
3. **Encryption Everywhere**: Data at rest and in transit
4. **Comprehensive Monitoring**: Full audit trail and alerting
5. **Secure Access**: No direct internet access to private resources

### Recommendations for Production
1. Implement multi-AZ deployment for high availability
2. Use AWS Config for compliance monitoring
3. Implement AWS Security Hub for centralized security findings
4. Set up AWS GuardDuty for threat detection
5. Consider AWS Inspector for automated vulnerability assessments

### Cost Considerations
- VPC Flow Logs: ~$0.50 per GB
- CloudTrail: ~$2.00 per 100,000 events
- CloudWatch: ~$0.30 per GB ingested
- EBS Encryption: No additional cost
- SSM: Free for EC2 instances

## Change Log

### Version 1.0 - Initial Security Checklist
- Comprehensive security controls implemented
- Monitoring and alerting configured
- Encryption enabled for all resources
- Access controls hardened

### Version 1.1 - Enhanced Security
- Network ACLs added for defense in depth
- VPC endpoints configured for secure communication
- Enhanced CloudWatch monitoring
- Improved documentation and procedures