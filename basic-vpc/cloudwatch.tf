# CloudWatch Alarms for Security Monitoring

# CPU Utilization Alarm
resource "aws_cloudwatch_metric_alarm" "cpu_utilization" {
  for_each = {
    public  = aws_instance.public.id
    private = aws_instance.private.id
  }

  alarm_name          = "cpu-utilization-${each.key}-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors EC2 CPU utilization"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  dimensions = {
    InstanceId = each.value
  }

  tags = {
    Name        = "cpu-alarm-${each.key}"
    Environment = var.environment
  }
}

# Network In Alarm
resource "aws_cloudwatch_metric_alarm" "network_in" {
  for_each = {
    public  = aws_instance.public.id
    private = aws_instance.private.id
  }

  alarm_name          = "network-in-${each.key}-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "NetworkIn"
  namespace           = "AWS/EC2"
  period              = "300"
  statistic           = "Average"
  threshold           = "1000000" # 1MB
  alarm_description   = "This metric monitors EC2 network inbound traffic"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  dimensions = {
    InstanceId = each.value
  }

  tags = {
    Name        = "network-in-alarm-${each.key}"
    Environment = var.environment
  }
}

# Status Check Alarm
resource "aws_cloudwatch_metric_alarm" "status_check" {
  for_each = {
    public  = aws_instance.public.id
    private = aws_instance.private.id
  }

  alarm_name          = "status-check-${each.key}-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "StatusCheckFailed"
  namespace           = "AWS/EC2"
  period              = "60"
  statistic           = "Maximum"
  threshold           = "0"
  alarm_description   = "This metric monitors EC2 status checks"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  dimensions = {
    InstanceId = each.value
  }

  tags = {
    Name        = "status-check-alarm-${each.key}"
    Environment = var.environment
  }
}

# SNS Topic for Security Alerts
resource "aws_sns_topic" "security_alerts" {
  name = "security-alerts-${var.environment}"

  tags = {
    Name        = "security-alerts"
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

# CloudWatch Dashboard for Security Monitoring
resource "aws_cloudwatch_dashboard" "security_dashboard" {
  dashboard_name = "security-dashboard-${var.environment}"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6

        properties = {
          metrics = [
            ["AWS/EC2", "CPUUtilization", "InstanceId", aws_instance.public.id],
            [".", ".", ".", aws_instance.private.id]
          ]
          period = 300
          stat   = "Average"
          region = "us-east-1"
          title  = "EC2 CPU Utilization"
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 0
        width  = 12
        height = 6

        properties = {
          metrics = [
            ["AWS/EC2", "NetworkIn", "InstanceId", aws_instance.public.id],
            [".", ".", ".", aws_instance.private.id]
          ]
          period = 300
          stat   = "Average"
          region = "us-east-1"
          title  = "EC2 Network In"
        }
      }
    ]
  })
}
