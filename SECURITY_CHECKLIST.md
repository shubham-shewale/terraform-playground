# Security Checklist for Terraform Deployments

## Pre-Deployment Security Checklist

### Network Security
- [ ] VPC CIDR ranges are non-overlapping and appropriate
- [ ] Security groups follow principle of least privilege
- [ ] No security groups allow 0.0.0.0/0 for critical ports (22, 80, 443)
- [ ] VPC Flow Logs are enabled for all VPCs
- [ ] Network ACLs are configured appropriately
- [ ] Subnets are properly segmented (public/private)

### Access Control
- [ ] IAM roles follow principle of least privilege
- [ ] No hardcoded credentials in Terraform files
- [ ] SSH access is restricted to specific IP ranges
- [ ] Key-based authentication is enforced for SSH
- [ ] Root login is disabled on all instances
- [ ] Bastion hosts are properly secured

### Encryption
- [ ] All EBS volumes are encrypted
- [ ] All S3 buckets have server-side encryption enabled
- [ ] TLS 1.2+ is enforced for all HTTPS connections
- [ ] ACM certificates are properly validated
- [ ] KMS keys are used for sensitive data

### Monitoring and Logging
- [ ] CloudTrail is enabled for all regions
- [ ] CloudWatch logs are configured for all resources
- [ ] Security alerts are configured via SNS
- [ ] VPC Flow Logs are enabled
- [ ] Access logging is enabled for S3 and CloudFront

### Application Security
- [ ] WAF is configured for web applications
- [ ] Security headers are implemented
- [ ] Content Security Policy is configured
- [ ] Rate limiting is implemented
- [ ] Input validation is in place

## Post-Deployment Security Checklist

### Verification
- [ ] All security groups are properly configured
- [ ] Encryption is working as expected
- [ ] Logging is capturing all relevant events
- [ ] Alerts are functioning correctly
- [ ] Access controls are working properly

### Testing
- [ ] Penetration testing has been conducted
- [ ] Security scanning tools have been run
- [ ] Vulnerability assessment is complete
- [ ] Access reviews have been performed
- [ ] Incident response procedures are tested

## Ongoing Security Maintenance

### Weekly Tasks
- [ ] Review CloudWatch alarms and logs
- [ ] Check for security updates and patches
- [ ] Review access logs for suspicious activity
- [ ] Verify backup and recovery procedures

### Monthly Tasks
- [ ] Conduct security group review
- [ ] Review IAM permissions and access
- [ ] Update security documentation
- [ ] Review compliance requirements

### Quarterly Tasks
- [ ] Conduct comprehensive security audit
- [ ] Update security policies and procedures
- [ ] Review and update incident response plan
- [ ] Conduct security awareness training

## Security Tools and Resources

### AWS Security Services
- [ ] AWS Config for compliance monitoring
- [ ] AWS Security Hub for security findings
- [ ] AWS GuardDuty for threat detection
- [ ] AWS Inspector for vulnerability assessment
- [ ] AWS Macie for data protection

### Third-Party Tools
- [ ] Terraform security scanning (tfsec, checkov)
- [ ] Container security scanning
- [ ] Dependency vulnerability scanning
- [ ] Static code analysis
- [ ] Dynamic application security testing

## Incident Response

### Preparation
- [ ] Incident response team is identified
- [ ] Contact information is up to date
- [ ] Escalation procedures are documented
- [ ] Communication plan is in place

### Response
- [ ] Incident detection and classification
- [ ] Containment and eradication
- [ ] Evidence preservation
- [ ] Communication with stakeholders
- [ ] Post-incident review and lessons learned

## Compliance and Governance

### Documentation
- [ ] Security policies are documented
- [ ] Procedures are up to date
- [ ] Risk assessments are current
- [ ] Compliance reports are generated

### Auditing
- [ ] Regular security audits are conducted
- [ ] Compliance monitoring is active
- [ ] Findings are tracked and remediated
- [ ] Continuous improvement is implemented

## Emergency Contacts

### Security Team
- Security Lead: [Contact Information]
- Incident Response Lead: [Contact Information]
- Infrastructure Lead: [Contact Information]

### External Contacts
- AWS Support: [Contact Information]
- Security Vendor: [Contact Information]
- Legal Team: [Contact Information]

## Notes

- This checklist should be reviewed and updated regularly
- All items should be completed before production deployment
- Regular reviews ensure ongoing security effectiveness
- Document any deviations with justification and risk assessment
