resource "aws_security_group" "this" {
  name        = "default_security_group"
  description = "Ingress: 22 only; Egress: all"
  vpc_id      = var.vpc_id

  ingress {
    description = "SSH from allowed IPs"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]   # Replace with corporate IP range
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "default_security_group" }
}

output "security_group_id" { value = aws_security_group.this.id }
