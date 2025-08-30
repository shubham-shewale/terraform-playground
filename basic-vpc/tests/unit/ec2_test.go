package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestEc2Instances(t *testing.T) {
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

	// Test Public EC2 Instance
	publicInstanceId := terraform.Output(t, terraformOptions, "public_instance_id")
	assert.NotEmpty(t, publicInstanceId)

	publicInstanceState := terraform.Output(t, terraformOptions, "public_instance_state")
	assert.Equal(t, "running", publicInstanceState)

	// Test Private EC2 Instance
	privateInstanceId := terraform.Output(t, terraformOptions, "private_instance_id")
	assert.NotEmpty(t, privateInstanceId)

	privateInstanceState := terraform.Output(t, terraformOptions, "private_instance_state")
	assert.Equal(t, "running", privateInstanceState)

	// Test instance types
	publicInstanceType := terraform.Output(t, terraformOptions, "public_instance_type")
	assert.Equal(t, "t3.micro", publicInstanceType)

	privateInstanceType := terraform.Output(t, terraformOptions, "private_instance_type")
	assert.Equal(t, "t3.micro", privateInstanceType)
}

func TestEc2Encryption(t *testing.T) {
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

	// Test EBS encryption
	publicEbsEncrypted := terraform.Output(t, terraformOptions, "public_ebs_encrypted")
	assert.Equal(t, "true", publicEbsEncrypted)

	privateEbsEncrypted := terraform.Output(t, terraformOptions, "private_ebs_encrypted")
	assert.Equal(t, "true", privateEbsEncrypted)

	// Test EBS volume type
	publicEbsVolumeType := terraform.Output(t, terraformOptions, "public_ebs_volume_type")
	assert.Equal(t, "gp3", publicEbsVolumeType)

	privateEbsVolumeType := terraform.Output(t, terraformOptions, "private_ebs_volume_type")
	assert.Equal(t, "gp3", privateEbsVolumeType)
}

func TestEc2Monitoring(t *testing.T) {
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

	// Test detailed monitoring
	publicMonitoring := terraform.Output(t, terraformOptions, "public_monitoring_enabled")
	assert.Equal(t, "true", publicMonitoring)

	privateMonitoring := terraform.Output(t, terraformOptions, "private_monitoring_enabled")
	assert.Equal(t, "true", privateMonitoring)
}

func TestEc2IamProfiles(t *testing.T) {
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

	// Test IAM instance profiles
	publicIamProfile := terraform.Output(t, terraformOptions, "public_iam_instance_profile")
	assert.Contains(t, publicIamProfile, "ssm-profile")

	privateIamProfile := terraform.Output(t, terraformOptions, "private_iam_instance_profile")
	assert.Contains(t, privateIamProfile, "ssm-profile")
}
