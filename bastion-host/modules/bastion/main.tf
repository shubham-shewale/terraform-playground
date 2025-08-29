resource "aws_instance" "this" {
  ami                         = var.ami
  instance_type               = "t2.micro"
  subnet_id                   = var.subnet_id
  key_name                    = var.key_name
  vpc_security_group_ids      = [var.security_group_id]
  associate_public_ip_address = true
  iam_instance_profile        = var.iam_instance_profile

  # Enable encryption at rest
  root_block_device {
    volume_type           = "gp3"
    volume_size           = 20
    encrypted             = true
    delete_on_termination = true
  }

  # Enable detailed monitoring
  monitoring = true

  # Security hardening user data
  user_data = <<-EOF
    #!/bin/bash
    # Security hardening for bastion host
    yum update -y
    
    # Install security tools
    yum install -y fail2ban
    
    # Configure fail2ban for SSH protection
    cat > /etc/fail2ban/jail.local << 'FAIL2BAN_EOF'
    [sshd]
    enabled = true
    port = ssh
    filter = sshd
    logpath = /var/log/secure
    maxretry = 3
    bantime = 3600
    findtime = 600
    FAIL2BAN_EOF
    
    # Start and enable fail2ban
    systemctl start fail2ban
    systemctl enable fail2ban
    
    # Disable root login
    sed -i 's/#PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
    
    # Restrict SSH to key-based authentication only
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
    
    # Restart SSH service
    systemctl restart sshd
  EOF

  tags = { 
    Name = "ssh_bastion"
    Environment = var.environment
  }
}

output "public_ip" { value = aws_instance.this.public_ip }
