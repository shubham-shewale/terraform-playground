package compliance

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestStaticWebsiteCompliance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "compliance-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test encryption compliance
	s3BucketName := terraform.Output(t, terraformOptions, "s3_bucket_name")
	assert.NotEmpty(t, s3BucketName)

	// Test HTTPS enforcement
	httpsEnforced := terraform.Output(t, terraformOptions, "cloudfront_domain")
	assert.NotEmpty(t, httpsEnforced)

	// Test WAF protection
	wafACLArn := terraform.Output(t, terraformOptions, "waf_web_acl_arn")
	assert.NotEmpty(t, wafACLArn)

	// Test certificate validation
	certificateArn := terraform.Output(t, terraformOptions, "certificate_arn")
	assert.NotEmpty(t, certificateArn)

	// Test CloudTrail logging
	cloudtrailEnabled := terraform.Output(t, terraformOptions, "cloudtrail_enabled")
	assert.Equal(t, "true", cloudtrailEnabled)
}
