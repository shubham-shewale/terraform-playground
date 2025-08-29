## Bastion Host Environment (AWS)

This Terraform configuration provisions a minimal AWS environment for secure access to private resources via a bastion host. It creates a VPC with public and private subnets, a bastion EC2 instance in the public subnet, a private EC2 instance in the private subnet, security groups, an EC2 key pair, and basic IAM/CloudWatch/SNS resources for observability.

### What gets created
- **VPC** with one public and one private subnet (single AZ by default)
- **Security Group** allowing SSH to bastion from configured CIDRs and internal access to the private instance
- **EC2 Key Pair** (from your provided public key)
- **Bastion host** (Amazon Linux 2) in the public subnet with an IAM role and instance profile
- **Private EC2 instance** (Amazon Linux 2) in the private subnet
- **CloudWatch** log group and a sample metric alarm (SSH login attempts) with an **SNS** topic for alerts

> Note: The Terraform state is configured to use an S3 backend in `main.tf`. Ensure the bucket exists or adjust the backend configuration before running.

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

### Inputs
- `region` (string) – AWS region. Default: `us-east-1`
- `vpc_cidr` (string) – VPC CIDR block. Default: `172.16.0.0/16`
- `azs` (list(string)) – Availability Zones. Default: `["us-east-1a"]`
- `public_subnet_cidrs` (list(string)) – Public subnet CIDRs. Default: `["172.16.1.0/24"]`
- `private_subnet_cidrs` (list(string)) – Private subnet CIDRs. Default: `["172.16.10.0/24"]`
- `key_name` (string) – Name for the EC2 key pair. Required
- `public_key` (string) – Public key content or `file("<path>")`. Required
- `allowed_ssh_cidrs` (list(string)) – CIDRs allowed to SSH to bastion. Default: `["0.0.0.0/0"]` (change in production)
- `environment` (string) – Tagging/environment label. Default: `dev`

### Outputs
- `vpc_id` – Created VPC ID
- `public_subnet_ids` – Public subnet IDs
- `private_subnet_ids` – Private subnet IDs
- `security_group_id` – Security group ID
- `key_pair_name` – EC2 key pair name
- `bastion_public_ip` – Public IPv4 of the bastion host
- `private_instance_ip` – Private IPv4 of the private instance

### Module and resource flow
The top-level `main.tf` wires together the following components:
- `module "vpc"` – Provisions VPC, one public and one private subnet based on inputs
- `module "security_group"` – Creates an SG for SSH ingress to bastion and internal access
- `module "key_pair"` – Registers your provided public key as an EC2 key pair
- `module "bastion"` – Launches a bastion EC2 instance in the public subnet, attaches SG, key, IAM instance profile
- `module "private_instance"` – Launches a private EC2 instance in the private subnet, attached to the internal SG
- IAM role/policy/instance profile for the bastion plus CloudWatch log group and a sample SSH attempts alarm that publishes to an SNS topic

AMI selection is done with the `aws_ami` data source for the latest Amazon Linux 2 HVM image owned by Amazon.

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

### Security notes
- Restrict `allowed_ssh_cidrs` to your trusted IPs only
- Rotate and protect your SSH keys; prefer short-lived access methods where possible
- Consider enabling session logging/SSM Session Manager for stronger auditability
- Monitor and tune the CloudWatch alarm thresholds and destinations
- Apply least-privilege to the bastion IAM role; expand only as required

### Costs
Running EC2 instances, NAT/egress, and CloudWatch/SNS incur charges. Destroy resources when not needed.


