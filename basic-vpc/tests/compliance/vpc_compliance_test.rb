# VPC Compliance Tests using InSpec

control 'vpc-1.0' do
  impact 1.0
  title 'VPC Configuration Compliance'
  desc 'Ensure VPC is configured according to security best practices'

  describe aws_vpc('basic-vpc') do
    it { should exist }
    its('cidr_block') { should eq '10.0.0.0/16' }
    its('instance_tenancy') { should eq 'default' }
    it { should be_enable_dns_support }
    it { should be_enable_dns_hostnames }
  end
end

control 'vpc-flow-logs-1.0' do
  impact 1.0
  title 'VPC Flow Logs Compliance'
  desc 'Ensure VPC Flow Logs are properly configured for network monitoring'

  describe aws_vpc('basic-vpc') do
    it { should have_flow_logs_enabled }
  end

  describe aws_cloudwatch_log_group('/aws/vpc/flowlogs') do
    it { should exist }
    its('retention_in_days') { should eq 30 }
  end
end

control 'subnets-1.0' do
  impact 0.8
  title 'Subnet Configuration Compliance'
  desc 'Ensure subnets are properly configured with correct CIDR blocks'

  describe aws_subnet(name: 'public-subnet') do
    it { should exist }
    its('cidr_block') { should eq '10.0.1.0/24' }
    its('availability_zone') { should match /us-east-1[a-d]/ }
    it { should be_map_public_ip_on_launch }
  end

  describe aws_subnet(name: 'private-subnet') do
    it { should exist }
    its('cidr_block') { should eq '10.0.2.0/24' }
    its('availability_zone') { should match /us-east-1[a-d]/ }
    it { should_not be_map_public_ip_on_launch }
  end
end

control 'internet-gateway-1.0' do
  impact 0.7
  title 'Internet Gateway Compliance'
  desc 'Ensure Internet Gateway is properly attached to VPC'

  describe aws_internet_gateway do
    it { should exist }
    it { should be_attached_to_vpc('basic-vpc') }
  end
end

control 'nat-gateway-1.0' do
  impact 0.7
  title 'NAT Gateway Compliance'
  desc 'Ensure NAT Gateway is properly configured in public subnet'

  describe aws_nat_gateway do
    it { should exist }
    its('state') { should eq 'available' }
    it { should be_in_subnet('public-subnet') }
  end
end

control 'route-tables-1.0' do
  impact 0.8
  title 'Route Table Compliance'
  desc 'Ensure route tables have correct routing configuration'

  describe aws_route_table(name: 'public-rt') do
    it { should exist }
    it { should have_route('0.0.0.0/0').target.gateway('basic-igw') }
  end

  describe aws_route_table(name: 'private-rt') do
    it { should exist }
    it { should have_route('0.0.0.0/0').target.nat_gateway }
  end
end