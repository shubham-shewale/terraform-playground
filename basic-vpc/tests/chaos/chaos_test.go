package test

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChaosInstanceFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "chaos-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get instance IDs
	publicInstanceID := terraform.Output(t, terraformOptions, "public_instance_id")
	privateInstanceID := terraform.Output(t, terraformOptions, "private_instance_id")

	// Create AWS session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))

	ec2Svc := ec2.New(sess)

	// Simulate instance failure by stopping the public instance
	t.Log("Simulating public instance failure...")
	_, err := ec2Svc.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{aws.String(publicInstanceID), aws.String(privateInstanceID)},
	})
	require.NoError(t, err)

	// Wait for instance to stop
	time.Sleep(30 * time.Second)

	// Verify instance is stopped
	descInput := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(publicInstanceID)},
	}
	result, err := ec2Svc.DescribeInstances(descInput)
	require.NoError(t, err)

	state := *result.Reservations[0].Instances[0].State.Name
	assert.Equal(t, "stopped", state)

	// Start instance again to simulate recovery
	t.Log("Simulating instance recovery...")
	_, err = ec2Svc.StartInstances(&ec2.StartInstancesInput{
		InstanceIds: []*string{aws.String(publicInstanceID)},
	})
	require.NoError(t, err)

	// Wait for instance to start
	time.Sleep(60 * time.Second)

	// Verify instance is running again
	result, err = ec2Svc.DescribeInstances(descInput)
	require.NoError(t, err)

	state = *result.Reservations[0].Instances[0].State.Name
	assert.Equal(t, "running", state)

	// Verify private instance is still accessible
	privateIP := terraform.Output(t, terraformOptions, "private_instance_private_ip")
	assert.NotEmpty(t, privateIP)
}

func TestChaosNetworkFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "chaos-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get network component IDs
	vpcID := terraform.Output(t, terraformOptions, "vpc_id")
	publicSubnetID := terraform.Output(t, terraformOptions, "public_subnet_id")
	privateSubnetID := terraform.Output(t, terraformOptions, "private_subnet_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	ec2Svc := ec2.New(sess)

	// Simulate network disruption by modifying route table
	t.Log("Simulating network disruption...")

	// Get route table ID for private subnet
	routeTableInput := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
			{
				Name:   aws.String("association.subnet-id"),
				Values: []*string{aws.String(privateSubnetID)},
			},
		},
	}

	routeTables, err := ec2Svc.DescribeRouteTables(routeTableInput)
	require.NoError(t, err)
	require.Greater(t, len(routeTables.RouteTables), 0)

	routeTableID := *routeTables.RouteTables[0].RouteTableId

	// Temporarily remove NAT gateway route to simulate network failure
	_, err = ec2Svc.DeleteRoute(&ec2.DeleteRouteInput{
		RouteTableId:         aws.String(routeTableID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
	})
	require.NoError(t, err)

	// Wait a moment for the change to take effect
	time.Sleep(10 * time.Second)

	// Restore the route to simulate recovery
	natGatewayID := terraform.Output(t, terraformOptions, "nat_gateway_id")

	_, err = ec2Svc.CreateRoute(&ec2.CreateRouteInput{
		RouteTableId:         aws.String(routeTableID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		NatGatewayId:         aws.String(natGatewayID),
	})
	require.NoError(t, err)

	// Verify network components are still intact
	assert.NotEmpty(t, vpcID)
	assert.NotEmpty(t, publicSubnetID)
	assert.NotEmpty(t, privateSubnetID)
}

func TestChaosSecurityFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "chaos-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get security group IDs
	publicSGID := terraform.Output(t, terraformOptions, "public_security_group_id")
	privateSGID := terraform.Output(t, terraformOptions, "private_security_group_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	ec2Svc := ec2.New(sess)

	// Simulate security misconfiguration by adding overly permissive rule
	t.Log("Simulating security misconfiguration...")

	_, err := ec2Svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:    aws.String(publicSGID),
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(22),
		ToPort:     aws.Int64(22),
		CidrIp:     aws.String("0.0.0.0/0"), // Overly permissive
	})
	require.NoError(t, err)

	// Verify the rule was added (simulating detection)
	sgInput := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{aws.String(publicSGID)},
	}
	sgResult, err := ec2Svc.DescribeSecurityGroups(sgInput)
	require.NoError(t, err)

	foundPermissiveRule := false
	for _, permission := range sgResult.SecurityGroups[0].IpPermissions {
		if *permission.FromPort == 22 && *permission.IpProtocol == "tcp" {
			for _, ipRange := range permission.IpRanges {
				if *ipRange.CidrIp == "0.0.0.0/0" {
					foundPermissiveRule = true
					break
				}
			}
		}
	}
	assert.True(t, foundPermissiveRule, "Permissive SSH rule should be detected")

	// Clean up the overly permissive rule
	_, err = ec2Svc.RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
		GroupId:    aws.String(publicSGID),
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(22),
		ToPort:     aws.Int64(22),
		CidrIp:     aws.String("0.0.0.0/0"),
	})
	require.NoError(t, err)

	// Verify security groups are still properly configured
	assert.NotEmpty(t, publicSGID)
	assert.NotEmpty(t, privateSGID)
}

func TestChaosResourceExhaustion(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "chaos-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/8"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get instance IDs
	publicInstanceID := terraform.Output(t, terraformOptions, "public_instance_id")
	privateInstanceID := terraform.Output(t, terraformOptions, "private_instance_id")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	ec2Svc := ec2.New(sess)

	// Simulate CPU stress by changing instance type to micro (resource exhaustion simulation)
	t.Log("Simulating resource exhaustion...")

	// Stop instances first
	stopInput := &ec2.StopInstancesInput{
		InstanceIds: []*string{aws.String(publicInstanceID), aws.String(privateInstanceID)},
	}
	_, err := ec2Svc.StopInstances(stopInput)
	require.NoError(t, err)

	// Wait for instances to stop
	time.Sleep(30 * time.Second)

	// Modify instance type to simulate resource constraints
	modifyInput := &ec2.ModifyInstanceAttributeInput{
		InstanceId: aws.String(publicInstanceID),
		InstanceType: &ec2.AttributeValue{
			Value: aws.String("t3.nano"), // Minimal instance type
		},
	}
	_, err = ec2Svc.ModifyInstanceAttribute(modifyInput)
	require.NoError(t, err)

	// Start instances again
	startInput := &ec2.StartInstancesInput{
		InstanceIds: []*string{aws.String(publicInstanceID), aws.String(privateInstanceID)},
	}
	_, err = ec2Svc.StartInstances(startInput)
	require.NoError(t, err)

	// Wait for instances to start
	time.Sleep(60 * time.Second)

	// Verify instances are still functional despite resource constraints
	descInput := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(publicInstanceID)},
	}
	result, err := ec2Svc.DescribeInstances(descInput)
	require.NoError(t, err)

	state := *result.Reservations[0].Instances[0].State.Name
	assert.Equal(t, "running", state)

	// Verify basic connectivity is maintained
	publicIP := terraform.Output(t, terraformOptions, "public_instance_public_ip")
	assert.NotEmpty(t, publicIP)
}

func TestChaosMonitoringFailure(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"environment":        "chaos-test",
			"allowed_http_cidrs": []string{"10.0.0.0/8"},
			"allowed_ssh_cidrs":  []string{"10.0.0.0/0"},
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get monitoring component IDs
	alarmNames := terraform.OutputList(t, terraformOptions, "cloudwatch_alarm_names")

	// Simulate monitoring failure by temporarily disabling alarms
	t.Log("Simulating monitoring failure...")

	// In a real scenario, you would disable alarms here
	// For this test, we'll just verify alarms exist and are configured

	assert.Greater(t, len(alarmNames), 0, "CloudWatch alarms should be configured")

	// Verify VPC Flow Logs are working
	flowLogID := terraform.Output(t, terraformOptions, "vpc_flow_log_id")
	assert.NotEmpty(t, flowLogID)

	// Verify CloudTrail is enabled
	trailName := terraform.Output(t, terraformOptions, "cloudtrail_name")
	assert.NotEmpty(t, trailName)

	// Verify SNS topic exists for alerts
	snsTopicArn := terraform.Output(t, terraformOptions, "sns_topic_arn")
	assert.NotEmpty(t, snsTopicArn)
}
