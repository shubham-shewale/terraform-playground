package cost

import (
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudFrontCostOptimization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "cost-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get CloudFront distribution details
	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")
	priceClass := terraform.Output(t, terraformOptions, "cloudfront_price_class")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test 1: Verify cost-effective price class
	t.Log("Testing CloudFront price class optimization...")

	// Price class should be cost-effective (not all edge locations)
	assert.Equal(t, "PriceClass_100", priceClass, "Should use cost-effective price class")

	// Test 2: Monitor data transfer costs
	t.Log("Monitoring CloudFront data transfer costs...")

	dataTransferMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/CloudFront"),
		MetricName: aws.String("BytesDownloaded"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("DistributionId"),
				Value: aws.String(distributionID),
			},
			{
				Name:  aws.String("Region"),
				Value: aws.String("Global"),
			},
		},
		StartTime:  aws.Time(time.Now().Add(-1 * time.Hour)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int64(1800),
		Statistics: []*string{aws.String("Sum")},
	})

	require.NoError(t, err)

	if len(dataTransferMetrics.Datapoints) > 0 {
		totalBytes := 0.0
		for _, datapoint := range dataTransferMetrics.Datapoints {
			if datapoint.Sum != nil {
				totalBytes += *datapoint.Sum
			}
		}

		// Convert bytes to GB
		totalGB := totalBytes / (1024 * 1024 * 1024)
		t.Logf("CloudFront data transfer in last hour: %.2f GB", totalGB)

		// Assert reasonable data transfer for cost optimization
		assert.Less(t, totalGB, float64(1), "Data transfer should be minimal for cost optimization")
	}

	// Test 3: Verify Origin Shield is enabled for cost reduction
	t.Log("Verifying Origin Shield configuration for cost optimization...")

	originShieldEnabled := terraform.Output(t, terraformOptions, "origin_shield_enabled")
	originShieldRegion := terraform.Output(t, terraformOptions, "origin_shield_region")

	assert.Equal(t, "true", originShieldEnabled, "Origin Shield should be enabled for cost optimization")
	assert.NotEmpty(t, originShieldRegion, "Origin Shield region should be configured")
}

func TestWAFCostOptimization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "cost-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get WAF details
	wafACLArn := terraform.Output(t, terraformOptions, "waf_web_acl_arn")
	rateLimit := terraform.Output(t, terraformOptions, "waf_rate_limit")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test 1: Verify reasonable rate limiting
	t.Log("Testing WAF rate limiting for cost optimization...")

	// Rate limit should be reasonable to avoid excessive costs
	assert.Equal(t, "2000", rateLimit, "Rate limit should be reasonable for cost optimization")

	// Test 2: Monitor WAF request volume
	t.Log("Monitoring WAF request volume for cost analysis...")

	wafMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/WAFV2"),
		MetricName: aws.String("AllowedRequests"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("WebACL"),
				Value: aws.String(extractWAFNameFromArn(wafACLArn)),
			},
			{
				Name:  aws.String("Region"),
				Value: aws.String("us-east-1"),
			},
		},
		StartTime:  aws.Time(time.Now().Add(-1 * time.Hour)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int64(1800),
		Statistics: []*string{aws.String("Sum")},
	})

	require.NoError(t, err)

	if len(wafMetrics.Datapoints) > 0 {
		totalRequests := 0.0
		for _, datapoint := range wafMetrics.Datapoints {
			if datapoint.Sum != nil {
				totalRequests += *datapoint.Sum
			}
		}
		t.Logf("WAF processed requests in last hour: %.0f", totalRequests)
	}

	// Test 3: Verify WAF rules are optimized
	t.Log("Verifying WAF rule optimization...")

	wafRuleCount := terraform.Output(t, terraformOptions, "waf_rule_count")
	assert.LessOrEqual(t, wafRuleCount, "10", "WAF should have reasonable number of rules for cost optimization")
}

func TestS3CostOptimization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "cost-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get S3 bucket details
	s3BucketName := terraform.Output(t, terraformOptions, "s3_bucket_name")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	s3Svc := s3.New(sess)

	// Test 1: Verify S3 storage class optimization
	t.Log("Testing S3 storage optimization...")

	// Check bucket versioning (cost impact)
	versioningResult, err := s3Svc.GetBucketVersioning(&s3.GetBucketVersioningInput{
		Bucket: aws.String(s3BucketName),
	})
	require.NoError(t, err)

	assert.Equal(t, "Enabled", *versioningResult.Status, "Versioning should be enabled for content protection")

	// Test 2: Verify server-side encryption (no additional cost)
	t.Log("Verifying S3 encryption configuration...")

	encryptionResult, err := s3Svc.GetBucketEncryption(&s3.GetBucketEncryptionInput{
		Bucket: aws.String(s3BucketName),
	})
	require.NoError(t, err)

	assert.NotEmpty(t, encryptionResult.ServerSideEncryptionConfiguration, "S3 SSE should be configured")

	// Test 3: Check lifecycle policies for cost optimization
	t.Log("Checking S3 lifecycle policies...")

	lifecycleResult, err := s3Svc.GetBucketLifecycleConfiguration(&s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(s3BucketName),
	})

	if err == nil && lifecycleResult.Rules != nil {
		t.Logf("S3 bucket has %d lifecycle rules configured", len(lifecycleResult.Rules))
		for _, rule := range lifecycleResult.Rules {
			if rule.Status != nil && *rule.Status == "Enabled" {
				t.Logf("Lifecycle rule: %s", *rule.ID)
			}
		}
	} else {
		t.Log("No lifecycle policies configured - consider adding for cost optimization")
	}
}

func TestCertificateCostOptimization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "cost-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get certificate details
	certificateArn := terraform.Output(t, terraformOptions, "certificate_arn")

	// Test 1: Verify certificate is properly configured
	t.Log("Testing ACM certificate cost optimization...")

	assert.NotEmpty(t, certificateArn, "ACM certificate should be configured")

	// Test 2: Verify DNS validation (cost-effective)
	t.Log("Verifying DNS validation for cost optimization...")

	certificateValidation := terraform.Output(t, terraformOptions, "certificate_validation_method")
	assert.Equal(t, "DNS", certificateValidation, "DNS validation should be used for cost optimization")
}

func TestMonitoringCostOptimization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "cost-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test 1: Verify CloudTrail is configured but not excessive
	t.Log("Testing CloudTrail cost optimization...")

	cloudtrailEnabled := terraform.Output(t, terraformOptions, "cloudtrail_enabled")
	assert.Equal(t, "true", cloudtrailEnabled, "CloudTrail should be enabled for auditing")

	// Test 2: Verify log retention is reasonable
	t.Log("Testing log retention cost optimization...")

	cloudfrontLogRetention := terraform.Output(t, terraformOptions, "cloudfront_log_retention_days")
	wafLogRetention := terraform.Output(t, terraformOptions, "waf_log_retention_days")

	assert.Equal(t, "365", cloudfrontLogRetention, "CloudFront log retention should be reasonable")
	assert.Equal(t, "365", wafLogRetention, "WAF log retention should be reasonable")
}

func TestCacheOptimizationCosts(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "cost-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test 1: Monitor cache hit ratio for cost optimization
	t.Log("Testing CloudFront cache optimization...")

	cacheHitMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/CloudFront"),
		MetricName: aws.String("Requests"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("DistributionId"),
				Value: aws.String(distributionID),
			},
		},
		StartTime:  aws.Time(time.Now().Add(-1 * time.Hour)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int64(1800),
		Statistics: []*string{aws.String("Sum")},
	})

	require.NoError(t, err)

	if len(cacheHitMetrics.Datapoints) > 0 {
		totalRequests := 0.0
		for _, datapoint := range cacheHitMetrics.Datapoints {
			if datapoint.Sum != nil {
				totalRequests += *datapoint.Sum
			}
		}
		t.Logf("Total CloudFront requests in last hour: %.0f", totalRequests)

		// High cache hit ratio reduces origin requests and costs
		// Note: In a real scenario, you'd compare cache hits vs total requests
		assert.Greater(t, totalRequests, float64(0), "Should have some requests to measure cache performance")
	}

	// Test 2: Verify compression is enabled for cost reduction
	t.Log("Testing compression for cost optimization...")

	compressionEnabled := terraform.Output(t, terraformOptions, "compression_enabled")
	assert.Equal(t, "true", compressionEnabled, "Compression should be enabled for cost optimization")
}

func TestDataTransferCostOptimization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "cost-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test 1: Monitor data transfer costs
	t.Log("Monitoring data transfer costs...")

	bytesDownloadedMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/CloudFront"),
		MetricName: aws.String("BytesDownloaded"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("DistributionId"),
				Value: aws.String(distributionID),
			},
		},
		StartTime:  aws.Time(time.Now().Add(-1 * time.Hour)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int64(1800),
		Statistics: []*string{aws.String("Sum")},
	})

	require.NoError(t, err)

	if len(bytesDownloadedMetrics.Datapoints) > 0 {
		totalBytes := 0.0
		for _, datapoint := range bytesDownloadedMetrics.Datapoints {
			if datapoint.Sum != nil {
				totalBytes += *datapoint.Sum
			}
		}

		totalGB := totalBytes / (1024 * 1024 * 1024)
		t.Logf("Data transfer out: %.2f GB in last hour", totalGB)

		// Estimate cost (rough calculation)
		estimatedCost := totalGB * 0.085 // CloudFront data transfer cost
		t.Logf("Estimated CloudFront cost: $%.2f for last hour", estimatedCost)

		// Assert reasonable data transfer
		assert.Less(t, totalGB, float64(10), "Data transfer should be reasonable for cost control")
	}
}

// Helper function to extract WAF name from ARN
func extractWAFNameFromArn(arn string) string {
	// ARN format: arn:aws:wafv2:region:account:regional/webacl/name/id
	parts := strings.Split(arn, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// Helper function to extract WAF ID from ARN
func extractWAFIDFromArn(arn string) string {
	// ARN format: arn:aws:wafv2:region:account:regional/webacl/name/id
	parts := strings.Split(arn, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
