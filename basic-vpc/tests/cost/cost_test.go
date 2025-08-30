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

func TestCostOptimizationInstanceSizing(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "cost-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify optimal instance types
	publicInstanceType := terraform.Output(t, terraformOptions, "public_instance_type")
	privateInstanceType := terraform.Output(t, terraformOptions, "private_instance_type")

	// Assert cost-effective instance types
	assert.Equal(t, "t3.micro", publicInstanceType, "Should use cost-effective t3.micro instance")
	assert.Equal(t, "t3.micro", privateInstanceType, "Should use cost-effective t3.micro instance")

	// Verify instances are using gp3 volumes (more cost-effective than gp2)
	publicVolumeType := terraform.Output(t, terraformOptions, "public_instance_volume_type")
	privateVolumeType := terraform.Output(t, terraformOptions, "private_instance_volume_type")

	assert.Equal(t, "gp3", publicVolumeType, "Should use cost-effective gp3 volumes")
	assert.Equal(t, "gp3", privateVolumeType, "Should use cost-effective gp3 volumes")
}

func TestCostOptimizationResourceUtilization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "cost-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
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

	// Check CPU utilization over time
	instances := []string{publicInstanceID, privateInstanceID}
	instanceNames := []string{"public", "private"}

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

				// Cost optimization: instances should not be over-provisioned
				assert.Less(t, avgCPU, float64(30), "Average CPU utilization should be under 30% for cost optimization")
				assert.Less(t, maxValue, float64(80), "Max CPU utilization should be under 80% for cost optimization")
			}
		}
	}
}

func TestCostOptimizationUnusedResources(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "cost-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify no unused Elastic IPs
	eipCount := terraform.Output(t, terraformOptions, "elastic_ip_count")
	assert.Equal(t, "1", eipCount, "Should only have 1 EIP (for NAT Gateway)")

	// Verify NAT Gateway is being used (not idle)
	natGatewayID := terraform.Output(t, terraformOptions, "nat_gateway_id")
	assert.NotEmpty(t, natGatewayID, "NAT Gateway should exist for private subnet egress")

	// Verify VPC Endpoints are configured (cost-effective alternative to NAT for AWS services)
	ssmEndpointID := terraform.Output(t, terraformOptions, "ssm_vpc_endpoint_id")
	ec2MessagesEndpointID := terraform.Output(t, terraformOptions, "ec2messages_vpc_endpoint_id")
	ssmMessagesEndpointID := terraform.Output(t, terraformOptions, "ssmmessages_vpc_endpoint_id")

	assert.NotEmpty(t, ssmEndpointID, "SSM VPC Endpoint should be configured for cost optimization")
	assert.NotEmpty(t, ec2MessagesEndpointID, "EC2Messages VPC Endpoint should be configured")
	assert.NotEmpty(t, ssmMessagesEndpointID, "SSMMessages VPC Endpoint should be configured")
}

func TestCostOptimizationStorageOptimization(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "cost-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify EBS volume sizes are reasonable
	publicVolumeSize := terraform.Output(t, terraformOptions, "public_instance_volume_size")
	privateVolumeSize := terraform.Output(t, terraformOptions, "private_instance_volume_size")

	publicSize := 20  // Default 20GB
	privateSize := 20 // Default 20GB

	assert.Equal(t, publicSize, publicVolumeSize, "Should use minimal volume size for cost optimization")
	assert.Equal(t, privateSize, privateVolumeSize, "Should use minimal volume size for cost optimization")

	// Verify encryption is enabled (no additional cost for EBS encryption)
	publicEncrypted := terraform.Output(t, terraformOptions, "public_instance_encrypted")
	privateEncrypted := terraform.Output(t, terraformOptions, "private_instance_encrypted")

	assert.Equal(t, "true", publicEncrypted, "EBS encryption should be enabled")
	assert.Equal(t, "true", privateEncrypted, "EBS encryption should be enabled")
}

func TestCostOptimizationMonitoringCosts(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "cost-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify detailed monitoring is enabled (minimal cost impact)
	publicMonitoring := terraform.Output(t, terraformOptions, "public_instance_monitoring")
	privateMonitoring := terraform.Output(t, terraformOptions, "private_instance_monitoring")

	assert.Equal(t, "true", publicMonitoring, "Detailed monitoring should be enabled")
	assert.Equal(t, "true", privateMonitoring, "Detailed monitoring should be enabled")

	// Verify CloudWatch log retention is reasonable
	vpcFlowLogRetention := terraform.Output(t, terraformOptions, "vpc_flow_log_retention_days")
	assert.Equal(t, "30", vpcFlowLogRetention, "Log retention should be 30 days for cost optimization")

	// Verify CloudTrail is configured but not excessive
	trailName := terraform.Output(t, terraformOptions, "cloudtrail_name")
	assert.NotEmpty(t, trailName, "CloudTrail should be configured for auditing")

	// Verify SNS topic exists for alerts (cost-effective notification system)
	snsTopicArn := terraform.Output(t, terraformOptions, "sns_topic_arn")
	assert.NotEmpty(t, snsTopicArn, "SNS topic should exist for cost-effective alerting")
}

func TestCostOptimizationDataTransfer(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "cost-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	publicInstanceID := terraform.Output(t, terraformOptions, "public_instance_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	cloudwatchSvc := cloudwatch.New(sess)

	// Check network data transfer costs
	networkMetrics, err := cloudwatchSvc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("NetworkOut"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(publicInstanceID),
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
		t.Logf("Total network out: %.2f GB in last hour", totalGB)

		// Assert reasonable data transfer (under 1GB/hour for test environment)
		assert.Less(t, totalGB, float64(1), "Data transfer should be minimal for cost optimization")
	}
}

func TestCostOptimizationReservedInstances(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "cost-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Verify instance types that would benefit from Reserved Instances
	publicInstanceType := terraform.Output(t, terraformOptions, "public_instance_type")
	privateInstanceType := terraform.Output(t, terraformOptions, "private_instance_type")

	// t3.micro is a good candidate for Reserved Instances in production
	assert.Equal(t, "t3.micro", publicInstanceType, "t3.micro is suitable for Reserved Instances")
	assert.Equal(t, "t3.micro", privateInstanceType, "t3.micro is suitable for Reserved Instances")

	// Verify consistent instance types (important for RI planning)
	assert.Equal(t, publicInstanceType, privateInstanceType, "Consistent instance types enable better RI utilization")

	// Check if instances are in the same AZ (important for RI planning)
	publicAZ := terraform.Output(t, terraformOptions, "public_instance_availability_zone")
	privateAZ := terraform.Output(t, terraformOptions, "private_instance_availability_zone")

	assert.Equal(t, publicAZ, privateAZ, "Instances in same AZ enable better RI utilization")
}
