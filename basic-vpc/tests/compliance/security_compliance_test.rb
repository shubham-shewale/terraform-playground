# Security Compliance Tests using InSpec

control 'ec2-instances-1.0' do
  impact 1.0
  title 'EC2 Instance Security Compliance'
  desc 'Ensure EC2 instances are configured according to security best practices'

  describe aws_ec2_instance(name: 'public-ec2') do
    it { should exist }
    it { should be_running }
    its('instance_type') { should eq 't3.micro' }
    it { should have_ebs_encryption_enabled }
    it { should have_detailed_monitoring_enabled }
  end

  describe aws_ec2_instance(name: 'private-ec2') do
    it { should exist }
    it { should be_running }
    its('instance_type') { should eq 't3.micro' }
    it { should have_ebs_encryption_enabled }
    it { should have_detailed_monitoring_enabled }
  end
end

control 'security-groups-1.0' do
  impact 1.0
  title 'Security Group Compliance'
  desc 'Ensure security groups follow principle of least privilege'

  describe aws_security_group(name: 'public-ec2-sg-test') do
    it { should exist }
    it { should allow_in(port: 80, protocol: 'tcp') }
    it { should_not allow_in(port: 80, protocol: 'tcp', cidr: '0.0.0.0/0') }
    it { should allow_out(port: 'all', protocol: 'all') }
  end

  describe aws_security_group(name: 'private-ec2-sg-test') do
    it { should exist }
    it { should allow_in_only_from_security_group('public-ec2-sg-test', port: 80, protocol: 'tcp') }
    it { should allow_out(port: 'all', protocol: 'all') }
  end
end

control 'network-acls-1.0' do
  impact 0.9
  title 'Network ACL Compliance'
  desc 'Ensure Network ACLs provide defense in depth'

  describe aws_network_acl(name: 'public-nacl') do
    it { should exist }
    it { should allow_in(port: 80, protocol: 'tcp') }
    it { should allow_in(port: 443, protocol: 'tcp') }
    it { should allow_in(port: 22, protocol: 'tcp') }
    it { should allow_in(port: 1024..65535, protocol: 'tcp') }
    it { should allow_out(port: 'all', protocol: 'all') }
  end

  describe aws_network_acl(name: 'private-nacl') do
    it { should exist }
    it { should allow_in(port: 80, protocol: 'tcp') }
    it { should allow_in(port: 22, protocol: 'tcp') }
    it { should allow_in(port: 443, protocol: 'tcp') }
    it { should allow_in(port: 1024..65535, protocol: 'tcp') }
    it { should allow_out(port: 'all', protocol: 'all') }
  end
end

control 'iam-roles-1.0' do
  impact 0.9
  title 'IAM Roles and Policies Compliance'
  desc 'Ensure IAM roles follow principle of least privilege'

  describe aws_iam_role('ssm-role-for-private-ec2') do
    it { should exist }
    it { should have_assume_role_policy }
    it { should be_attached_to_policy('AmazonSSMManagedInstanceCore') }
  end

  describe aws_iam_role('vpc-flow-log-role') do
    it { should exist }
    it { should have_assume_role_policy }
  end

  describe aws_iam_instance_profile('ssm-profile-for-private-ec2') do
    it { should exist }
    it { should have_role_attached('ssm-role-for-private-ec2') }
  end
end

control 'encryption-1.0' do
  impact 1.0
  title 'Encryption Compliance'
  desc 'Ensure all data is encrypted at rest and in transit'

  # Test EBS volumes are encrypted
  aws_ec2_instances.instance_ids.each do |instance_id|
    describe aws_ebs_volume(aws_ec2_instance(instance_id).block_device_mappings.first.volume_id) do
      it { should be_encrypted }
      its('encryption_type') { should eq 'AES256' }
    end
  end

  # Test S3 bucket encryption
  describe aws_s3_bucket('basic-vpc-cloudtrail-logs-*') do
    it { should exist }
    it { should have_default_encryption_enabled }
    its('encryption_type') { should eq 'AES256' }
  end
end

control 'vpc-endpoints-1.0' do
  impact 0.8
  title 'VPC Endpoints Compliance'
  desc 'Ensure VPC endpoints are properly configured for secure communication'

  describe aws_vpc_endpoint(service_name: 'com.amazonaws.us-east-1.ssm') do
    it { should exist }
    its('vpc_endpoint_type') { should eq 'Interface' }
    it { should have_private_dns_enabled }
  end

  describe aws_vpc_endpoint(service_name: 'com.amazonaws.us-east-1.ec2messages') do
    it { should exist }
    its('vpc_endpoint_type') { should eq 'Interface' }
    it { should have_private_dns_enabled }
  end

  describe aws_vpc_endpoint(service_name: 'com.amazonaws.us-east-1.ssmmessages') do
    it { should exist }
    its('vpc_endpoint_type') { should eq 'Interface' }
    it { should have_private_dns_enabled }
  end
end