resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name        = "basic-vpc"
    Environment = var.environment
  }
}

# Internet Gateway
resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name        = "basic-igw"
    Environment = var.environment
  }
}

# Public Subnet
resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnet_cidr
  availability_zone       = var.availability_zone
  map_public_ip_on_launch = true

  tags = {
    Name        = "public-subnet"
    Environment = var.environment
  }
}

# Private Subnet
resource "aws_subnet" "private" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.private_subnet_cidr
  availability_zone       = var.availability_zone
  map_public_ip_on_launch = false

  tags = {
    Name        = "private-subnet"
    Environment = var.environment
  }
}

# Elastic IP for NAT Gateway
resource "aws_eip" "nat" {
  domain = "vpc"

  tags = {
    Name        = "nat-eip"
    Environment = var.environment
  }
}

# NAT Gateway in Public Subnet
resource "aws_nat_gateway" "nat" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public.id

  tags = {
    Name        = "basic-nat"
    Environment = var.environment
  }

  depends_on = [aws_internet_gateway.igw]
}

# Public Route Table
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.igw.id
  }

  tags = {
    Name        = "public-rt"
    Environment = var.environment
  }
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

# Private Route Table
resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.nat.id
  }

  tags = {
    Name        = "private-rt"
    Environment = var.environment
  }
}

resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.private.id
  route_table_id = aws_route_table.private.id
}

# Security Group for Public EC2 (allow inbound HTTP from anywhere, outbound all)
resource "aws_security_group" "public_sg" {
  vpc_id = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]  # Allow public access; restrict to specific IPs in production
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

# Security Group for Private EC2 (allow inbound HTTP from public SG, outbound all)
resource "aws_security_group" "private_sg" {
  vpc_id = aws_vpc.main.id

  ingress {
    from_port       = 80
    to_port         = 80
    protocol        = "tcp"
    security_groups = [aws_security_group.public_sg.id]  # Allow from public instance
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

# User Data script for HTTP server (returns instance private IP)
locals {
  user_data_script = <<-EOF
    #!/bin/bash
    yum update -y
    yum install -y python3
    cat <<EOPY > /tmp/server.py
    import http.server
    import socketserver
    import urllib.request

    class Handler(http.server.SimpleHTTPRequestHandler):
        def do_GET(self):
            self.send_response(200)
            self.send_header('Content-type', 'text/plain')
            self.end_headers()
            with urllib.request.urlopen('http://169.254.169.254/latest/meta-data/local-ipv4') as response:
                ip = response.read().decode('utf-8')
            self.wfile.write(bytes(ip, 'utf-8'))

    with socketserver.TCPServer(("", 80), Handler) as httpd:
        httpd.serve_forever()
    EOPY
    nohup python3 /tmp/server.py &
  EOF
}

# Private EC2 Instance
resource "aws_instance" "private" {
  ami           = data.aws_ami.amazon_linux.id
  instance_type = "t3.micro"
  subnet_id     = aws_subnet.private.id
  vpc_security_group_ids = [aws_security_group.private_sg.id]

  user_data = local.user_data_script

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"  # CIS benchmark: Enforce IMDSv2
  }

  tags = {
    Name        = "private-ec2"
    Environment = var.environment
  }
}

# Public EC2 Instance (with curl to private in user data)
resource "aws_instance" "public" {
  ami           = data.aws_ami.amazon_linux.id
  instance_type = "t3.micro"
  subnet_id     = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.public_sg.id]

  user_data = <<-EOF
    ${local.user_data_script}
    # Curl the private instance and log the response
    curl http://${aws_instance.private.private_ip}:80 > /tmp/private_ip_response.log
  EOF

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"  # CIS benchmark: Enforce IMDSv2
  }

  tags = {
    Name        = "public-ec2"
    Environment = var.environment
  }

  depends_on = [aws_instance.private]  # Ensure private is created first
}