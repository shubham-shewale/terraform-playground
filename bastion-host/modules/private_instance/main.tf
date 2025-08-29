resource "aws_instance" "this" {
  ami                         = var.ami
  instance_type               = "t2.micro"
  subnet_id                   = var.subnet_id
  key_name                    = var.key_name
  vpc_security_group_ids      = [var.security_group_id]
  associate_public_ip_address = false

  # Enable encryption at rest
  root_block_device {
    volume_type           = "gp3"
    volume_size           = 20
    encrypted             = true
    delete_on_termination = true
  }

  # Enable detailed monitoring
  monitoring = true

  tags = { 
    Name = "private_instance"
    Environment = var.environment
  }
}

output "private_ip" { value = aws_instance.this.private_ip }
