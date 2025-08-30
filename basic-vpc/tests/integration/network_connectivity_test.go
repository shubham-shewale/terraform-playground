package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestNetworkConnectivity(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "test",
			"allowed_http_cidrs": []string{"0.0.0.0/0"}, // Allow all for testing
			"allowed_ssh_cidrs":  []string{"0.0.0.0/0"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test that instances are created
	publicInstanceId := terraform.Output(t, terraformOptions, "public_instance_id")
	assert.NotEmpty(t, publicInstanceId)

	privateInstanceId := terraform.Output(t, terraformOptions, "private_instance_id")
	assert.NotEmpty(t, privateInstanceId)

	// Test instance states from Terraform outputs
	publicInstanceState := terraform.Output(t, terraformOptions, "public_instance_state")
	assert.Equal(t, "running", publicInstanceState)

	privateInstanceState := terraform.Output(t, terraformOptions, "private_instance_state")
	assert.Equal(t, "running", privateInstanceState)
}

func TestVpcSubnetConfiguration(t *testing.T) {
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

	// Test VPC configuration
	vpcId := terraform.Output(t, terraformOptions, "vpc_id")
	assert.NotEmpty(t, vpcId)

	vpcCidr := terraform.Output(t, terraformOptions, "vpc_cidr_block")
	assert.Equal(t, "10.0.0.0/16", vpcCidr)

	// Test subnets exist
	publicSubnetId := terraform.Output(t, terraformOptions, "public_subnet_id")
	assert.NotEmpty(t, publicSubnetId)

	privateSubnetId := terraform.Output(t, terraformOptions, "private_subnet_id")
	assert.NotEmpty(t, privateSubnetId)

	// Test subnet CIDRs
	publicSubnetCidr := terraform.Output(t, terraformOptions, "public_subnet_cidr")
	assert.Equal(t, "10.0.1.0/24", publicSubnetCidr)

	privateSubnetCidr := terraform.Output(t, terraformOptions, "private_subnet_cidr")
	assert.Equal(t, "10.0.2.0/24", privateSubnetCidr)
}

func TestInternetGatewayAndNatGateway(t *testing.T) {
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

	// Test Internet Gateway
	igwId := terraform.Output(t, terraformOptions, "internet_gateway_id")
	assert.NotEmpty(t, igwId)

	// Test NAT Gateway
	natId := terraform.Output(t, terraformOptions, "nat_gateway_id")
	assert.NotEmpty(t, natId)

	natState := terraform.Output(t, terraformOptions, "nat_gateway_state")
	assert.Equal(t, "available", natState)

	// Test NAT Gateway is in public subnet
	natSubnetId := terraform.Output(t, terraformOptions, "nat_gateway_subnet_id")
	publicSubnetId := terraform.Output(t, terraformOptions, "public_subnet_id")
	assert.Equal(t, publicSubnetId, natSubnetId)
}

func TestRouteTables(t *testing.T) {
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

	// Test Public Route Table
	publicRtId := terraform.Output(t, terraformOptions, "public_route_table_id")
	assert.NotEmpty(t, publicRtId)

	// Test Private Route Table
	privateRtId := terraform.Output(t, terraformOptions, "private_route_table_id")
	assert.NotEmpty(t, privateRtId)

	// Test route table associations
	publicSubnetRtId := terraform.Output(t, terraformOptions, "public_subnet_route_table_id")
	assert.Equal(t, publicRtId, publicSubnetRtId)

	privateSubnetRtId := terraform.Output(t, terraformOptions, "private_subnet_route_table_id")
	assert.Equal(t, privateRtId, privateSubnetRtId)
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

	// Test VPC Flow Logs are enabled
	flowLogId := terraform.Output(t, terraformOptions, "vpc_flow_log_id")
	assert.NotEmpty(t, flowLogId)

	flowLogTrafficType := terraform.Output(t, terraformOptions, "vpc_flow_log_traffic_type")
	assert.Equal(t, "ALL", flowLogTrafficType)

	// Test CloudWatch Log Group
	logGroupName := terraform.Output(t, terraformOptions, "vpc_flow_log_group_name")
	assert.Equal(t, "/aws/vpc/flowlogs", logGroupName)

	logGroupRetention := terraform.Output(t, terraformOptions, "vpc_flow_log_retention_days")
	assert.Equal(t, "30", logGroupRetention)
}
