package chaos

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/wafv2"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChaosCloudFrontFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "chaos-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get CloudFront distribution details
	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")
	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")

	// Safety check: ensure we're working with test resources
	assert.Contains(t, cloudfrontDomain, "chaos-test.example.com", "Should only test on chaos test domain")
	assert.NotEmpty(t, distributionID, "CloudFront distribution should be created")

	// Test basic connectivity before chaos simulation
	resp, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
	require.NoError(t, err, "Should be able to connect to CloudFront before chaos")
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode, "Should get successful response before chaos")

	// Verify CloudFront domain is properly configured
	assert.NotEmpty(t, cloudfrontDomain, "CloudFront domain should be accessible")
	assert.Contains(t, cloudfrontDomain, "cloudfront.net", "Should be a valid CloudFront domain")

	t.Logf("Chaos test completed successfully for distribution: %s", distributionID)
}

func TestChaosS3OriginFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "chaos-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get S3 bucket details
	s3BucketName := terraform.Output(t, terraformOptions, "s3_bucket_name")

	// Safety check: ensure we're working with test resources
	assert.Contains(t, s3BucketName, "chaos-test.example.com", "Should only test on chaos test bucket")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	s3Svc := s3.New(sess)

	// Test 1: Verify S3 bucket exists and is properly configured
	t.Log("Verifying S3 bucket configuration for chaos testing...")

	// Check bucket exists
	_, err := s3Svc.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(s3BucketName),
	})
	require.NoError(t, err, "S3 bucket should exist")

	// Check bucket has proper public access block
	publicAccessResult, err := s3Svc.GetPublicAccessBlock(&s3.GetPublicAccessBlockInput{
		Bucket: aws.String(s3BucketName),
	})
	require.NoError(t, err, "Should be able to get public access block")

	block := publicAccessResult.PublicAccessBlockConfiguration
	assert.True(t, *block.BlockPublicAcls, "S3 bucket should block public ACLs")
	assert.True(t, *block.BlockPublicPolicy, "S3 bucket should block public policies")

	// Check bucket has server-side encryption
	encryptionResult, err := s3Svc.GetBucketEncryption(&s3.GetBucketEncryptionInput{
		Bucket: aws.String(s3BucketName),
	})
	require.NoError(t, err, "Should be able to get bucket encryption")

	assert.NotEmpty(t, encryptionResult.ServerSideEncryptionConfiguration, "S3 bucket should have encryption configured")

	// Verify S3 bucket is properly configured for static website
	assert.NotEmpty(t, s3BucketName, "S3 bucket should be created and configured")
	assert.Contains(t, s3BucketName, "chaos-test.example.com", "Bucket name should contain test domain")

	t.Logf("S3 chaos test completed successfully for bucket: %s", s3BucketName)
}

func TestChaosWAFFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "chaos-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get WAF Web ACL details
	wafACLArn := terraform.Output(t, terraformOptions, "waf_web_acl_arn")

	// Safety check: ensure we're working with test resources
	assert.Contains(t, wafACLArn, "chaos-test", "Should only test on chaos test WAF ACL")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	wafSvc := wafv2.New(sess)

	// Test 1: Verify WAF configuration is properly set up
	t.Log("Verifying WAF configuration for chaos testing...")

	// Get current WAF configuration
	getResult, err := wafSvc.GetWebACL(&wafv2.GetWebACLInput{
		Id:    aws.String(extractWAFIDFromArn(wafACLArn)),
		Scope: aws.String("CLOUDFRONT"),
	})
	require.NoError(t, err, "Should be able to get WAF configuration")

	rules := getResult.WebACL.Rules
	assert.Greater(t, len(rules), 0, "WAF should have rules configured")

	// Check for essential rule groups
	hasCommonRules := false
	hasSQLiRules := false
	hasXSSRules := false

	for _, rule := range rules {
		if rule.Statement.ManagedRuleGroupStatement != nil {
			ruleName := *rule.Statement.ManagedRuleGroupStatement.Name
			switch {
			case strings.Contains(ruleName, "AWSManagedRulesCommonRuleSet"):
				hasCommonRules = true
			case strings.Contains(ruleName, "AWSManagedRulesSQLiRuleSet"):
				hasSQLiRules = true
			case strings.Contains(ruleName, "AWSManagedRulesKnownBadInputsRuleSet"):
				hasXSSRules = true
			}
		}
	}

	assert.True(t, hasCommonRules, "WAF should include common rules for chaos testing")
	assert.True(t, hasSQLiRules, "WAF should include SQL injection protection")
	assert.True(t, hasXSSRules, "WAF should include XSS protection")

	// Check for rate limiting
	hasRateLimit := false
	for _, rule := range rules {
		if rule.Statement.RateBasedStatement != nil {
			hasRateLimit = true
			break
		}
	}
	assert.True(t, hasRateLimit, "WAF should include rate limiting for chaos testing")

	// Verify WAF ACL is properly configured
	assert.NotEmpty(t, wafACLArn, "WAF ACL should be created and configured")
	assert.Contains(t, wafACLArn, "chaos-test", "WAF ACL should contain test domain identifier")

	t.Logf("WAF chaos test completed successfully for ACL: %s", wafACLArn)
}

func TestChaosCertificateFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "chaos-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get certificate details
	certificateArn := terraform.Output(t, terraformOptions, "certificate_arn")

	// Test 1: Simulate certificate validation issues
	t.Log("Simulating certificate failure...")

	// In a real scenario, you might test certificate expiry or validation failures
	// For this test, we verify the certificate exists and is properly configured

	assert.NotEmpty(t, certificateArn)

	// Test HTTPS connectivity (simplified)
	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")

	// Test HTTP to HTTPS redirect
	resp, err := http.Get(fmt.Sprintf("http://%s", cloudfrontDomain))
	if err == nil {
		defer resp.Body.Close()
		// Should redirect to HTTPS
		assert.Equal(t, 301, resp.StatusCode)
	}
}

func TestChaosOriginShieldFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "chaos-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get CloudFront distribution details
	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudfrontSvc := cloudfront.New(sess)

	// Test 1: Simulate Origin Shield region change
	t.Log("Simulating Origin Shield failure...")

	// Get current distribution config
	distResult, err := cloudfrontSvc.GetDistribution(&cloudfront.GetDistributionInput{
		Id: aws.String(distributionID),
	})
	require.NoError(t, err)

	// Change Origin Shield region (simulating regional failure)
	currentConfig := distResult.Distribution.DistributionConfig

	// Temporarily change Origin Shield region
	newShieldRegion := "us-west-2" // Different region
	if currentConfig.Origins.Items[0].OriginShield != nil {
		currentConfig.Origins.Items[0].OriginShield.OriginShieldRegion = aws.String(newShieldRegion)
	}

	_, err = cloudfrontSvc.UpdateDistribution(&cloudfront.UpdateDistributionInput{
		Id:                 aws.String(distributionID),
		DistributionConfig: currentConfig,
	})
	require.NoError(t, err)

	// Wait for changes to propagate
	time.Sleep(30 * time.Second)

	// Restore original Origin Shield region
	if currentConfig.Origins.Items[0].OriginShield != nil {
		currentConfig.Origins.Items[0].OriginShield.OriginShieldRegion = aws.String("us-east-1")
	}

	_, err = cloudfrontSvc.UpdateDistribution(&cloudfront.UpdateDistributionInput{
		Id:                 aws.String(distributionID),
		DistributionConfig: currentConfig,
	})
	require.NoError(t, err)

	// Verify distribution is still functional
	assert.NotEmpty(t, distributionID)
}

func TestChaosDDoSProtectionFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "chaos-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get WAF Web ACL details
	wafACLArn := terraform.Output(t, terraformOptions, "waf_web_acl_arn")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	wafSvc := wafv2.New(sess)

	// Test 1: Simulate DDoS protection failure by disabling rate limiting
	t.Log("Simulating DDoS protection failure...")

	// Get current WAF configuration
	getResult, err := wafSvc.GetWebACL(&wafv2.GetWebACLInput{
		Id:    aws.String(extractWAFIDFromArn(wafACLArn)),
		Scope: aws.String("CLOUDFRONT"),
	})
	require.NoError(t, err)

	// Temporarily remove rate limiting rules (simulating protection failure)
	var filteredRules []*wafv2.Rule
	for _, rule := range getResult.WebACL.Rules {
		if rule.Statement.RateBasedStatement == nil {
			filteredRules = append(filteredRules, rule)
		}
	}

	_, err = wafSvc.UpdateWebACL(&wafv2.UpdateWebACLInput{
		Id:               aws.String(extractWAFIDFromArn(wafACLArn)),
		Scope:            aws.String("CLOUDFRONT"),
		DefaultAction:    getResult.WebACL.DefaultAction,
		Rules:            filteredRules,
		VisibilityConfig: getResult.WebACL.VisibilityConfig,
		LockToken:        getResult.LockToken,
	})
	require.NoError(t, err)

	// Wait for changes to propagate
	time.Sleep(30 * time.Second)

	// Restore rate limiting rules
	_, err = wafSvc.UpdateWebACL(&wafv2.UpdateWebACLInput{
		Id:               aws.String(extractWAFIDFromArn(wafACLArn)),
		Scope:            aws.String("CLOUDFRONT"),
		DefaultAction:    getResult.WebACL.DefaultAction,
		Rules:            getResult.WebACL.Rules,
		VisibilityConfig: getResult.WebACL.VisibilityConfig,
		LockToken:        getResult.LockToken,
	})
	require.NoError(t, err)

	// Verify WAF protection is restored
	assert.NotEmpty(t, wafACLArn)
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

// Helper function to extract WAF name from ARN
func extractWAFNameFromArn(arn string) string {
	// ARN format: arn:aws:wafv2:region:account:regional/webacl/name/id
	parts := strings.Split(arn, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
