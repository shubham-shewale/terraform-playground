package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSecurityGroups(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "test",
			"allowed_http_cidrs": []string{"203.0.113.0/24"},
			"allowed_ssh_cidrs":  []string{"203.0.113.0/24"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test Public Security Group
	publicSgId := terraform.Output(t, terraformOptions, "public_security_group_id")
	assert.NotEmpty(t, publicSgId)

	// Test Private Security Group
	privateSgId := terraform.Output(t, terraformOptions, "private_security_group_id")
	assert.NotEmpty(t, privateSgId)

	// Test security group rules
	publicSgIngressRules := terraform.OutputList(t, terraformOptions, "public_sg_ingress_rules")
	assert.Greater(t, len(publicSgIngressRules), 0)

	privateSgIngressRules := terraform.OutputList(t, terraformOptions, "private_sg_ingress_rules")
	assert.Greater(t, len(privateSgIngressRules), 0)
}

func TestSecurityGroupRules(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "test",
			"allowed_http_cidrs": []string{"203.0.113.0/24"},
			"allowed_ssh_cidrs":  []string{"203.0.113.0/24"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test HTTP access restriction
	publicSgHttpAllowed := terraform.Output(t, terraformOptions, "public_sg_http_from_allowed_cidrs")
	assert.Equal(t, "true", publicSgHttpAllowed)

	// Test that default unrestricted access is not allowed
	publicSgNoDefaultOpen := terraform.Output(t, terraformOptions, "public_sg_no_default_open")
	assert.Equal(t, "true", publicSgNoDefaultOpen)

	// Test private SG allows traffic from public SG
	privateSgAllowsPublic := terraform.Output(t, terraformOptions, "private_sg_allows_public_sg")
	assert.Equal(t, "true", privateSgAllowsPublic)
}

func TestNetworkACLs(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "test",
			"allowed_http_cidrs": []string{"203.0.113.0/24"},
			"allowed_ssh_cidrs":  []string{"203.0.113.0/24"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test Public NACL
	publicNaclId := terraform.Output(t, terraformOptions, "public_nacl_id")
	assert.NotEmpty(t, publicNaclId)

	// Test Private NACL
	privateNaclId := terraform.Output(t, terraformOptions, "private_nacl_id")
	assert.NotEmpty(t, privateNaclId)

	// Test NACL rules
	publicNaclIngressRules := terraform.OutputList(t, terraformOptions, "public_nacl_ingress_rules")
	assert.Greater(t, len(publicNaclIngressRules), 0)

	publicNaclEgressRules := terraform.OutputList(t, terraformOptions, "public_nacl_egress_rules")
	assert.Greater(t, len(publicNaclEgressRules), 0)
}

func TestNaclRules(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "test",
			"allowed_http_cidrs": []string{"203.0.113.0/24"},
			"allowed_ssh_cidrs":  []string{"203.0.113.0/24"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test HTTP rule allows specific CIDR
	publicNaclHttpAllowed := terraform.Output(t, terraformOptions, "public_nacl_http_from_allowed")
	assert.Equal(t, "true", publicNaclHttpAllowed)

	// Test SSH rule allows specific CIDR
	publicNaclSshAllowed := terraform.Output(t, terraformOptions, "public_nacl_ssh_from_allowed")
	assert.Equal(t, "true", publicNaclSshAllowed)

	// Test ephemeral ports are allowed for return traffic
	publicNaclEphemeralAllowed := terraform.Output(t, terraformOptions, "public_nacl_ephemeral_allowed")
	assert.Equal(t, "true", publicNaclEphemeralAllowed)

	// Test private NACL allows traffic from public subnet
	privateNaclAllowsPublicSubnet := terraform.Output(t, terraformOptions, "private_nacl_allows_public_subnet")
	assert.Equal(t, "true", privateNaclAllowsPublicSubnet)
}
