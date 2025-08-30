package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestCloudWatchAlarms(t *testing.T) {
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

	// Test CPU utilization alarms
	publicCpuAlarmName := terraform.Output(t, terraformOptions, "public_cpu_alarm_name")
	assert.Contains(t, publicCpuAlarmName, "cpu-utilization-public-test")

	privateCpuAlarmName := terraform.Output(t, terraformOptions, "private_cpu_alarm_name")
	assert.Contains(t, privateCpuAlarmName, "cpu-utilization-private-test")

	// Test Network alarms
	publicNetworkAlarmName := terraform.Output(t, terraformOptions, "public_network_alarm_name")
	assert.Contains(t, publicNetworkAlarmName, "network-in-public-test")

	// Test Status Check alarms
	publicStatusAlarmName := terraform.Output(t, terraformOptions, "public_status_alarm_name")
	assert.Contains(t, publicStatusAlarmName, "status-check-public-test")
}

func TestCloudWatchAlarmConfiguration(t *testing.T) {
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

	// Test alarm thresholds
	cpuThreshold := terraform.Output(t, terraformOptions, "cpu_alarm_threshold")
	assert.Equal(t, "80", cpuThreshold)

	networkThreshold := terraform.Output(t, terraformOptions, "network_alarm_threshold")
	assert.Equal(t, "1000000", networkThreshold)

	statusThreshold := terraform.Output(t, terraformOptions, "status_alarm_threshold")
	assert.Equal(t, "0", statusThreshold)

	// Test alarm evaluation periods
	alarmEvaluationPeriods := terraform.Output(t, terraformOptions, "alarm_evaluation_periods")
	assert.Equal(t, "2", alarmEvaluationPeriods)
}

func TestCloudWatchDashboard(t *testing.T) {
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

	// Test dashboard creation
	dashboardName := terraform.Output(t, terraformOptions, "cloudwatch_dashboard_name")
	assert.Contains(t, dashboardName, "security-dashboard-test")

	// Test dashboard widgets
	dashboardWidgets := terraform.OutputList(t, terraformOptions, "dashboard_widgets")
	assert.Greater(t, len(dashboardWidgets), 0)

	// Test dashboard has CPU and Network widgets
	dashboardHasCpuWidget := terraform.Output(t, terraformOptions, "dashboard_has_cpu_widget")
	assert.Equal(t, "true", dashboardHasCpuWidget)

	dashboardHasNetworkWidget := terraform.Output(t, terraformOptions, "dashboard_has_network_widget")
	assert.Equal(t, "true", dashboardHasNetworkWidget)
}

func TestSnsTopic(t *testing.T) {
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

	// Test SNS topic creation
	snsTopicArn := terraform.Output(t, terraformOptions, "sns_topic_arn")
	assert.NotEmpty(t, snsTopicArn)
	assert.Contains(t, snsTopicArn, "security-alerts-test")

	// Test SNS topic policy
	snsTopicPolicyAttached := terraform.Output(t, terraformOptions, "sns_topic_policy_attached")
	assert.Equal(t, "true", snsTopicPolicyAttached)

	// Test CloudWatch can publish to SNS
	snsAllowsCloudWatch := terraform.Output(t, terraformOptions, "sns_allows_cloudwatch")
	assert.Equal(t, "true", snsAllowsCloudWatch)
}
