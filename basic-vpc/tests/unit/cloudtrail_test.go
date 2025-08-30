package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestCloudTrail(t *testing.T) {
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

	// Test CloudTrail creation
	cloudtrailId := terraform.Output(t, terraformOptions, "cloudtrail_id")
	assert.NotEmpty(t, cloudtrailId)

	cloudtrailArn := terraform.Output(t, terraformOptions, "cloudtrail_arn")
	assert.NotEmpty(t, cloudtrailArn)
	assert.Contains(t, cloudtrailArn, "basic-vpc-cloudtrail")

	// Test CloudTrail configuration
	isMultiRegion := terraform.Output(t, terraformOptions, "cloudtrail_is_multi_region")
	assert.Equal(t, "true", isMultiRegion)

	isLoggingEnabled := terraform.Output(t, terraformOptions, "cloudtrail_logging_enabled")
	assert.Equal(t, "true", isLoggingEnabled)

	includeGlobalEvents := terraform.Output(t, terraformOptions, "cloudtrail_include_global_events")
	assert.Equal(t, "true", includeGlobalEvents)
}

func TestCloudTrailS3Bucket(t *testing.T) {
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

	// Test S3 bucket creation
	bucketId := terraform.Output(t, terraformOptions, "cloudtrail_bucket_id")
	assert.NotEmpty(t, bucketId)
	assert.Contains(t, bucketId, "basic-vpc-cloudtrail-logs")

	// Test bucket versioning
	bucketVersioningEnabled := terraform.Output(t, terraformOptions, "cloudtrail_bucket_versioning")
	assert.Equal(t, "Enabled", bucketVersioningEnabled)

	// Test server-side encryption
	bucketEncryptionAlgorithm := terraform.Output(t, terraformOptions, "cloudtrail_bucket_encryption")
	assert.Equal(t, "AES256", bucketEncryptionAlgorithm)
}

func TestCloudTrailS3Security(t *testing.T) {
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

	// Test public access blocks
	blockPublicAcls := terraform.Output(t, terraformOptions, "cloudtrail_bucket_block_public_acls")
	assert.Equal(t, "true", blockPublicAcls)

	blockPublicPolicy := terraform.Output(t, terraformOptions, "cloudtrail_bucket_block_public_policy")
	assert.Equal(t, "true", blockPublicPolicy)

	ignorePublicAcls := terraform.Output(t, terraformOptions, "cloudtrail_bucket_ignore_public_acls")
	assert.Equal(t, "true", ignorePublicAcls)

	restrictPublicBuckets := terraform.Output(t, terraformOptions, "cloudtrail_bucket_restrict_public_buckets")
	assert.Equal(t, "true", restrictPublicBuckets)
}

func TestCloudTrailBucketPolicy(t *testing.T) {
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

	// Test bucket policy allows CloudTrail access
	bucketPolicyAllowsCloudTrail := terraform.Output(t, terraformOptions, "bucket_policy_allows_cloudtrail")
	assert.Equal(t, "true", bucketPolicyAllowsCloudTrail)

	// Test bucket policy allows CloudTrail to get bucket ACL
	bucketPolicyAllowsAclCheck := terraform.Output(t, terraformOptions, "bucket_policy_allows_acl_check")
	assert.Equal(t, "true", bucketPolicyAllowsAclCheck)

	// Test bucket policy allows CloudTrail to put objects
	bucketPolicyAllowsPutObject := terraform.Output(t, terraformOptions, "bucket_policy_allows_put_object")
	assert.Equal(t, "true", bucketPolicyAllowsPutObject)
}

func TestCloudTrailEventSelectors(t *testing.T) {
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

	// Test event selector configuration
	readWriteType := terraform.Output(t, terraformOptions, "cloudtrail_read_write_type")
	assert.Equal(t, "All", readWriteType)

	includeManagementEvents := terraform.Output(t, terraformOptions, "cloudtrail_include_management_events")
	assert.Equal(t, "true", includeManagementEvents)

	// Test data resource logging
	dataResourceType := terraform.Output(t, terraformOptions, "cloudtrail_data_resource_type")
	assert.Equal(t, "AWS::S3::Object", dataResourceType)

	dataResourceValues := terraform.OutputList(t, terraformOptions, "cloudtrail_data_resource_values")
	assert.Greater(t, len(dataResourceValues), 0)
	assert.Contains(t, dataResourceValues[0], "/*")
}
