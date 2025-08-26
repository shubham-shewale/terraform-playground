resource "aws_instance" "this" {
  ami                    = var.ami
  instance_type          = "t2.micro"
  subnet_id              = var.subnet_id
  key_name               = var.key_name
  vpc_security_group_ids = [var.security_group_id]
  associate_public_ip_address = false

  tags = { Name = "private_instance" }
}

output "private_ip" { value = aws_instance.this.private_ip }
