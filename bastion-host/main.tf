terraform {
  backend "s3" {
    bucket = "my-terraform-state-bucket-211125380337"
    key    = "terraform-playground-bastion-host.tfstate"
    region = "us-east-1"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

data "aws_ami" "amazon_linux" {
  most_recent = true

  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-*-x86_64-gp2"]
  }

  owners = ["137112412989"] # Amazon
}

module "vpc" {
  source               = "./modules/vpc"
  cidr_block           = var.vpc_cidr
  azs                  = var.azs
  public_subnet_cidrs  = var.public_subnet_cidrs
  private_subnet_cidrs = var.private_subnet_cidrs
}

module "security_group" {
  source                = "./modules/security_group"
  vpc_id                = module.vpc.vpc_id
  allowed_ssh_cidrs     = var.allowed_ssh_cidrs
  private_subnet_cidrs  = var.private_subnet_cidrs
  environment           = var.environment
}

module "key_pair" {
  source     = "./modules/key_pair"
  key_name   = var.key_name
  public_key = var.public_key
}

module "bastion" {
  source                = "./modules/bastion"
  subnet_id             = module.vpc.public_subnet_ids[0]
  key_name              = module.key_pair.key_name
  security_group_id     = module.security_group.security_group_id
  ami                   = data.aws_ami.amazon_linux.id
  environment           = var.environment
  iam_instance_profile  = aws_iam_instance_profile.bastion_profile.name
}

module "private_instance" {
  source            = "./modules/private_instance"
  subnet_id         = module.vpc.private_subnet_ids[0]
  key_name          = module.key_pair.key_name
  security_group_id = module.security_group.security_group_id
  ami               = data.aws_ami.amazon_linux.id
  environment       = var.environment
}
