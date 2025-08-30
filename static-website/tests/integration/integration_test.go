package integration

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestStaticWebsiteIntegration(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "integration-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test that all components work together
	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")
	s3BucketName := terraform.Output(t, terraformOptions, "s3_bucket_name")
	wafACLArn := terraform.Output(t, terraformOptions, "waf_web_acl_arn")

	assert.NotEmpty(t, cloudfrontDomain)
	assert.NotEmpty(t, s3BucketName)
	assert.NotEmpty(t, wafACLArn)

	// Verify outputs are consistent
	assert.NotEqual(t, cloudfrontDomain, s3BucketName)
}
