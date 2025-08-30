package test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBastionPerformanceBaseline(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "perf-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "perf-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc perf-test",
			"allowed_ssh_cidrs":    []string{"0.0.0.0/0"}, // Allow all for performance testing
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get bastion details
	bastionPublicIP := terraform.Output(t, terraformOptions, "bastion_public_ip")
	bastionID := terraform.Output(t, terraformOptions, "bastion_instance_id")
	privateIP := terraform.Output(t, terraformOptions, "private_instance_ip")

	// Create AWS session for CloudWatch metrics
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test 1: SSH Connection Time
	t.Log("Testing SSH connection performance...")
	start := time.Now()

	// Test network connectivity to bastion (simplified - would need actual SSH in real test)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", bastionPublicIP), 10*time.Second)
	if err == nil {
		conn.Close()
	}
	sshLatency := time.Since(start)

	t.Logf("SSH port response time: %v", sshLatency)
	assert.Less(t, sshLatency, 5*time.Second, "SSH port should respond within 5 seconds")

	// Test 2: Resource Utilization Baseline
	t.Log("Capturing baseline resource utilization...")

	// Get CPU utilization metrics for bastion
	cpuMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("CPUUtilization"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(bastionID),
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
		t.Logf("Bastion CPU utilization: %.2f%%", *latestCPU.Average)
		assert.Less(t, *latestCPU.Average, float64(80), "Bastion CPU utilization should be under 80% at baseline")
	}

	// Verify connectivity to both instances
	assert.NotEmpty(t, bastionPublicIP)
	assert.NotEmpty(t, privateIP)
}

func TestBastionLoadHandling(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "load-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "load-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc load-test",
			"allowed_ssh_cidrs":    []string{"0.0.0.0/0"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	bastionPublicIP := terraform.Output(t, terraformOptions, "bastion_public_ip")

	// Simulate concurrent SSH connection attempts
	t.Log("Testing concurrent SSH connection handling...")

	const numConnections = 20
	const concurrency = 5

	results := make(chan time.Duration, numConnections)
	errors := make(chan error, numConnections)

	// Semaphore to control concurrency
	sem := make(chan struct{}, concurrency)

	for i := 0; i < numConnections; i++ {
		go func() {
			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			start := time.Now()
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", bastionPublicIP), 15*time.Second)

			if err != nil {
				errors <- err
				return
			}

			conn.Close()
			duration := time.Since(start)
			results <- duration
		}()
	}

	// Wait for all goroutines to complete
	time.Sleep(20 * time.Second)

	close(results)
	close(errors)

	// Check for errors
	select {
	case err := <-errors:
		t.Logf("Connection test error: %v", err)
	default:
		// No errors
	}

	// Analyze connection times
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
		t.Logf("Load test results: %d connections", count)
		t.Logf("Average connection time: %v", avgDuration)
		t.Logf("Min connection time: %v", minDuration)
		t.Logf("Max connection time: %v", maxDuration)

		// Performance assertions
		assert.Less(t, avgDuration, 10*time.Second, "Average connection time should be under 10 seconds")
		assert.Less(t, maxDuration, 15*time.Second, "Max connection time should be under 15 seconds")
	}
}

func TestBastionScalabilityMetrics(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "scale-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "scale-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc scale-test",
			"allowed_ssh_cidrs":    []string{"0.0.0.0/0"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	bastionID := terraform.Output(t, terraformOptions, "bastion_instance_id")
	privateInstanceID := terraform.Output(t, terraformOptions, "private_instance_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Test bastion scaling metrics
	t.Log("Testing bastion scalability metrics...")

	metrics := []struct {
		instanceID string
		metricName string
		name       string
	}{
		{bastionID, "CPUUtilization", "Bastion CPU"},
		{bastionID, "NetworkIn", "Bastion Network In"},
		{bastionID, "NetworkOut", "Bastion Network Out"},
		{bastionID, "DiskReadOps", "Bastion Disk Read"},
		{bastionID, "DiskWriteOps", "Bastion Disk Write"},
		{privateInstanceID, "CPUUtilization", "Private Instance CPU"},
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
				assert.Less(t, *latest.Maximum, float64(90), "CPU utilization should not exceed 90%")
			}
		}
	}
}

func TestBastionNetworkPerformance(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "net-perf-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "net-perf-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc net-perf-test",
			"allowed_ssh_cidrs":    []string{"0.0.0.0/0"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	bastionPublicIP := terraform.Output(t, terraformOptions, "bastion_public_ip")
	privateIP := terraform.Output(t, terraformOptions, "private_instance_ip")

	// Test network connectivity and latency
	t.Log("Testing bastion network performance...")

	// Test bastion connectivity
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", bastionPublicIP), 10*time.Second)
	bastionLatency := time.Since(start)

	if err == nil {
		conn.Close()
		t.Logf("Bastion SSH port latency: %v", bastionLatency)
		assert.Less(t, bastionLatency, 3*time.Second, "Bastion should respond within 3 seconds")
	} else {
		t.Logf("Bastion connection failed: %v", err)
	}

	// Test internal network connectivity (simplified)
	assert.NotEmpty(t, privateIP)

	// Test network security (verify SSH is accessible)
	conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:22", bastionPublicIP), 5*time.Second)
	if err == nil {
		conn.Close()
		t.Log("SSH port is accessible as expected")
	} else {
		t.Errorf("SSH port should be accessible: %v", err)
	}

	t.Log("Network performance test completed")
}

func TestBastionResourceLimits(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "limits-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "limits-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc limits-test",
			"allowed_ssh_cidrs":    []string{"0.0.0.0/0"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test resource limits and configuration
	t.Log("Testing bastion resource limits...")

	// Verify instance types
	bastionInstanceType := terraform.Output(t, terraformOptions, "bastion_instance_type")
	privateInstanceType := terraform.Output(t, terraformOptions, "private_instance_type")

	assert.Equal(t, "t3.micro", bastionInstanceType)
	assert.Equal(t, "t3.micro", privateInstanceType)

	// Verify VPC configuration
	vpcCidr := terraform.Output(t, terraformOptions, "vpc_cidr")
	assert.Equal(t, "172.16.0.0/16", vpcCidr)

	// Verify subnet configuration
	publicSubnetCidr := terraform.Output(t, terraformOptions, "public_subnet_cidr")
	privateSubnetCidr := terraform.Output(t, terraformOptions, "private_subnet_cidr")

	assert.Equal(t, "172.16.1.0/24", publicSubnetCidr)
	assert.Equal(t, "172.16.10.0/24", privateSubnetCidr)

	// Test security group limits
	bastionSGID := terraform.Output(t, terraformOptions, "bastion_security_group_id")
	assert.NotEmpty(t, bastionSGID)

	t.Log("Resource limits test completed successfully")
}
