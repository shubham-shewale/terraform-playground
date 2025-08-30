package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestVpcCreation(t *testing.T) {
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

	// Test VPC creation
	vpcId := terraform.Output(t, terraformOptions, "vpc_id")
	assert.NotEmpty(t, vpcId)

	// Test VPC attributes
	vpcCidr := terraform.Output(t, terraformOptions, "vpc_cidr_block")
	assert.Equal(t, "10.0.0.0/16", vpcCidr)

	// Test DNS settings
	enableDnsSupport := terraform.Output(t, terraformOptions, "vpc_enable_dns_support")
	assert.Equal(t, "true", enableDnsSupport)

	enableDnsHostnames := terraform.Output(t, terraformOptions, "vpc_enable_dns_hostnames")
	assert.Equal(t, "true", enableDnsHostnames)
}

func TestVpcTagging(t *testing.T) {
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

	// Test VPC tags
	vpcTags := terraform.OutputMap(t, terraformOptions, "vpc_tags")
	assert.Equal(t, "basic-vpc", vpcTags["Name"])
	assert.Equal(t, "test", vpcTags["Environment"])
}

func TestVpcFlowLogs(t *testing.T) {
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

	// Test VPC Flow Logs creation
	flowLogId := terraform.Output(t, terraformOptions, "vpc_flow_log_id")
	assert.NotEmpty(t, flowLogId)

	// Test Flow Logs attributes
	flowLogTrafficType := terraform.Output(t, terraformOptions, "vpc_flow_log_traffic_type")
	assert.Equal(t, "ALL", flowLogTrafficType)

	// Test CloudWatch Log Group
	logGroupName := terraform.Output(t, terraformOptions, "vpc_flow_log_group_name")
	assert.Equal(t, "/aws/vpc/flowlogs", logGroupName)
}
