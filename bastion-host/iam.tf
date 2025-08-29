# IAM Role for Bastion Host
resource "aws_iam_role" "bastion_role" {
  name = "bastion-host-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "bastion-role"
    Environment = var.environment
  }
}

# IAM Policy for Bastion Host - SSM enabled
resource "aws_iam_role_policy" "bastion_policy" {
  name = "bastion-host-policy"
  role = aws_iam_role.bastion_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeInstances",
          "ec2:DescribeSecurityGroups",
          "ec2:DescribeVpcs",
          "ec2:DescribeSubnets"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:StartSession",
          "ssm:TerminateSession",
          "ssm:DescribeSessions",
          "ssm:GetConnectionStatus"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "ssmmessages:CreateControlChannel",
          "ssmmessages:CreateDataChannel",
          "ssmmessages:OpenControlChannel",
          "ssmmessages:OpenDataChannel"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "ec2messages:AcknowledgeMessage",
          "ec2messages:DeleteMessage",
          "ec2messages:FailMessage",
          "ec2messages:GetEndpoint",
          "ec2messages:GetMessages",
          "ec2messages:SendReply"
        ]
        Resource = "*"
      }
    ]
  })
}

# Instance Profile for Bastion Host
resource "aws_iam_instance_profile" "bastion_profile" {
  name = "bastion-host-profile"
  role = aws_iam_role.bastion_role.name
}

# CloudWatch Log Group for Bastion Host logs
resource "aws_cloudwatch_log_group" "bastion_logs" {
  name              = "/aws/bastion/ssh-logs"
  retention_in_days = 30

  tags = {
    Name        = "bastion-ssh-logs"
    Environment = var.environment
  }
}

# CloudWatch Alarm for SSH login attempts
resource "aws_cloudwatch_metric_alarm" "ssh_attempts" {
  alarm_name          = "ssh-login-attempts-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "SSHLoginAttempts"
  namespace           = "AWS/Bastion"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "Monitor SSH login attempts on bastion host"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  tags = {
    Name        = "ssh-attempts-alarm"
    Environment = var.environment
  }
}

# SNS Topic for Security Alerts
resource "aws_sns_topic" "security_alerts" {
  name = "bastion-security-alerts-${var.environment}"

  tags = {
    Name        = "bastion-security-alerts"
    Environment = var.environment
  }
}

# SNS Topic Policy
resource "aws_sns_topic_policy" "security_alerts" {
  arn = aws_sns_topic.security_alerts.arn

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "cloudwatch.amazonaws.com"
        }
        Action   = "SNS:Publish"
        Resource = aws_sns_topic.security_alerts.arn
      }
    ]
  })
}
