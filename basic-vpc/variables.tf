variable "vpc_cidr" {
  default = "10.0.0.0/16"
}

variable "public_subnet_cidr" {
  default = "10.0.1.0/24"
}

variable "private_subnet_cidr" {
  default = "10.0.2.0/24"
}

variable "availability_zone" {
  default = "us-east-1a"
}

variable "environment" {
  default = "dev"
}

variable "region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "allowed_http_cidrs" {
  description = "CIDR blocks allowed to access HTTP (port 80)"
  type        = list(string)
  default     = [] # No default - must be explicitly set for security
}

variable "allowed_ssh_cidrs" {
  description = "CIDR blocks allowed to access SSH (port 22)"
  type        = list(string)
  default     = [] # No default - must be explicitly set for security
}