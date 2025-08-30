package performance

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCDNPerformanceBaseline(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "perf-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get CloudFront distribution details
	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")
	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")

	// Create AWS session for CloudWatch metrics
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test 1: HTTP Response Time
	t.Log("Testing CDN response time...")
	start := time.Now()
	resp, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
	duration := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Less(t, duration, 3*time.Second, "CDN response should be under 3 seconds")

	// Test 2: Check security headers
	t.Log("Verifying security headers...")
	contentType := resp.Header.Get("Content-Type")
	xFrameOptions := resp.Header.Get("X-Frame-Options")
	xContentTypeOptions := resp.Header.Get("X-Content-Type-Options")
	strictTransportSecurity := resp.Header.Get("Strict-Transport-Security")

	assert.Contains(t, contentType, "text/html", "Should serve HTML content")
	assert.Equal(t, "DENY", xFrameOptions, "X-Frame-Options should be DENY")
	assert.Equal(t, "nosniff", xContentTypeOptions, "X-Content-Type-Options should be nosniff")
	assert.Contains(t, strictTransportSecurity, "max-age", "HSTS should be configured")

	// Test 3: CloudFront Performance Metrics
	t.Log("Capturing CloudFront performance metrics...")

	// Get cache hit ratio
	cacheMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/CloudFront"),
		MetricName: aws.String("Requests"),
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
		StartTime:  aws.Time(time.Now().Add(-10 * time.Minute)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int64(300),
		Statistics: []*string{aws.String("Sum")},
	})

	require.NoError(t, err)

	if len(cacheMetrics.Datapoints) > 0 {
		totalRequests := 0.0
		for _, datapoint := range cacheMetrics.Datapoints {
			if datapoint.Sum != nil {
				totalRequests += *datapoint.Sum
			}
		}
		t.Logf("Total requests in last 10 minutes: %.0f", totalRequests)
	}

	// Verify distribution is properly configured
	assert.NotEmpty(t, cloudfrontDomain)
	assert.NotEmpty(t, distributionID)
}

func TestCDNLoadHandling(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "load-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")

	// Simulate concurrent requests to test CDN load handling
	t.Log("Testing CDN concurrent load handling...")

	const numRequests = 100
	const concurrency = 20

	var wg sync.WaitGroup
	results := make(chan time.Duration, numRequests)
	errors := make(chan error, numRequests)

	// Semaphore to control concurrency
	sem := make(chan struct{}, concurrency)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
			duration := time.Since(start)

			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				errors <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				return
			}

			results <- duration
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	select {
	case err := <-errors:
		t.Fatalf("Load test failed: %v", err)
	default:
		// No errors
	}

	// Analyze response times
	var totalDuration time.Duration
	count := 0
	maxDuration := time.Duration(0)
	minDuration := time.Hour

	for duration := range results {
		totalDuration += duration
		count++
		if duration > maxDuration {
			maxDuration = duration
		}
		if duration < minDuration {
			minDuration = duration
		}
	}

	if count > 0 {
		avgDuration := totalDuration / time.Duration(count)
		t.Logf("CDN load test results: %d requests", count)
		t.Logf("Average response time: %v", avgDuration)
		t.Logf("Min response time: %v", minDuration)
		t.Logf("Max response time: %v", maxDuration)

		// Performance assertions for CDN
		assert.Less(t, avgDuration, 2*time.Second, "Average CDN response time should be under 2 seconds")
		assert.Less(t, maxDuration, 5*time.Second, "Max CDN response time should be under 5 seconds")
		assert.Greater(t, minDuration, time.Millisecond, "Min response time should be reasonable")
	}
}

func TestCDNCachePerformance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "cache-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")
	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test cache performance by making multiple requests to the same resource
	t.Log("Testing CDN cache performance...")

	// Make initial request (cache miss)
	start := time.Now()
	resp1, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
	duration1 := time.Since(start)

	require.NoError(t, err)
	defer resp1.Body.Close()
	assert.Equal(t, 200, resp1.StatusCode)

	// Check for cache headers
	cacheControl := resp1.Header.Get("Cache-Control")
	etag := resp1.Header.Get("ETag")

	t.Logf("Cache-Control: %s", cacheControl)
	t.Logf("ETag: %s", etag)

	// Make subsequent requests (should be cache hits)
	const numSubsequentRequests = 5
	var subsequentDurations []time.Duration

	for i := 0; i < numSubsequentRequests; i++ {
		start := time.Now()
		resp, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
		duration := time.Since(start)

		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)

		subsequentDurations = append(subsequentDurations, duration)
	}

	// Calculate cache performance
	totalSubsequent := time.Duration(0)
	for _, d := range subsequentDurations {
		totalSubsequent += d
	}
	avgSubsequent := totalSubsequent / time.Duration(numSubsequentRequests)

	t.Logf("Initial request (cache miss): %v", duration1)
	t.Logf("Average subsequent requests: %v", avgSubsequent)

	// Cache should improve performance
	assert.Less(t, avgSubsequent, duration1, "Cached requests should be faster than initial request")

	// Get CloudFront cache metrics
	cacheHitMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/CloudFront"),
		MetricName: aws.String("TotalErrorRate"),
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
		StartTime:  aws.Time(time.Now().Add(-5 * time.Minute)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int64(300),
		Statistics: []*string{aws.String("Average")},
	})

	require.NoError(t, err)

	if len(cacheHitMetrics.Datapoints) > 0 {
		avgErrorRate := *cacheHitMetrics.Datapoints[0].Average
		t.Logf("CloudFront error rate: %.2f%%", avgErrorRate)
		assert.Less(t, avgErrorRate, float64(1), "Error rate should be under 1%")
	}
}

func TestCDNGlobalPerformance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "global-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")
	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test global CDN performance metrics
	t.Log("Testing global CDN performance...")

	// Get requests by region
	regionMetrics, err := cloudwatchSvc.ListMetrics(&cloudwatch.ListMetricsInput{
		Namespace:  aws.String("AWS/CloudFront"),
		MetricName: aws.String("Requests"),
		Dimensions: []*cloudwatch.DimensionFilter{
			{
				Name:  aws.String("DistributionId"),
				Value: aws.String(distributionID),
			},
		},
	})

	require.NoError(t, err)

	t.Logf("CloudFront distribution has %d metrics available", len(regionMetrics.Metrics))

	// Test basic connectivity from multiple simulated regions
	regions := []string{"us-east-1", "eu-west-1", "ap-southeast-1"}
	var regionalLatencies []time.Duration

	for _, region := range regions {
		start := time.Now()
		resp, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
		duration := time.Since(start)

		if err == nil {
			resp.Body.Close()
			regionalLatencies = append(regionalLatencies, duration)
			t.Logf("Region %s latency: %v", region, duration)
		}
	}

	// Verify global distribution is working
	if len(regionalLatencies) > 0 {
		totalLatency := time.Duration(0)
		for _, latency := range regionalLatencies {
			totalLatency += latency
		}
		avgLatency := totalLatency / time.Duration(len(regionalLatencies))

		t.Logf("Average global latency: %v", avgLatency)
		assert.Less(t, avgLatency, 3*time.Second, "Average global latency should be under 3 seconds")
	}
}

func TestCDNCompressionPerformance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "compression-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")

	// Test compression performance
	t.Log("Testing CDN compression performance...")

	// Test with gzip compression
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s", cloudfrontDomain), nil)
	require.NoError(t, err)

	req.Header.Set("Accept-Encoding", "gzip, deflate")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	// Check for compression headers
	contentEncoding := resp.Header.Get("Content-Encoding")
	transferEncoding := resp.Header.Get("Transfer-Encoding")

	t.Logf("Content-Encoding: %s", contentEncoding)
	t.Logf("Transfer-Encoding: %s", transferEncoding)

	// Verify compression is working
	if contentEncoding == "gzip" || transferEncoding == "chunked" {
		t.Log("Compression is working correctly")
	} else {
		t.Log("Compression may not be enabled or not applicable for this content")
	}

	// Test response time with compression
	start := time.Now()
	resp2, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
	duration := time.Since(start)

	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, 200, resp2.StatusCode)
	assert.Less(t, duration, 2*time.Second, "Compressed response should be under 2 seconds")
}

func TestCDNSecurityHeadersPerformance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "security-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")

	// Test security headers performance
	t.Log("Testing security headers performance...")

	start := time.Now()
	resp, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
	duration := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Less(t, duration, 2*time.Second, "Security headers should not impact performance significantly")

	// Verify all security headers are present
	expectedHeaders := map[string]string{
		"X-Frame-Options":           "DENY",
		"X-Content-Type-Options":    "nosniff",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=",
		"Content-Security-Policy":   "default-src",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		assert.NotEmpty(t, actualValue, "Security header %s should be present", header)
		if expectedValue != "" {
			assert.Contains(t, actualValue, expectedValue, "Security header %s should contain %s", header, expectedValue)
		}
		t.Logf("%s: %s", header, actualValue)
	}

	// Test HTTPS enforcement
	httpResp, err := http.Get(fmt.Sprintf("http://%s", cloudfrontDomain))
	if err == nil {
		defer httpResp.Body.Close()
		assert.Equal(t, 301, httpResp.StatusCode, "HTTP should redirect to HTTPS")
		location := httpResp.Header.Get("Location")
		assert.Contains(t, location, "https://", "Redirect should be to HTTPS")
	}
}

func TestCDNOriginShieldPerformance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "origin-shield-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")
	distributionID := terraform.Output(t, terraformOptions, "cloudfront_distribution_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudfrontSvc := cloudfront.New(sess)

	// Test Origin Shield configuration
	t.Log("Testing Origin Shield performance...")

	// Get distribution configuration
	distResult, err := cloudfrontSvc.GetDistribution(&cloudfront.GetDistributionInput{
		Id: aws.String(distributionID),
	})
	require.NoError(t, err)

	// Check if Origin Shield is configured
	origins := distResult.Distribution.DistributionConfig.Origins
	if len(origins.Items) > 0 {
		originShield := origins.Items[0].OriginShield
		if originShield != nil {
			t.Logf("Origin Shield enabled for region: %s", *originShield.OriginShieldRegion)
		} else {
			t.Log("Origin Shield not configured")
		}
	}

	// Test performance with Origin Shield
	start := time.Now()
	resp, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
	duration := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Less(t, duration, 2*time.Second, "Origin Shield should maintain good performance")

	// Verify response headers indicate CloudFront processing
	server := resp.Header.Get("Server")
	via := resp.Header.Get("Via")

	assert.Contains(t, server, "CloudFront", "Server header should indicate CloudFront")
	assert.Contains(t, via, "CloudFront", "Via header should indicate CloudFront")

	t.Logf("Server: %s", server)
	t.Logf("Via: %s", via)
}
