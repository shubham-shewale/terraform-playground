resource "aws_security_group" "bastion" {
  name        = "bastion_security_group"
  description = "Security group for bastion host with restricted SSH access"
  vpc_id      = var.vpc_id

  ingress {
    description = "SSH from allowed IPs only"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = length(var.allowed_ssh_cidrs) > 0 ? var.allowed_ssh_cidrs : ["127.0.0.1/32"] # Default deny if not specified
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

# Private instance SG (SSH only from bastion SG; no public ingress)
resource "aws_security_group" "private" {
  name        = "private_instance_security_group"
  description = "Private instance SG; SSH only from bastion SG"
  vpc_id      = var.vpc_id

  ingress {
    description     = "SSH from bastion SG"
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = [aws_security_group.bastion.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "private_instance_security_group"
    Environment = var.environment
  }
}

output "bastion_security_group_id" { value = aws_security_group.bastion.id }
output "private_security_group_id" { value = aws_security_group.private.id }