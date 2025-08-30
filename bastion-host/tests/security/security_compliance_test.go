package security

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSecurityGroupsCompliance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../..",
		Vars: map[string]interface{}{
			"region":               "us-east-1",
			"vpc_cidr":             "10.3.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.3.1.0/24"},
			"private_subnet_cidrs": []string{"10.3.10.0/24"},
			"key_name":             "test-security-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
			"allowed_ssh_cidrs":    []string{"203.0.113.0/24"},
			"environment":          "test",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify security groups exist
	securityGroupId := terraform.Output(t, terraformOptions, "security_group_id")
	assert.NotEmpty(t, securityGroupId)

	// In a real compliance test, you would verify:
	// 1. Security groups don't allow unrestricted access (0.0.0.0/0 for SSH)
	// 2. Private instances only accept SSH from bastion security group
	// 3. HTTPS access is properly restricted
}

func TestEncryptionCompliance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../..",
		Vars: map[string]interface{}{
			"region":               "us-east-1",
			"vpc_cidr":             "10.4.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.4.1.0/24"},
			"private_subnet_cidrs": []string{"10.4.10.0/24"},
			"key_name":             "test-encryption-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
			"allowed_ssh_cidrs":    []string{"203.0.113.0/24"},
			"environment":          "test",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify instances are created
	bastionPublicIp := terraform.Output(t, terraformOptions, "bastion_public_ip")
	privateInstanceIp := terraform.Output(t, terraformOptions, "private_instance_ip")

	assert.NotEmpty(t, bastionPublicIp)
	assert.NotEmpty(t, privateInstanceIp)

	// In a real compliance test, you would verify:
	// 1. EBS volumes are encrypted
	// 2. CloudTrail is enabled
	// 3. VPC Flow Logs are enabled
	// 4. S3 buckets have encryption enabled
}

func TestNetworkSecurityCompliance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../..",
		Vars: map[string]interface{}{
			"region":               "us-east-1",
			"vpc_cidr":             "10.5.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.5.1.0/24"},
			"private_subnet_cidrs": []string{"10.5.10.0/24"},
			"key_name":             "test-network-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
			"allowed_ssh_cidrs":    []string{"203.0.113.0/24"},
			"environment":          "test",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify network components
	vpcId := terraform.Output(t, terraformOptions, "vpc_id")
	publicSubnetIds := terraform.OutputList(t, terraformOptions, "public_subnet_ids")
	privateSubnetIds := terraform.OutputList(t, terraformOptions, "private_subnet_ids")

	assert.NotEmpty(t, vpcId)
	assert.Len(t, publicSubnetIds, 1)
	assert.Len(t, privateSubnetIds, 1)

	// In a real compliance test, you would verify:
	// 1. Network ACLs are properly configured
	// 2. VPC endpoints are created for SSM
	// 3. No public IPs assigned to private instances
	// 4. Security groups follow least privilege principle
}

func TestMonitoringCompliance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../..",
		Vars: map[string]interface{}{
			"region":               "us-east-1",
			"vpc_cidr":             "10.6.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.6.1.0/24"},
			"private_subnet_cidrs": []string{"10.6.10.0/24"},
			"key_name":             "test-monitoring-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
			"allowed_ssh_cidrs":    []string{"203.0.113.0/24"},
			"environment":          "test",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify monitoring components are created
	bastionPublicIp := terraform.Output(t, terraformOptions, "bastion_public_ip")
	assert.NotEmpty(t, bastionPublicIp)

	// In a real compliance test, you would verify:
	// 1. CloudWatch alarms are configured
	// 2. CloudTrail is enabled
	// 3. VPC Flow Logs are enabled
	// 4. SNS topics are configured for alerts
	// 5. Detailed monitoring is enabled on instances
}

func TestAccessControlCompliance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../..",
		Vars: map[string]interface{}{
			"region":               "us-east-1",
			"vpc_cidr":             "10.7.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.7.1.0/24"},
			"private_subnet_cidrs": []string{"10.7.10.0/24"},
			"key_name":             "test-access-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
			"allowed_ssh_cidrs":    []string{"203.0.113.0/24"},
			"environment":          "test",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify access control components
	keyPairName := terraform.Output(t, terraformOptions, "key_pair_name")
	assert.NotEmpty(t, keyPairName)

	// In a real compliance test, you would verify:
	// 1. SSH keys are properly configured
	// 2. IAM roles have minimal required permissions
	// 3. Root login is disabled
	// 4. Password authentication is disabled
	// 5. Fail2ban is configured
}
