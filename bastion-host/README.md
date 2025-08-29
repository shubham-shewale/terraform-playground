# Bastion Host Environment (AWS)

This Terraform configuration provisions a secure bastion host environment for accessing private AWS resources. It creates a hardened VPC infrastructure with comprehensive security controls, encrypted instances, monitoring, and secure access management.

## 🏗️ Architecture Overview

The configuration creates a secure bastion host setup with:
- **Network Security**: VPC with public/private subnets, VPC Flow Logs, and secure routing
- **Instance Security**: Hardened bastion and private instances with encryption and monitoring
- **Access Control**: Restricted SSH access with key-based authentication and fail2ban
- **Monitoring & Auditing**: CloudTrail, CloudWatch alarms, and comprehensive logging
- **Remote Management**: Secure access patterns without exposing private resources

## 📦 What Gets Created

### Network Infrastructure
- **VPC** with DNS support and VPC Flow Logs
- **Public subnet** for bastion host access
- **Private subnet** for protected resources
- **Internet Gateway** and **NAT Gateway** for controlled egress
- **Security Groups** with least-privilege access rules

### Compute Resources
- **Bastion Host** in public subnet with security hardening
- **Private EC2 instance** in private subnet
- **Encrypted EBS volumes** with automatic encryption
- **Detailed monitoring** enabled for all instances
- **Fail2ban integration** for SSH protection

### Security & Access Control
- **SSH Key Pair** for secure authentication
- **IAM roles** with minimal required permissions
- **Security hardening** scripts for OS protection
- **Network ACLs** for additional traffic filtering

### Monitoring & Logging
- **CloudTrail** for API call auditing
- **CloudWatch alarms** for SSH login attempts
- **SNS notifications** for security alerts
- **VPC Flow Logs** for network monitoring

### Storage & Backup
- **Encrypted S3 buckets** for CloudTrail logs
- **Versioning and lifecycle policies**
- **Public access blocks** and security policies

> **Note**: The Terraform state is configured to use an S3 backend in `main.tf`. Ensure the bucket exists or adjust the backend configuration before running.

### Prerequisites
- Terraform v1.3+ (or compatible)
- AWS credentials configured (via environment, shared credentials file, or AWS SSO)
- Existing S3 bucket for remote state (or modify the backend in `main.tf`)
- An SSH public key file to register as the EC2 key pair

### Quick start
```bash
cd bastion-host

# (Optional) Create a tfvars file
cat > bastion.auto.tfvars <<'EOF'
region                = "us-east-1"
vpc_cidr              = "172.16.0.0/16"
azs                   = ["us-east-1a"]
public_subnet_cidrs   = ["172.16.1.0/24"]
private_subnet_cidrs  = ["172.16.10.0/24"]
key_name              = "my-key"
public_key            = file("~/.ssh/id_rsa.pub")
allowed_ssh_cidrs     = ["203.0.113.0/24"]   # replace with your IP/CIDR
environment           = "dev"
EOF

# Initialize (ensure the S3 backend bucket configured in main.tf exists)
terraform init

# Review the plan
terraform plan

# Apply
terraform apply
```

### SSH access flow
1. Obtain the bastion public IP from outputs or the console:
   - Output: `bastion_public_ip`
2. SSH to the bastion:
   ```bash
   ssh -i ~/.ssh/my-key ec2-user@$(terraform output -raw bastion_public_ip)
   ```
3. From the bastion, SSH to the private instance using its private IP:
   ```bash
   ssh -i ~/.ssh/my-key ec2-user@$(terraform output -raw private_instance_ip)
   ```

## 🔧 Configuration Variables

### Required Variables
- `key_name` (string) – **REQUIRED**: Name for the EC2 key pair
- `public_key` (string) – **REQUIRED**: Public key content or `file("<path>")`
- `allowed_ssh_cidrs` (list(string)) – **REQUIRED**: CIDR blocks allowed SSH access (no default for security)

### Optional Variables
- `region` (string) – AWS region. Default: `us-east-1`
- `vpc_cidr` (string) – VPC CIDR block. Default: `172.16.0.0/16`
- `azs` (list(string)) – Availability Zones. Default: `["us-east-1a"]`
- `public_subnet_cidrs` (list(string)) – Public subnet CIDRs. Default: `["172.16.1.0/24"]`
- `private_subnet_cidrs` (list(string)) – Private subnet CIDRs. Default: `["172.16.10.0/24"]`
- `environment` (string) – Environment tag. Default: `dev`

## ⚠️ Security Configuration

**Critical Security Notice**: For production deployments, you must explicitly set:
```hcl
allowed_ssh_cidrs = ["YOUR_TRUSTED_IP_RANGE/32"]  # Replace with your IP
```

Default unrestricted access (`0.0.0.0/0`) has been removed for security. Only explicitly allowed IP ranges can access the bastion host.

### Outputs
- `vpc_id` – Created VPC ID
- `public_subnet_ids` – Public subnet IDs
- `private_subnet_ids` – Private subnet IDs
- `security_group_id` – Security group ID
- `key_pair_name` – EC2 key pair name
- `bastion_public_ip` – Public IPv4 of the bastion host
- `private_instance_ip` – Private IPv4 of the private instance

## 🏛️ Architecture Components

### Network Layer
- **VPC** with DNS support and VPC Flow Logs
- **Public subnet** for bastion host access
- **Private subnet** for protected resources
- **Internet Gateway** and **NAT Gateway** for secure egress
- **Security Groups** with granular access rules

### Security Layer
- **EC2 Key Pair** for SSH authentication
- **Security Groups** with least-privilege rules
- **Network ACLs** for additional traffic filtering
- **IAM Roles** with minimal required permissions
- **Fail2ban** for SSH brute force protection

### Compute Layer
- **Bastion Host** with security hardening and monitoring
- **Private Instance** with encryption and access controls
- **Encrypted EBS volumes** with automatic encryption
- **User data scripts** for OS hardening and security
- **Detailed monitoring** for all instances

### Monitoring Layer
- **CloudTrail** for API call auditing
- **CloudWatch alarms** for SSH login attempts
- **SNS notifications** for security alerts
- **VPC Flow Logs** for network traffic monitoring

### Storage Layer
- **Encrypted S3 buckets** for CloudTrail logs
- **Versioning and lifecycle policies**
- **Public access blocks** and TLS-only policies

AMI selection uses the latest Amazon Linux 2 HVM image with security hardening applied through user data scripts.

### Remote state
This configuration uses an S3 backend configured in `main.tf`:
```hcl
backend "s3" {
  bucket = "my-terraform-state-bucket-211125380337"
  key    = "terraform-playground-bastion-host.tfstate"
  region = "us-east-1"
}
```
Ensure the bucket exists and you have access, or update these values before `terraform init`.

### Destroy
```bash
terraform destroy
```

## 🔒 Security Checklist

### Pre-Deployment
- [x] **Access Control**: Configure `allowed_ssh_cidrs` with specific IP ranges only
- [x] **SSH Keys**: Generate and protect SSH key pairs securely
- [x] **Network Security**: VPC Flow Logs enabled for network monitoring
- [x] **Instance Security**: Fail2ban and SSH hardening configured
- [x] **Monitoring**: CloudTrail and CloudWatch alarms configured
- [x] **Encryption**: All EBS volumes encrypted at rest

### Post-Deployment Verification
- [ ] Verify bastion host is accessible from allowed IPs only
- [ ] Confirm SSH key-based authentication is working
- [ ] Test private instance access through bastion
- [ ] Validate fail2ban is blocking unauthorized access attempts
- [ ] Check CloudWatch alarms are triggering on SSH attempts
- [ ] Verify VPC Flow Logs are capturing traffic

## 🛡️ Security Features

### Access Security
- **Restricted SSH**: Access limited to specific IP ranges
- **Key-Based Authentication**: Password authentication disabled
- **Fail2ban Protection**: Automatic IP blocking for brute force attempts
- **Root Login Disabled**: Direct root access prevented

### Network Security
- **Network Segmentation**: Public/private subnet isolation
- **Security Groups**: Least-privilege access rules
- **VPC Flow Logs**: Network traffic monitoring and analysis

### Instance Security
- **OS Hardening**: Security hardening scripts applied
- **Encrypted Storage**: EBS volumes encrypted at rest
- **Monitoring**: Detailed CloudWatch monitoring enabled

### Monitoring & Alerting
- **SSH Monitoring**: Login attempt tracking and alerting
- **API Auditing**: CloudTrail logging for all AWS API calls
- **Security Alerts**: SNS notifications for security events

## ⚠️ Security Considerations

### Production Deployment
- **IP Restrictions**: Never use `0.0.0.0/0` for SSH access
- **Key Management**: Regularly rotate SSH keys and revoke old ones
- **Session Logging**: Consider enabling SSM Session Manager for auditability
- **Multi-Factor**: Implement additional authentication layers where possible

### Operational Security
- **Access Reviews**: Regularly audit who has bastion access
- **Log Monitoring**: Set up alerts for suspicious SSH activity
- **Credential Rotation**: Rotate IAM credentials and SSH keys regularly
- **Network Monitoring**: Monitor VPC Flow Logs for anomalies

### Advanced Security
- **Session Recording**: Enable SSH session recording for compliance
- **Jump Host Rotation**: Consider rotating bastion hosts regularly
- **Network ACLs**: Implement additional network-level restrictions
- **Endpoint Protection**: Consider adding endpoint protection agents

### Costs
Running EC2 instances, NAT/egress, and CloudWatch/SNS incur charges. Destroy resources when not needed.


