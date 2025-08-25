# IAM Role for SSM on Private Instance
resource "aws_iam_role" "ssm_role" {
  name = "ssm-role-for-private-ec2"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      },
    ]
  })

  tags = {
    Name        = "ssm-role"
    Environment = var.environment
  }
}

# Attach AmazonSSMManagedInstanceCore policy
resource "aws_iam_role_policy_attachment" "ssm_policy" {
  role       = aws_iam_role.ssm_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

# Instance Profile
resource "aws_iam_instance_profile" "ssm_profile" {
  name = "ssm-profile-for-private-ec2"
  role = aws_iam_role.ssm_role.name
}

# Security Group for VPC Endpoints
resource "aws_security_group" "vpc_endpoint_sg" {
  vpc_id = aws_vpc.main.id

  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.private_sg.id]
  }

  tags = {
    Name        = "vpc-endpoint-sg"
    Environment = var.environment
  }
}

# VPC Endpoint for SSM
resource "aws_vpc_endpoint" "ssm" {
  vpc_id              = aws_vpc.main.id
  service_name        = "com.amazonaws.us-east-1.ssm"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = [aws_subnet.private.id]
  security_group_ids  = [aws_security_group.vpc_endpoint_sg.id]
  private_dns_enabled = true

  tags = {
    Name        = "ssm-endpoint"
    Environment = var.environment
  }
}

# VPC Endpoint for EC2 Messages
resource "aws_vpc_endpoint" "ec2messages" {
  vpc_id              = aws_vpc.main.id
  service_name        = "com.amazonaws.us-east-1.ec2messages"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = [aws_subnet.private.id]
  security_group_ids  = [aws_security_group.vpc_endpoint_sg.id]
  private_dns_enabled = true

  tags = {
    Name        = "ec2messages-endpoint"
    Environment = var.environment
  }
}

# VPC Endpoint for SSM Messages
resource "aws_vpc_endpoint" "ssmmessages" {
  vpc_id              = aws_vpc.main.id
  service_name        = "com.amazonaws.us-east-1.ssmmessages"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = [aws_subnet.private.id]
  security_group_ids  = [aws_security_group.vpc_endpoint_sg.id]
  private_dns_enabled = true

  tags = {
    Name        = "ssmmessages-endpoint"
    Environment = var.environment
  }
}