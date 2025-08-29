# ec2.tf

# Security Group for Public EC2 (restrict access to specific IPs)
resource "aws_security_group" "public_sg" {
  vpc_id = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = var.allowed_http_cidrs # Restrict to specific IPs
  }

  # Allow SSH access only from specific IPs
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.allowed_ssh_cidrs
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

  # Allow SSH access only from bastion or specific IPs
  ingress {
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = [aws_security_group.public_sg.id]
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

# User Data script for Apache HTTP server with security hardening
locals {
  user_data_script = <<-EOF
    #!/bin/bash
    # Security hardening script
    yum update -y
    yum install -y httpd mod_security mod_ssl
    
    # Configure Apache security headers
    cat > /etc/httpd/conf.d/security-headers.conf << 'APACHE_EOF'
    Header always set X-Content-Type-Options nosniff
    Header always set X-Frame-Options DENY
    Header always set X-XSS-Protection "1; mode=block"
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
    Header always set Referrer-Policy "strict-origin-when-cross-origin"
    Header always set Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"
    APACHE_EOF
    
    # Disable Apache server signature
    sed -i 's/ServerTokens OS/ServerTokens Prod/' /etc/httpd/conf/httpd.conf
    sed -i 's/ServerSignature On/ServerSignature Off/' /etc/httpd/conf/httpd.conf
    
    # Start and enable Apache
    systemctl start httpd
    systemctl enable httpd
    
    # Get instance private IP
    PRIVATE_IP=$(curl http://169.254.169.254/latest/meta-data/local-ipv4)
    echo $PRIVATE_IP > /var/www/html/index.html
    
    # Security: Remove default Apache welcome page
    rm -f /etc/httpd/conf.d/welcome.conf
    
    # Security: Set proper file permissions
    chown -R apache:apache /var/www/html
    chmod -R 755 /var/www/html
  EOF
}

# Private EC2 Instance with encryption at rest
resource "aws_instance" "private" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.private.id
  vpc_security_group_ids = [aws_security_group.private_sg.id]
  iam_instance_profile   = aws_iam_instance_profile.ssm_profile.name

  # Enable encryption at rest
  root_block_device {
    volume_type           = "gp3"
    volume_size           = 20
    encrypted             = true
    delete_on_termination = true
  }

  user_data = local.user_data_script

  # Enable detailed monitoring
  monitoring = true

  tags = {
    Name        = "private-ec2"
    Environment = var.environment
  }
}

# Public EC2 Instance with encryption at rest
resource "aws_instance" "public" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.public_sg.id]

  # Enable encryption at rest
  root_block_device {
    volume_type           = "gp3"
    volume_size           = 20
    encrypted             = true
    delete_on_termination = true
  }

  user_data = <<-EOF
    ${local.user_data_script}
    # Curl the private instance and log the response
    curl http://${aws_instance.private.private_ip}:80 > /tmp/private_ip_response.log
  EOF

  # Enable detailed monitoring
  monitoring = true

  tags = {
    Name        = "public-ec2"
    Environment = var.environment
  }

  depends_on = [aws_instance.private] # Ensure private is created first
}