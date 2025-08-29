variable "name" {
  type = string
}

resource "aws_cloudfront_response_headers_policy" "this" {
  name    = var.name
  comment = "Security headers for static website"

  security_headers_config {
    content_type_options {
      override = true
    }
    frame_options {
      frame_option = "DENY"
      override     = true
    }
    referrer_policy {
      referrer_policy = "strict-origin-when-cross-origin"
      override        = true
    }
    xss_protection {
      mode_block = true
      protection = true
      override   = true
    }
    strict_transport_security {
      access_control_max_age_sec = 31536000
      include_subdomains         = true
      override                   = true
    }
    content_security_policy {
      content_security_policy = "default-src 'self'; script-src 'self'; style-src 'self'"
      override                 = true
    }
  }
}

output "id" {
  value = aws_cloudfront_response_headers_policy.this.id
}

