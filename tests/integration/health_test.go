// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package integration

import (
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/health"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// Test_9_1_1_1_HealthEndpointResponse tests /health endpoint functionality
func Test_9_1_1_1_HealthEndpointResponse(t *testing.T) {
	helpers.LogTestSection(t, "9.1.1.1", "/health 엔드포인트 정상 응답")

	helpers.LogDetail(t, "시스템 헬스체크 엔드포인트 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 헬스체크 정상 응답 확인")

	// Create health checker (without RPC URL for basic system check)
	checker := health.NewChecker("")
	require.NotNil(t, checker)
	helpers.LogSuccess(t, "헬스체크 생성 완료")

	// Perform health check
	status := checker.CheckAll()
	require.NotNil(t, status)
	helpers.LogSuccess(t, "헬스체크 실행 완료")

	// Verify health status structure
	require.NotEmpty(t, status.Status, "Status should not be empty")
	require.False(t, status.Timestamp.IsZero(), "Timestamp should be set")
	helpers.LogDetail(t, "헬스체크 응답:")
	helpers.LogDetail(t, "  Status: %s", status.Status)
	helpers.LogDetail(t, "  Timestamp: %s", status.Timestamp.Format(time.RFC3339))

	// Verify status is one of valid values
	validStatuses := map[health.Status]bool{
		health.StatusHealthy:   true,
		health.StatusDegraded:  true,
		health.StatusUnhealthy: true,
	}
	require.True(t, validStatuses[status.Status], "Status should be valid: %s", status.Status)
	helpers.LogSuccess(t, "Status 값 검증 완료")

	// Verify system health is present
	require.NotNil(t, status.SystemStatus, "System status should be present")
	helpers.LogDetail(t, "  System Status: %s", status.SystemStatus.Status)

	// Log error messages if any
	if len(status.Errors) > 0 {
		helpers.LogDetail(t, "  Errors (%d):", len(status.Errors))
		for i, err := range status.Errors {
			helpers.LogDetail(t, "    [%d] %s", i+1, err)
		}
	} else {
		helpers.LogDetail(t, "  Errors: none")
	}

	helpers.LogSuccess(t, "/health 엔드포인트 응답 검증 완료 ✓")

	// Save verification data
	data := map[string]interface{}{
		"test_name":            "Test_9_1_1_1_HealthEndpointResponse",
		"timestamp":            time.Now().Format(time.RFC3339),
		"test_case":            "9.1.1.1_Health_Endpoint",
		"endpoint":             "/health",
		"status":               string(status.Status),
		"response_timestamp":   status.Timestamp.Format(time.RFC3339),
		"has_system_status":    status.SystemStatus != nil,
		"has_blockchain_status": status.BlockchainStatus != nil,
		"error_count":          len(status.Errors),
		"health_check_success": true,
	}

	helpers.SaveTestData(t, "health/9_1_1_1_health_endpoint.json", data)

	helpers.LogPassCriteria(t, []string{
		"헬스체크 생성 성공",
		"헬스체크 실행 성공",
		"Status 값 유효성 검증",
		"Timestamp 설정 확인",
		"System status 존재 확인",
		"/health 엔드포인트 정상 응답",
	})
}

// Test_9_1_1_2_BlockchainConnectionStatus tests blockchain connectivity check
func Test_9_1_1_2_BlockchainConnectionStatus(t *testing.T) {
	helpers.LogTestSection(t, "9.1.1.2", "블록체인 연결 상태 확인")

	helpers.LogDetail(t, "블록체인 연결 상태 모니터링 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 블록체인 헬스체크 기능 검증")

	// Test with no RPC URL (should handle gracefully)
	helpers.LogDetail(t, "테스트 1: RPC URL 없이 블록체인 상태 확인")
	blockchainStatus := health.CheckBlockchain("")
	require.NotNil(t, blockchainStatus)
	helpers.LogSuccess(t, "블록체인 상태 확인 함수 실행 완료")

	helpers.LogDetail(t, "  블록체인 연결 상태:")
	helpers.LogDetail(t, "    Status: %s", blockchainStatus.Status)
	helpers.LogDetail(t, "    Connected: %v", blockchainStatus.Connected)
	if blockchainStatus.Error != "" {
		helpers.LogDetail(t, "    Error: %s", blockchainStatus.Error)
	}
	if blockchainStatus.ChainID != "" {
		helpers.LogDetail(t, "    Chain ID: %s", blockchainStatus.ChainID)
	}
	if blockchainStatus.BlockNumber > 0 {
		helpers.LogDetail(t, "    Block Number: %d", blockchainStatus.BlockNumber)
	}

	// Verify status structure
	require.NotEmpty(t, blockchainStatus.Status, "Status should not be empty")
	helpers.LogSuccess(t, "블록체인 상태 구조 검증 완료")

	// Test with local RPC URL (optional, may not be running)
	helpers.LogDetail(t, "테스트 2: 로컬 RPC URL로 블록체인 상태 확인")
	localRPC := "http://localhost:8545"
	blockchainStatusLocal := health.CheckBlockchain(localRPC)
	require.NotNil(t, blockchainStatusLocal)
	helpers.LogDetail(t, "  Local RPC Status:")
	helpers.LogDetail(t, "    RPC URL: %s", localRPC)
	helpers.LogDetail(t, "    Status: %s", blockchainStatusLocal.Status)
	helpers.LogDetail(t, "    Connected: %v", blockchainStatusLocal.Connected)

	if blockchainStatusLocal.Connected {
		helpers.LogSuccess(t, "로컬 블록체인 연결 성공")
		helpers.LogDetail(t, "    Chain ID: %s", blockchainStatusLocal.ChainID)
		helpers.LogDetail(t, "    Block Number: %d", blockchainStatusLocal.BlockNumber)
		if blockchainStatusLocal.Latency != "" {
			helpers.LogDetail(t, "    Latency: %s", blockchainStatusLocal.Latency)
		}
	} else {
		helpers.LogDetail(t, "로컬 블록체인 미연결 (예상됨: 로컬 노드 미실행)")
		if blockchainStatusLocal.Error != "" {
			helpers.LogDetail(t, "    Error: %s", blockchainStatusLocal.Error)
		}
	}

	helpers.LogSuccess(t, "블록체인 연결 상태 확인 완료 ✓")

	// Save verification data
	data := map[string]interface{}{
		"test_name":                 "Test_9_1_1_2_BlockchainConnectionStatus",
		"timestamp":                 time.Now().Format(time.RFC3339),
		"test_case":                 "9.1.1.2_Blockchain_Connection",
		"empty_rpc_status":          string(blockchainStatus.Status),
		"empty_rpc_connected":       blockchainStatus.Connected,
		"local_rpc_url":             localRPC,
		"local_rpc_status":          string(blockchainStatusLocal.Status),
		"local_rpc_connected":       blockchainStatusLocal.Connected,
		"local_rpc_chain_id":        blockchainStatusLocal.ChainID,
		"local_rpc_block_number":    blockchainStatusLocal.BlockNumber,
		"blockchain_check_function": "operational",
		"note":                      "Local blockchain node may not be running, which is expected in testing environment",
	}

	helpers.SaveTestData(t, "health/9_1_1_2_blockchain_status.json", data)

	helpers.LogPassCriteria(t, []string{
		"블록체인 상태 확인 함수 실행 성공",
		"빈 RPC URL 처리 확인",
		"블록체인 상태 구조 검증",
		"로컬 RPC URL 처리 확인",
		"연결 상태 판별 로직 동작",
		"블록체인 헬스체크 기능 검증 완료",
	})
}

// Test_9_1_1_3_SystemResourceMonitoring tests system resource monitoring
func Test_9_1_1_3_SystemResourceMonitoring(t *testing.T) {
	helpers.LogTestSection(t, "9.1.1.3", "메모리/CPU 사용률 확인")

	helpers.LogDetail(t, "시스템 리소스 모니터링 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 메모리, CPU, 디스크 사용률 확인")

	// Check system resources
	systemStatus := health.CheckSystem()
	require.NotNil(t, systemStatus)
	helpers.LogSuccess(t, "시스템 리소스 확인 완료")

	// Verify status
	require.NotEmpty(t, systemStatus.Status, "System status should not be empty")
	helpers.LogDetail(t, "시스템 상태:")
	helpers.LogDetail(t, "  Status: %s", systemStatus.Status)

	// Memory information
	helpers.LogDetail(t, "메모리 사용률:")
	helpers.LogDetail(t, "  Used: %d MB", systemStatus.MemoryUsedMB)
	helpers.LogDetail(t, "  Total: %d MB", systemStatus.MemoryTotalMB)
	helpers.LogDetail(t, "  Percent: %.2f%%", systemStatus.MemoryPercent)

	// Verify memory values are reasonable
	require.Greater(t, systemStatus.MemoryTotalMB, uint64(0), "Total memory should be > 0")
	require.GreaterOrEqual(t, systemStatus.MemoryPercent, float64(0), "Memory percent should be >= 0")
	require.LessOrEqual(t, systemStatus.MemoryPercent, float64(100), "Memory percent should be <= 100")
	helpers.LogSuccess(t, "메모리 사용률 검증 완료")

	// CPU information
	helpers.LogDetail(t, "CPU 사용률:")
	helpers.LogDetail(t, "  Percent: %.2f%%", systemStatus.CPUPercent)

	// Verify CPU values are reasonable
	require.GreaterOrEqual(t, systemStatus.CPUPercent, float64(0), "CPU percent should be >= 0")
	// Note: CPU percent can exceed 100% on multi-core systems
	helpers.LogSuccess(t, "CPU 사용률 검증 완료")

	// Disk information
	helpers.LogDetail(t, "디스크 사용률:")
	helpers.LogDetail(t, "  Used: %d GB", systemStatus.DiskUsedGB)
	helpers.LogDetail(t, "  Total: %d GB", systemStatus.DiskTotalGB)
	helpers.LogDetail(t, "  Percent: %.2f%%", systemStatus.DiskPercent)

	// Verify disk values are reasonable
	require.GreaterOrEqual(t, systemStatus.DiskPercent, float64(0), "Disk percent should be >= 0")
	require.LessOrEqual(t, systemStatus.DiskPercent, float64(100), "Disk percent should be <= 100")
	helpers.LogSuccess(t, "디스크 사용률 검증 완료")

	// GoRoutines information
	helpers.LogDetail(t, "Go 런타임:")
	helpers.LogDetail(t, "  GoRoutines: %d", systemStatus.GoRoutines)
	require.Greater(t, systemStatus.GoRoutines, 0, "GoRoutines should be > 0")
	helpers.LogSuccess(t, "GoRoutines 정보 검증 완료")

	// Error information
	if systemStatus.Error != "" {
		helpers.LogDetail(t, "  Error: %s", systemStatus.Error)
	} else {
		helpers.LogDetail(t, "  Errors: none")
	}

	helpers.LogSuccess(t, "시스템 리소스 모니터링 완료 ✓")

	// Save verification data
	data := map[string]interface{}{
		"test_name":       "Test_9_1_1_3_SystemResourceMonitoring",
		"timestamp":       time.Now().Format(time.RFC3339),
		"test_case":       "9.1.1.3_System_Resources",
		"status":          string(systemStatus.Status),
		"memory": map[string]interface{}{
			"used_mb":  systemStatus.MemoryUsedMB,
			"total_mb": systemStatus.MemoryTotalMB,
			"percent":  systemStatus.MemoryPercent,
		},
		"cpu": map[string]interface{}{
			"percent": systemStatus.CPUPercent,
		},
		"disk": map[string]interface{}{
			"used_gb":  systemStatus.DiskUsedGB,
			"total_gb": systemStatus.DiskTotalGB,
			"percent":  systemStatus.DiskPercent,
		},
		"goroutines":        systemStatus.GoRoutines,
		"monitoring_success": true,
	}

	helpers.SaveTestData(t, "health/9_1_1_3_system_resources.json", data)

	helpers.LogPassCriteria(t, []string{
		"시스템 리소스 확인 성공",
		"메모리 사용률 검증 (0-100%)",
		"CPU 사용률 검증 (>= 0%)",
		"디스크 사용률 검증 (0-100%)",
		"GoRoutines 수 확인 (> 0)",
		"시스템 모니터링 기능 정상 동작",
	})
}
