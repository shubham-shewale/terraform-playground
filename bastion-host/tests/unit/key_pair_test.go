package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestKeyPairModule(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../modules/key_pair",
		Vars: map[string]interface{}{
			"key_name":   "test-bastion-key",
			"public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ test@example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test key pair creation
	keyName := terraform.Output(t, terraformOptions, "key_name")
	assert.NotEmpty(t, keyName)
	assert.Equal(t, "test-bastion-key", keyName)
}

func TestKeyPairWithDifferentName(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../modules/key_pair",
		Vars: map[string]interface{}{
			"key_name":   "prod-bastion-key",
			"public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u1J4QJzB8K3nQqJc7fW4HqQ prod@example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test key pair creation with different name
	keyName := terraform.Output(t, terraformOptions, "key_name")
	assert.NotEmpty(t, keyName)
	assert.Equal(t, "prod-bastion-key", keyName)
}
