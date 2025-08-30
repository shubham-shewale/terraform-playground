package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSsmRole(t *testing.T) {
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

	// Test SSM IAM Role creation
	ssmRoleName := terraform.Output(t, terraformOptions, "ssm_role_name")
	assert.Contains(t, ssmRoleName, "ssm-role-for-private-ec2")

	ssmRoleArn := terraform.Output(t, terraformOptions, "ssm_role_arn")
	assert.NotEmpty(t, ssmRoleArn)
	assert.Contains(t, ssmRoleArn, "ssm-role-for-private-ec2")

	// Test assume role policy
	ssmRoleAssumePolicy := terraform.Output(t, terraformOptions, "ssm_role_assume_policy")
	assert.Contains(t, ssmRoleAssumePolicy, "ec2.amazonaws.com")
}

func TestSsmPolicyAttachment(t *testing.T) {
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

	// Test SSM managed policy attachment
	ssmPolicyAttached := terraform.Output(t, terraformOptions, "ssm_policy_attached")
	assert.Equal(t, "true", ssmPolicyAttached)

	ssmPolicyArn := terraform.Output(t, terraformOptions, "ssm_policy_arn")
	assert.Equal(t, "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore", ssmPolicyArn)
}

func TestSsmInstanceProfile(t *testing.T) {
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

	// Test instance profile creation
	instanceProfileName := terraform.Output(t, terraformOptions, "ssm_instance_profile_name")
	assert.Contains(t, instanceProfileName, "ssm-profile-for-private-ec2")

	instanceProfileArn := terraform.Output(t, terraformOptions, "ssm_instance_profile_arn")
	assert.NotEmpty(t, instanceProfileArn)

	// Test instance profile role
	instanceProfileRole := terraform.Output(t, terraformOptions, "ssm_instance_profile_role")
	assert.Contains(t, instanceProfileRole, "ssm-role-for-private-ec2")
}

func TestVpcEndpoints(t *testing.T) {
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

	// Test SSM VPC Endpoint
	ssmEndpointId := terraform.Output(t, terraformOptions, "ssm_endpoint_id")
	assert.NotEmpty(t, ssmEndpointId)

	ssmEndpointServiceName := terraform.Output(t, terraformOptions, "ssm_endpoint_service_name")
	assert.Contains(t, ssmEndpointServiceName, "ssm")

	// Test EC2 Messages VPC Endpoint
	ec2messagesEndpointId := terraform.Output(t, terraformOptions, "ec2messages_endpoint_id")
	assert.NotEmpty(t, ec2messagesEndpointId)

	// Test SSM Messages VPC Endpoint
	ssmmessagesEndpointId := terraform.Output(t, terraformOptions, "ssmmessages_endpoint_id")
	assert.NotEmpty(t, ssmmessagesEndpointId)
}

func TestVpcEndpointConfiguration(t *testing.T) {
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

	// Test VPC endpoint type
	endpointType := terraform.Output(t, terraformOptions, "vpc_endpoint_type")
	assert.Equal(t, "Interface", endpointType)

	// Test private DNS enabled
	privateDnsEnabled := terraform.Output(t, terraformOptions, "vpc_endpoint_private_dns_enabled")
	assert.Equal(t, "true", privateDnsEnabled)

	// Test VPC endpoint subnet
	endpointSubnetId := terraform.Output(t, terraformOptions, "vpc_endpoint_subnet_id")
	assert.NotEmpty(t, endpointSubnetId)

	// Test VPC endpoint security group
	endpointSecurityGroupId := terraform.Output(t, terraformOptions, "vpc_endpoint_security_group_id")
	assert.NotEmpty(t, endpointSecurityGroupId)
}

func TestVpcEndpointSecurityGroup(t *testing.T) {
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

	// Test VPC endpoint security group allows HTTPS
	endpointSgAllowsHttps := terraform.Output(t, terraformOptions, "endpoint_sg_allows_https")
	assert.Equal(t, "true", endpointSgAllowsHttps)

	// Test VPC endpoint security group allows traffic from private SG
	endpointSgAllowsPrivateSg := terraform.Output(t, terraformOptions, "endpoint_sg_allows_private_sg")
	assert.Equal(t, "true", endpointSgAllowsPrivateSg)

	// Test VPC endpoint security group name
	endpointSgName := terraform.Output(t, terraformOptions, "endpoint_sg_name")
	assert.Contains(t, endpointSgName, "vpc-endpoint-sg")
}
