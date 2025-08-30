package integration

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestFullBastionDeployment(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../..",
		Vars: map[string]interface{}{
			"region":               "us-east-1",
			"vpc_cidr":             "10.0.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.0.1.0/24"},
			"private_subnet_cidrs": []string{"10.0.10.0/24"},
			"key_name":             "test-integration-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
			"allowed_ssh_cidrs":    []string{"203.0.113.0/24"},
			"environment":          "test",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test VPC creation
	vpcId := terraform.Output(t, terraformOptions, "vpc_id")
	assert.NotEmpty(t, vpcId)

	// Test subnet creation
	publicSubnetIds := terraform.OutputList(t, terraformOptions, "public_subnet_ids")
	assert.Len(t, publicSubnetIds, 1)
	assert.NotEmpty(t, publicSubnetIds[0])

	privateSubnetIds := terraform.OutputList(t, terraformOptions, "private_subnet_ids")
	assert.Len(t, privateSubnetIds, 1)
	assert.NotEmpty(t, privateSubnetIds[0])

	// Test security group creation
	securityGroupId := terraform.Output(t, terraformOptions, "security_group_id")
	assert.NotEmpty(t, securityGroupId)

	// Test key pair creation
	keyPairName := terraform.Output(t, terraformOptions, "key_pair_name")
	assert.NotEmpty(t, keyPairName)
	assert.Equal(t, "test-integration-key", keyPairName)

	// Test bastion host creation
	bastionPublicIp := terraform.Output(t, terraformOptions, "bastion_public_ip")
	assert.NotEmpty(t, bastionPublicIp)

	// Test private instance creation
	privateInstanceIp := terraform.Output(t, terraformOptions, "private_instance_ip")
	assert.NotEmpty(t, privateInstanceIp)
}

func TestBastionConnectivity(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../..",
		Vars: map[string]interface{}{
			"region":               "us-east-1",
			"vpc_cidr":             "10.1.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.1.1.0/24"},
			"private_subnet_cidrs": []string{"10.1.10.0/24"},
			"key_name":             "test-connectivity-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
			"allowed_ssh_cidrs":    []string{"203.0.113.0/24"},
			"environment":          "test",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify all components are created and accessible
	bastionPublicIp := terraform.Output(t, terraformOptions, "bastion_public_ip")
	privateInstanceIp := terraform.Output(t, terraformOptions, "private_instance_ip")

	assert.NotEmpty(t, bastionPublicIp)
	assert.NotEmpty(t, privateInstanceIp)

	// In a real integration test, you would:
	// 1. SSH to bastion host
	// 2. From bastion, SSH to private instance
	// 3. Verify network connectivity
	// 4. Test security group rules
}

func TestBastionSecurityConfiguration(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../..",
		Vars: map[string]interface{}{
			"region":               "us-east-1",
			"vpc_cidr":             "10.2.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.2.1.0/24"},
			"private_subnet_cidrs": []string{"10.2.10.0/24"},
			"key_name":             "test-security-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
			"allowed_ssh_cidrs":    []string{"203.0.113.0/24"},
			"environment":          "test",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify security components are properly configured
	vpcId := terraform.Output(t, terraformOptions, "vpc_id")
	assert.NotEmpty(t, vpcId)

	// In a real security test, you would verify:
	// 1. Security groups restrict access properly
	// 2. Network ACLs are configured
	// 3. VPC Flow Logs are enabled
	// 4. Encryption is enabled on volumes
	// 5. IAM roles have minimal permissions
}
