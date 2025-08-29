variable "vpc_id" { 
  type = string 
}

variable "allowed_ssh_cidrs" {
  description = "CIDR blocks allowed to access SSH (port 22)"
  type        = list(string)
  default     = ["0.0.0.0/0"] # Restrict this in production
}

variable "private_subnet_cidrs" {
  description = "Private subnet CIDR blocks for egress rules"
  type        = list(string)
  default     = ["172.16.10.0/24"]
}

variable "environment" {
  description = "Environment name for tagging"
  type        = string
  default     = "dev"
}
