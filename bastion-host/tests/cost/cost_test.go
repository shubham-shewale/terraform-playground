package test

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBastionCostOptimizationInstanceSizing(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "cost-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "cost-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc cost-test",
			"allowed_ssh_cidrs":    []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify optimal instance types for bastion
	bastionInstanceType := terraform.Output(t, terraformOptions, "bastion_instance_type")
	privateInstanceType := terraform.Output(t, terraformOptions, "private_instance_type")

	// Assert cost-effective instance types
	assert.Equal(t, "t3.micro", bastionInstanceType, "Bastion should use cost-effective t3.micro instance")
	assert.Equal(t, "t3.micro", privateInstanceType, "Private instance should use cost-effective t3.micro instance")

	// Verify instances are using gp3 volumes (more cost-effective than gp2)
	bastionVolumeType := terraform.Output(t, terraformOptions, "bastion_volume_type")
	privateVolumeType := terraform.Output(t, terraformOptions, "private_instance_volume_type")

	assert.Equal(t, "gp3", bastionVolumeType, "Should use cost-effective gp3 volumes for bastion")
	assert.Equal(t, "gp3", privateVolumeType, "Should use cost-effective gp3 volumes for private instance")
}

func TestBastionCostOptimizationResourceUtilization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "cost-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "cost-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc cost-test",
			"allowed_ssh_cidrs":    []string{"10.0.0.0/8"},
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

	// Check CPU utilization for bastion host
	instances := []string{bastionID, privateInstanceID}
	instanceNames := []string{"bastion", "private"}

	for i, instanceID := range instances {
		metrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/EC2"),
			MetricName: aws.String("CPUUtilization"),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("InstanceId"),
					Value: aws.String(instanceID),
				},
			},
			StartTime:  aws.Time(time.Now().Add(-1 * time.Hour)),
			EndTime:    aws.Time(time.Now()),
			Period:     aws.Int64(1800), // 30 minutes
			Statistics: []*string{aws.String("Average"), aws.String("Maximum")},
		})

		require.NoError(t, err)

		if len(metrics.Datapoints) > 0 {
			totalAvg := 0.0
			maxValue := 0.0
			count := 0

			for _, datapoint := range metrics.Datapoints {
				if datapoint.Average != nil {
					totalAvg += *datapoint.Average
					count++
				}
				if datapoint.Maximum != nil && *datapoint.Maximum > maxValue {
					maxValue = *datapoint.Maximum
				}
			}

			if count > 0 {
				avgCPU := totalAvg / float64(count)
				t.Logf("%s instance - Average CPU: %.2f%%, Max CPU: %.2f%%", instanceNames[i], avgCPU, maxValue)

				// Cost optimization: bastion should have low utilization
				if instanceNames[i] == "bastion" {
					assert.Less(t, avgCPU, float64(20), "Bastion average CPU should be under 20% for cost optimization")
					assert.Less(t, maxValue, float64(60), "Bastion max CPU should be under 60% for cost optimization")
				} else {
					assert.Less(t, avgCPU, float64(30), "Private instance average CPU should be under 30%")
					assert.Less(t, maxValue, float64(80), "Private instance max CPU should be under 80%")
				}
			}
		}
	}
}

func TestBastionCostOptimizationUnusedResources(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "cost-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "cost-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc cost-test",
			"allowed_ssh_cidrs":    []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify no unused Elastic IPs (bastion should use EIP efficiently)
	bastionEIP := terraform.Output(t, terraformOptions, "bastion_elastic_ip")
	assert.NotEmpty(t, bastionEIP, "Bastion should have an EIP for accessibility")

	// Verify NAT Gateway exists but is used efficiently
	natGatewayID := terraform.Output(t, terraformOptions, "nat_gateway_id")
	assert.NotEmpty(t, natGatewayID, "NAT Gateway should exist for private subnet egress")

	// Verify VPC Endpoints are configured for cost-effective AWS service access
	vpcEndpointCount := terraform.Output(t, terraformOptions, "vpc_endpoint_count")
	assert.Greater(t, vpcEndpointCount, "0", "VPC Endpoints should be configured for cost optimization")
}

func TestBastionCostOptimizationStorageOptimization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "cost-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "cost-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc cost-test",
			"allowed_ssh_cidrs":    []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify EBS volume sizes are minimal for bastion use case
	bastionVolumeSize := terraform.Output(t, terraformOptions, "bastion_volume_size")
	privateVolumeSize := terraform.Output(t, terraformOptions, "private_instance_volume_size")

	// Bastion typically needs minimal storage
	assert.LessOrEqual(t, bastionVolumeSize, 20, "Bastion should use minimal volume size (≤20GB)")
	assert.LessOrEqual(t, privateVolumeSize, 20, "Private instance should use minimal volume size (≤20GB)")

	// Verify encryption is enabled (no additional cost)
	bastionEncrypted := terraform.Output(t, terraformOptions, "bastion_encrypted")
	privateEncrypted := terraform.Output(t, terraformOptions, "private_instance_encrypted")

	assert.Equal(t, "true", bastionEncrypted, "Bastion EBS encryption should be enabled")
	assert.Equal(t, "true", privateEncrypted, "Private instance EBS encryption should be enabled")
}

func TestBastionCostOptimizationMonitoringCosts(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "cost-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "cost-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc cost-test",
			"allowed_ssh_cidrs":    []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify detailed monitoring is enabled for cost-effective operations
	bastionMonitoring := terraform.Output(t, terraformOptions, "bastion_monitoring")
	privateMonitoring := terraform.Output(t, terraformOptions, "private_instance_monitoring")

	assert.Equal(t, "true", bastionMonitoring, "Bastion detailed monitoring should be enabled")
	assert.Equal(t, "true", privateMonitoring, "Private instance detailed monitoring should be enabled")

	// Verify CloudWatch log retention is reasonable
	bastionLogRetention := terraform.Output(t, terraformOptions, "bastion_log_retention_days")
	assert.Equal(t, "30", bastionLogRetention, "Bastion log retention should be 30 days for cost optimization")

	// Verify CloudTrail is configured but not excessive
	trailName := terraform.Output(t, terraformOptions, "cloudtrail_name")
	assert.NotEmpty(t, trailName, "CloudTrail should be configured for auditing")

	// Verify SNS topic exists for alerts (cost-effective notification system)
	snsTopicArn := terraform.Output(t, terraformOptions, "sns_topic_arn")
	assert.NotEmpty(t, snsTopicArn, "SNS topic should exist for cost-effective alerting")
}

func TestBastionCostOptimizationDataTransfer(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "cost-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "cost-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc cost-test",
			"allowed_ssh_cidrs":    []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	bastionID := terraform.Output(t, terraformOptions, "bastion_instance_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Check network data transfer costs for bastion
	networkMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("NetworkOut"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(bastionID),
			},
		},
		StartTime:  aws.Time(time.Now().Add(-1 * time.Hour)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int64(1800),
		Statistics: []*string{aws.String("Sum")},
	})

	require.NoError(t, err)

	if len(networkMetrics.Datapoints) > 0 {
		totalBytes := 0.0
		for _, datapoint := range networkMetrics.Datapoints {
			if datapoint.Sum != nil {
				totalBytes += *datapoint.Sum
			}
		}

		// Convert bytes to GB
		totalGB := totalBytes / (1024 * 1024 * 1024)
		t.Logf("Bastion total network out: %.2f GB in last hour", totalGB)

		// Assert reasonable data transfer for bastion (should be low)
		assert.Less(t, totalGB, float64(0.5), "Bastion data transfer should be minimal for cost optimization")
	}
}

func TestBastionCostOptimizationSpotInstances(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":          "cost-test",
			"vpc_cidr":             "172.16.0.0/16",
			"azs":                  []string{"us-east-1a"},
			"public_subnet_cidrs":  []string{"172.16.1.0/24"},
			"private_subnet_cidrs": []string{"172.16.10.0/24"},
			"key_name":             "cost-test-key",
			"public_key":           "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsH1rKj8L9q5QJvXc cost-test",
			"allowed_ssh_cidrs":    []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify instance types suitable for Spot Instances
	bastionInstanceType := terraform.Output(t, terraformOptions, "bastion_instance_type")
	privateInstanceType := terraform.Output(t, terraformOptions, "private_instance_type")

	// t3.micro is generally available as Spot Instances
	assert.Equal(t, "t3.micro", bastionInstanceType, "t3.micro is suitable for Spot Instances")
	assert.Equal(t, "t3.micro", privateInstanceType, "t3.micro is suitable for Spot Instances")

	// Verify instances are configured for potential Spot usage
	// (In production, you might want to use Spot Instances for cost optimization)
	bastionTenancy := terraform.Output(t, terraformOptions, "bastion_tenancy")
	assert.Equal(t, "default", bastionTenancy, "Default tenancy allows Spot Instance usage")

	// Check if instances are in same AZ (important for Spot strategy)
	bastionAZ := terraform.Output(t, terraformOptions, "bastion_availability_zone")
	privateAZ := terraform.Output(t, terraformOptions, "private_instance_availability_zone")

	assert.Equal(t, bastionAZ, privateAZ, "Instances in same AZ optimize Spot Instance strategy")
}
