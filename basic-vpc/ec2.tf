# ec2.tf

# Security Group for Public EC2 (allow inbound HTTP from anywhere, outbound all)
resource "aws_security_group" "public_sg" {
  vpc_id = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # Allow public access; restrict to specific IPs in production
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "public-sg"
    Environment = var.environment
  }
}

# Security Group for Private EC2 (allow inbound HTTP from public SG, outbound all, including 443 for SSM)
resource "aws_security_group" "private_sg" {
  vpc_id = aws_vpc.main.id

  ingress {
    from_port       = 80
    to_port         = 80
    protocol        = "tcp"
    security_groups = [aws_security_group.public_sg.id] # Allow from public instance
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "private-sg"
    Environment = var.environment
  }
}

# AMI for Amazon Linux 2 (latest)
data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-*-x86_64-gp2"]
  }
}

# User Data script for Apache HTTP server (returns instance private IP)
locals {
  user_data_script = <<-EOF
    #!/bin/bash
    yum update -y
    yum install -y httpd
    systemctl start httpd
    systemctl enable httpd
    PRIVATE_IP=$(curl http://169.254.169.254/latest/meta-data/local-ipv4)
    echo $PRIVATE_IP > /var/www/html/index.html
  EOF
}

# Private EC2 Instance
resource "aws_instance" "private" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.private.id
  vpc_security_group_ids = [aws_security_group.private_sg.id]
  iam_instance_profile   = aws_iam_instance_profile.ssm_profile.name

  user_data = local.user_data_script

  tags = {
    Name        = "private-ec2"
    Environment = var.environment
  }
}

# Public EC2 Instance (with curl to private in user data)
resource "aws_instance" "public" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.public_sg.id]

  user_data = <<-EOF
    ${local.user_data_script}
    # Curl the private instance and log the response
    curl http://${aws_instance.private.private_ip}:80 > /tmp/private_ip_response.log
  EOF

  tags = {
    Name        = "public-ec2"
    Environment = var.environment
  }

  depends_on = [aws_instance.private] # Ensure private is created first
}