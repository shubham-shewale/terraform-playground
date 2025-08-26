output "vpc_id"                 { value = module.vpc.vpc_id }
output "public_subnet_ids"      { value = module.vpc.public_subnet_ids }
output "private_subnet_ids"     { value = module.vpc.private_subnet_ids }
output "security_group_id"      { value = module.security_group.security_group_id }
output "key_pair_name"          { value = module.key_pair.key_name }
output "bastion_public_ip"      { value = module.bastion.public_ip }
output "private_instance_ip"    { value = module.private_instance.private_ip }
