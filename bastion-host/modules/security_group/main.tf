resource "aws_security_group" "this" {
  name        = "bastion_security_group"
  description = "Security group for bastion host with restricted SSH access"
  vpc_id      = var.vpc_id

  ingress {
    description = "SSH from allowed IPs only"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.allowed_ssh_cidrs # Restrict to specific IPs
  }

  # Allow outbound traffic to private subnets only
  egress {
    description = "Allow outbound to private subnets"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.private_subnet_cidrs
  }

  # Allow HTTPS for updates and SSM
  egress {
    description = "Allow HTTPS for updates"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow HTTP for updates
  egress {
    description = "Allow HTTP for updates"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow DNS
  egress {
    description = "Allow DNS"
    from_port   = 53
    to_port     = 53
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "Allow DNS UDP"
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { 
    Name = "bastion_security_group"
    Environment = var.environment
  }
}

output "security_group_id" { value = aws_security_group.this.id }
