package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSecurityGroupIntegration(t *testing.T) {
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
	assert.Greater(t, len(publicSgIngressRules), 0, "Public SG should have ingress rules")

	privateSgIngressRules := terraform.OutputList(t, terraformOptions, "private_sg_ingress_rules")
	assert.Greater(t, len(privateSgIngressRules), 0, "Private SG should have ingress rules")
}

func TestSecurityGroupRulesValidation(t *testing.T) {
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

func TestNaclIntegration(t *testing.T) {
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
	assert.Greater(t, len(publicNaclIngressRules), 0, "Public NACL should have ingress rules")

	publicNaclEgressRules := terraform.OutputList(t, terraformOptions, "public_nacl_egress_rules")
	assert.Greater(t, len(publicNaclEgressRules), 0, "Public NACL should have egress rules")
}

func TestNaclSubnetAssociation(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test public subnet NACL association
	publicSubnetNaclId := terraform.Output(t, terraformOptions, "public_subnet_nacl_id")
	publicNaclId := terraform.Output(t, terraformOptions, "public_nacl_id")
	assert.Equal(t, publicNaclId, publicSubnetNaclId)

	// Test private subnet NACL association
	privateSubnetNaclId := terraform.Output(t, terraformOptions, "private_subnet_nacl_id")
	privateNaclId := terraform.Output(t, terraformOptions, "private_nacl_id")
	assert.Equal(t, privateNaclId, privateSubnetNaclId)
}

func TestIamRolesAndPolicies(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test SSM IAM Role
	ssmRoleName := terraform.Output(t, terraformOptions, "ssm_role_name")
	assert.Contains(t, ssmRoleName, "ssm-role-for-private-ec2")

	ssmRoleArn := terraform.Output(t, terraformOptions, "ssm_role_arn")
	assert.NotEmpty(t, ssmRoleArn)

	// Test VPC Flow Log IAM Role
	vpcFlowLogRoleName := terraform.Output(t, terraformOptions, "vpc_flow_log_role_name")
	assert.Contains(t, vpcFlowLogRoleName, "vpc-flow-log-role")

	vpcFlowLogRoleArn := terraform.Output(t, terraformOptions, "vpc_flow_log_role_arn")
	assert.NotEmpty(t, vpcFlowLogRoleArn)

	// Test attached policies
	ssmPolicyAttached := terraform.Output(t, terraformOptions, "ssm_policy_attached")
	assert.Equal(t, "true", ssmPolicyAttached)

	vpcFlowLogPolicyAttached := terraform.Output(t, terraformOptions, "vpc_flow_log_policy_attached")
	assert.Equal(t, "true", vpcFlowLogPolicyAttached)
}

func TestInstanceProfileAttachment(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test instance profile exists
	instanceProfileName := terraform.Output(t, terraformOptions, "ssm_instance_profile_name")
	assert.Contains(t, instanceProfileName, "ssm-profile-for-private-ec2")

	instanceProfileArn := terraform.Output(t, terraformOptions, "ssm_instance_profile_arn")
	assert.NotEmpty(t, instanceProfileArn)

	// Test instances have the instance profile attached
	publicInstanceProfile := terraform.Output(t, terraformOptions, "public_instance_iam_profile")
	assert.Contains(t, publicInstanceProfile, instanceProfileName)

	privateInstanceProfile := terraform.Output(t, terraformOptions, "private_instance_iam_profile")
	assert.Contains(t, privateInstanceProfile, instanceProfileName)
}
