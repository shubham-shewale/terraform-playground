package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestStaticWebsiteEndToEnd(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "../../",
		Vars: map[string]interface{}{
			"domain_name": "e2e-test.example.com",
		},
	}

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Get the CloudFront domain
	cloudfrontDomain := terraform.Output(t, terraformOptions, "cloudfront_domain")
	assert.NotEmpty(t, cloudfrontDomain)

	// Test HTTPS access
	resp, err := http.Get(fmt.Sprintf("https://%s", cloudfrontDomain))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	// Test security headers
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "text/html")

	// Test HTTP to HTTPS redirect
	httpResp, err := http.Get(fmt.Sprintf("http://%s", cloudfrontDomain))
	if err == nil {
		defer httpResp.Body.Close()
		assert.Equal(t, 301, httpResp.StatusCode)
	}
}
