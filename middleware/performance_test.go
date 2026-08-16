package middleware

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemPerformanceRequiresSustainedCPUOverload(t *testing.T) {
	config := common.PerformanceMonitorConfig{Enabled: true, CPUThreshold: 90}

	assert.Nil(t, checkSystemPerformanceStatus(config, common.SystemStatus{CPUUsage: 99, CPUOverloadSamples: 1}))
	err := checkSystemPerformanceStatus(config, common.SystemStatus{CPUUsage: 99, CPUOverloadSamples: 2})
	require.NotNil(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, err.StatusCode)
	assert.Equal(t, "system_cpu_overloaded", string(err.GetErrorCode()))
}
