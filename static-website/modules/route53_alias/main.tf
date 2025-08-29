variable "domain_name" { type = string }
variable "distribution_domain_name" { type = string }
variable "distribution_hosted_zone_id" { type = string }

data "aws_route53_zone" "this" { name = var.domain_name }

resource "aws_route53_record" "alias" {
  zone_id = data.aws_route53_zone.this.zone_id
  name    = var.domain_name
  type    = "A"
  alias {
    name                   = var.distribution_domain_name
    zone_id                = var.distribution_hosted_zone_id
    evaluate_target_health = false
  }
}

output "fqdn" { value = aws_route53_record.alias.fqdn }

