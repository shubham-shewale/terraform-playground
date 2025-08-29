# Basic VPC (AWS)

This Terraform configuration provisions a secure, production-ready VPC infrastructure with comprehensive security controls, monitoring, and access management. It creates a VPC with public and private subnets, hardened EC2 instances, VPC Flow Logs, CloudTrail auditing, CloudWatch monitoring, and secure Systems Manager access.

## 🏗️ Architecture Overview

The configuration creates a secure VPC environment with:
- **Network Security**: VPC with public/private subnets, Network ACLs, and secure routing
- **Instance Security**: Hardened EC2 instances with encryption, security headers, and monitoring
- **Access Control**: Least-privilege security groups and IAM roles
- **Monitoring & Auditing**: VPC Flow Logs, CloudTrail, and CloudWatch alarms
- **Remote Management**: SSM access for private instances without internet exposure

## 📦 What Gets Created

### Network Infrastructure
- **VPC** with DNS support and hostnames enabled
- **Public subnet** and **private subnet** in configurable availability zone
- **Internet Gateway** and **NAT Gateway** with secure route tables
- **Network ACLs** providing defense-in-depth security layer

### Compute Resources
- **Public EC2 instance** in public subnet with Apache web server
- **Private EC2 instance** in private subnet with Apache web server
- **Encrypted EBS volumes** (gp3) with automatic encryption
- **Detailed monitoring** enabled for all instances
- **Security hardening** with fail2ban and SSH restrictions

### Security & Access Control
- **Security Groups** with least-privilege access rules
- **IAM roles** and instance profiles for SSM access
- **VPC Endpoints** for secure SSM communication
- **Network ACLs** for additional traffic filtering

### Monitoring & Logging
- **VPC Flow Logs** to CloudWatch Logs for network monitoring
- **CloudTrail** with encrypted S3 storage for API auditing
- **CloudWatch alarms** (CPU, Network, Status Check) with SNS notifications
- **CloudWatch dashboard** for infrastructure monitoring

### Storage & Backup
- **Encrypted S3 buckets** for CloudTrail logs
- **Versioning and lifecycle policies** for log management
- **Public access blocks** and TLS-only policies

> **Note**: Remote state is configured in `backend.tf` to an S3 bucket. Ensure it exists or adjust before running.

### Prerequisites
- Terraform v1.3+
- AWS credentials configured
- Existing S3 bucket for the backend (or edit `backend.tf`)

### Quick start
```bash
cd basic-vpc

# Initialize (ensure S3 backend in backend.tf exists or update it)
terraform init

# Review changes
terraform plan -out tfplan

# Apply
terraform apply tfplan
```

### Access and test
- The public instance exposes HTTP on port 80 from `allowed_http_cidrs`. Visit its public IP.
- The public instance also curls the private instance on port 80 at boot; see `/tmp/private_ip_response.log` on the public instance.
- Use SSM Session Manager to connect to the private instance (no bastion required) once SSM agent registers.

## 🔧 Configuration Variables

### Required Variables
- `allowed_http_cidrs` (list(string)) – **REQUIRED**: CIDR blocks allowed HTTP access (no default for security)
- `allowed_ssh_cidrs` (list(string)) – **REQUIRED**: CIDR blocks allowed SSH access (no default for security)

### Optional Variables
- `vpc_cidr` (string) – VPC CIDR block. Default: `10.0.0.0/16`
- `public_subnet_cidr` (string) – Public subnet CIDR. Default: `10.0.1.0/24`
- `private_subnet_cidr` (string) – Private subnet CIDR. Default: `10.0.2.0/24`
- `availability_zone` (string) – Availability zone. Default: `us-east-1a`
- `environment` (string) – Environment tag. Default: `dev`
- `region` (string) – AWS region. Default: `us-east-1`

## ⚠️ Security Configuration

**Important**: For production deployments, you must explicitly set:
```hcl
allowed_http_cidrs = ["YOUR_TRUSTED_IP_RANGE/32"]  # Replace with your IP
allowed_ssh_cidrs  = ["YOUR_TRUSTED_IP_RANGE/32"]  # Replace with your IP
```

Default unrestricted access (`0.0.0.0/0`) has been removed for security.

### Outputs
- `public_instance_public_ip`
- `public_instance_private_ip`
- `private_instance_private_ip`

## 🏛️ Architecture Components

### Network Layer
- **VPC** with DNS support and VPC Flow Logs to CloudWatch
- **Network ACLs** providing defense-in-depth security
- **Public/Private subnets** with secure route tables
- **Internet Gateway** and **NAT Gateway** for controlled egress
- **VPC Endpoints** for secure SSM communication

### Security Layer
- **Security Groups** with least-privilege access rules
- **Network ACLs** for additional traffic filtering
- **IAM Roles** following principle of least privilege
- **Encrypted storage** for all data at rest

### Compute Layer
- **EC2 instances** with Amazon Linux 2 and security hardening
- **Encrypted EBS volumes** (gp3) with automatic encryption
- **User data scripts** for Apache hardening and security headers
- **SSM integration** for secure remote management

### Monitoring Layer
- **CloudTrail** with encrypted S3 storage for API auditing
- **CloudWatch alarms** for CPU, Network, and Status monitoring
- **SNS notifications** for security alerts
- **CloudWatch dashboard** for infrastructure visibility

### Storage Layer
- **Encrypted S3 buckets** for log storage
- **Versioning and lifecycle policies** for data management
- **Public access blocks** and TLS-only policies

### Remote state
Defined in `backend.tf`:
```hcl
backend "s3" {
  bucket = "my-terraform-state-bucket-381492134996"
  key    = "terraform-playground-basic-vpc.tfstate"
  region = "us-east-1"
}
```
Make sure this bucket exists and you have access, or update the values before `terraform init`.

### Destroy
```bash
terraform destroy
```

## 🔒 Security Checklist

### Pre-Deployment
- [x] **Access Control**: Configure `allowed_http_cidrs` and `allowed_ssh_cidrs` with specific IP ranges
- [x] **Network Security**: VPC Flow Logs enabled for network monitoring
- [x] **Encryption**: All EBS volumes and S3 buckets encrypted
- [x] **Monitoring**: CloudTrail and CloudWatch alarms configured
- [x] **Remote Access**: SSM endpoints configured for secure management

### Post-Deployment Verification
- [ ] Verify VPC Flow Logs are capturing traffic
- [ ] Confirm CloudTrail is logging API calls
- [ ] Test SSM Session Manager access to private instance
- [ ] Validate security group rules are working as expected
- [ ] Check CloudWatch alarms are triggering correctly

## 🛡️ Security Features

### Network Security
- **Defense in Depth**: Security Groups + Network ACLs
- **Traffic Monitoring**: VPC Flow Logs with CloudWatch integration
- **Secure Routing**: NAT Gateway for private subnet egress

### Instance Security
- **OS Hardening**: Security headers, fail2ban, SSH restrictions
- **Encryption**: EBS volumes encrypted at rest
- **Access Control**: Key-based SSH authentication only

### Monitoring & Alerting
- **API Auditing**: CloudTrail with encrypted S3 storage
- **Performance Monitoring**: CPU, Network, and Status alarms
- **Security Alerts**: SNS notifications for security events

## ⚠️ Security Considerations

### Production Deployment
- **Restrict Access**: Never use `0.0.0.0/0` for production workloads
- **Multi-AZ**: Consider deploying across multiple availability zones
- **Load Balancing**: Use ALB/NLB for public ingress instead of direct EC2 access
- **Backup Strategy**: Implement regular backups and disaster recovery

### Operational Security
- **Credential Rotation**: Regularly rotate access keys and SSH keys
- **Monitoring**: Set up alerts and monitoring dashboards
- **Access Reviews**: Regularly audit IAM permissions and access patterns
- **Log Retention**: Configure appropriate log retention policies

### Cost Optimization
- **Resource Sizing**: Right-size EC2 instances for your workload
- **Log Management**: Set appropriate retention periods for logs
- **Monitoring**: Use CloudWatch metrics efficiently

### Diagram
![basic-vpc topology](basic-vpc.png)