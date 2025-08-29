variable "region" {
  description = "AWS region"
  default     = "us-east-1"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  default     = "172.16.0.0/16"
}

variable "azs" {
  type        = list(string)
  description = "Availability Zones"
  default     = ["us-east-1a"]
}

variable "public_subnet_cidrs" {
  type        = list(string)
  description = "Public subnet CIDRs"
  default     = ["172.16.1.0/24"]
}

variable "private_subnet_cidrs" {
  type        = list(string)
  description = "Private subnet CIDRs"
  default     = ["172.16.10.0/24"]
}

variable "key_name" {
  description = "Key pair name"
}

variable "public_key" {
  description = "Path to your public key"
}

variable "allowed_ssh_cidrs" {
  description = "CIDR blocks allowed to access SSH (port 22)"
  type        = list(string)
  default     = [] # No default - must be explicitly set for security
}

variable "environment" {
  description = "Environment name for tagging"
  type        = string
  default     = "dev"
}
