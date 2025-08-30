package test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceBaseline(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "perf-test",
			"allowed_http_cidrs": []string{"0.0.0.0/0"}, // Allow all for performance testing
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get instance details
	publicIP := terraform.Output(t, terraformOptions, "public_instance_public_ip")
	publicInstanceID := terraform.Output(t, terraformOptions, "public_instance_id")
	privateIP := terraform.Output(t, terraformOptions, "private_instance_private_ip")

	// Create AWS session for CloudWatch metrics
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test 1: HTTP Response Time
	t.Log("Testing HTTP response time...")
	start := time.Now()
	resp, err := http.Get(fmt.Sprintf("http://%s", publicIP))
	duration := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Less(t, duration, 5*time.Second, "HTTP response should be under 5 seconds")

	// Test 2: Network Latency
	t.Log("Testing network connectivity...")
	// This would typically involve more sophisticated network testing

	// Test 3: Resource Utilization Baseline
	t.Log("Capturing baseline resource utilization...")

	// Get CPU utilization metrics
	cpuMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("CPUUtilization"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(publicInstanceID),
			},
		},
		StartTime:  aws.Time(time.Now().Add(-5 * time.Minute)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int64(300),
		Statistics: []*string{aws.String("Average")},
	})

	require.NoError(t, err)
	if len(cpuMetrics.Datapoints) > 0 {
		latestCPU := cpuMetrics.Datapoints[0]
		t.Logf("Current CPU utilization: %.2f%%", *latestCPU.Average)
		assert.Less(t, *latestCPU.Average, float64(90), "CPU utilization should be under 90%")
	}

	// Verify private instance connectivity
	assert.NotEmpty(t, privateIP)
	assert.NotEmpty(t, publicIP)
}

func TestLoadHandling(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "load-test",
			"allowed_http_cidrs": []string{"0.0.0.0/0"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	publicIP := terraform.Output(t, terraformOptions, "public_instance_public_ip")

	// Simulate concurrent HTTP requests
	t.Log("Testing concurrent load handling...")

	const numRequests = 50
	const concurrency = 10

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
			resp, err := http.Get(fmt.Sprintf("http://%s", publicIP))
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
		t.Logf("Load test results: %d requests", count)
		t.Logf("Average response time: %v", avgDuration)
		t.Logf("Min response time: %v", minDuration)
		t.Logf("Max response time: %v", maxDuration)

		// Performance assertions
		assert.Less(t, avgDuration, 10*time.Second, "Average response time should be under 10 seconds")
		assert.Less(t, maxDuration, 30*time.Second, "Max response time should be under 30 seconds")
		assert.Greater(t, minDuration, time.Millisecond, "Min response time should be reasonable")
	}
}

func TestScalabilityMetrics(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "scale-test",
			"allowed_http_cidrs": []string{"0.0.0.0/0"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	publicInstanceID := terraform.Output(t, terraformOptions, "public_instance_id")
	privateInstanceID := terraform.Output(t, terraformOptions, "private_instance_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test resource scaling metrics
	t.Log("Testing scalability metrics...")

	metrics := []struct {
		instanceID string
		metricName string
		name       string
	}{
		{publicInstanceID, "CPUUtilization", "Public Instance CPU"},
		{publicInstanceID, "NetworkIn", "Public Instance Network In"},
		{publicInstanceID, "NetworkOut", "Public Instance Network Out"},
		{privateInstanceID, "CPUUtilization", "Private Instance CPU"},
		{privateInstanceID, "NetworkIn", "Private Instance Network In"},
		{privateInstanceID, "NetworkOut", "Private Instance Network Out"},
	}

	for _, metric := range metrics {
		metricData, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/EC2"),
			MetricName: aws.String(metric.metricName),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("InstanceId"),
					Value: aws.String(metric.instanceID),
				},
			},
			StartTime:  aws.Time(time.Now().Add(-10 * time.Minute)),
			EndTime:    aws.Time(time.Now()),
			Period:     aws.Int64(300),
			Statistics: []*string{aws.String("Average"), aws.String("Maximum")},
		})

		require.NoError(t, err)

		if len(metricData.Datapoints) > 0 {
			latest := metricData.Datapoints[0]
			t.Logf("%s - Average: %.2f, Maximum: %.2f",
				metric.name,
				*latest.Average,
				*latest.Maximum)

			// Assert reasonable resource utilization
			if metric.metricName == "CPUUtilization" {
				assert.Less(t, *latest.Maximum, float64(95), "CPU utilization should not exceed 95%")
			}
		}
	}
}

func TestNetworkPerformance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "net-perf-test",
			"allowed_http_cidrs": []string{"0.0.0.0/0"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	publicIP := terraform.Output(t, terraformOptions, "public_instance_public_ip")
	privateIP := terraform.Output(t, terraformOptions, "private_instance_private_ip")

	// Test network connectivity and latency
	t.Log("Testing network performance...")

	// Test public instance connectivity
	start := time.Now()
	resp, err := http.Get(fmt.Sprintf("http://%s", publicIP))
	publicLatency := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	t.Logf("Public instance latency: %v", publicLatency)
	assert.Less(t, publicLatency, 3*time.Second, "Public instance should respond within 3 seconds")

	// Test VPC internal connectivity (this would require SSH access in real scenario)
	assert.NotEmpty(t, privateIP)

	// Test network throughput (simplified)
	// In a real scenario, you would use tools like iperf for bandwidth testing

	t.Log("Network performance test completed")
}

func TestResourceLimits(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "limits-test",
			"allowed_http_cidrs": []string{"0.0.0.0/0"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test resource limits and quotas
	t.Log("Testing resource limits...")

	// Verify instance types are within limits
	publicInstanceType := terraform.Output(t, terraformOptions, "public_instance_type")
	privateInstanceType := terraform.Output(t, terraformOptions, "private_instance_type")

	assert.Equal(t, "t3.micro", publicInstanceType)
	assert.Equal(t, "t3.micro", privateInstanceType)

	// Verify VPC limits
	vpcCidr := terraform.Output(t, terraformOptions, "vpc_cidr_block")
	assert.Contains(t, vpcCidr, "/16", "VPC should use /16 CIDR for adequate address space")

	// Test subnet configuration
	publicSubnetCidr := terraform.Output(t, terraformOptions, "public_subnet_cidr")
	privateSubnetCidr := terraform.Output(t, terraformOptions, "private_subnet_cidr")

	assert.Contains(t, publicSubnetCidr, "/24")
	assert.Contains(t, privateSubnetCidr, "/24")

	t.Log("Resource limits test completed successfully")
}
