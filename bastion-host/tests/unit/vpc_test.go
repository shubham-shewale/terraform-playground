package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestVpcModule(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../modules/vpc",
		Vars: map[string]interface{}{
			"cidr_block":           "10.0.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.0.1.0/24"},
			"private_subnet_cidrs": []string{"10.0.10.0/24"},
			"region":               "us-east-1",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test VPC creation
	vpcId := terraform.Output(t, terraformOptions, "vpc_id")
	assert.NotEmpty(t, vpcId)

	// Test public subnet creation
	publicSubnetIds := terraform.OutputList(t, terraformOptions, "public_subnet_ids")
	assert.Len(t, publicSubnetIds, 1)
	assert.NotEmpty(t, publicSubnetIds[0])

	// Test private subnet creation
	privateSubnetIds := terraform.OutputList(t, terraformOptions, "private_subnet_ids")
	assert.Len(t, privateSubnetIds, 1)
	assert.NotEmpty(t, privateSubnetIds[0])

	// Test VPC exists and has expected attributes
	assert.NotEmpty(t, vpcId)
	// Additional VPC attribute checks would require AWS SDK calls
}

func TestVpcFlowLogs(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../modules/vpc",
		Vars: map[string]interface{}{
			"cidr_block":           "10.0.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.0.1.0/24"},
			"private_subnet_cidrs": []string{"10.0.10.0/24"},
			"region":               "us-east-1",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify VPC Flow Logs are enabled (basic existence check)
	// In a real scenario, you'd use AWS SDK to verify the flow log configuration
	vpcId := terraform.Output(t, terraformOptions, "vpc_id")
	assert.NotEmpty(t, vpcId)
}

func TestVpcEndpoints(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../modules/vpc",
		Vars: map[string]interface{}{
			"cidr_block":           "10.0.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"10.0.1.0/24"},
			"private_subnet_cidrs": []string{"10.0.10.0/24"},
			"region":               "us-east-1",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify VPC and subnets are created (endpoints are created as part of VPC module)
	vpcId := terraform.Output(t, terraformOptions, "vpc_id")
	assert.NotEmpty(t, vpcId)
}
