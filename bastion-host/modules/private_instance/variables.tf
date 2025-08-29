variable "ami" { type = string }
variable "subnet_id" { type = string }
variable "key_name" { type = string }
variable "security_group_id" { type = string }
variable "environment" { 
  type = string
  default = "dev"
}
