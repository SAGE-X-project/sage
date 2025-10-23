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

package order

import (
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/message"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

func TestResultBuilder(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		// 명세 요구사항: 빌더의 기본값 검증
		helpers.LogTestSection(t, "8.1.6", "결과 빌더 기본값 검증")

		helpers.LogDetail(t, "결과 빌더 생성 중")
		builder := NewResultBuilder()
		helpers.LogSuccess(t, "빌더 생성 완료")

		// 명세 요구사항: 기본 상태로 빌드
		helpers.LogDetail(t, "기본 설정으로 결과 빌드")
		res := builder.Build()
		helpers.LogSuccess(t, "결과 빌드 완료")

		// 명세 요구사항: 모든 기본값이 false/empty여야 함
		require.False(t, res.IsProcessed, "기본 IsProcessed는 false여야 함")
		require.False(t, res.IsDuplicate, "기본 IsDuplicate는 false여야 함")
		require.False(t, res.IsWaiting, "기본 IsWaiting는 false여야 함")
		require.Empty(t, res.ReadyMessages, "기본 ReadyMessages는 빈 슬라이스여야 함")

		helpers.LogSuccess(t, "모든 기본값 검증 완료")
		helpers.LogDetail(t, "결과 상태:")
		helpers.LogDetail(t, "  IsProcessed: %v", res.IsProcessed)
		helpers.LogDetail(t, "  IsDuplicate: %v", res.IsDuplicate)
		helpers.LogDetail(t, "  IsWaiting: %v", res.IsWaiting)
		helpers.LogDetail(t, "  ReadyMessages 개수: %d", len(res.ReadyMessages))

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"빌더가 성공적으로 생성됨",
			"기본 상태로 결과 빌드 완료",
			"IsProcessed 기본값 = false",
			"IsDuplicate 기본값 = false",
			"IsWaiting 기본값 = false",
			"ReadyMessages 기본값 = 빈 슬라이스",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "8.1.6_결과_빌더_기본값",
			"result": map[string]interface{}{
				"is_processed":         res.IsProcessed,
				"is_duplicate":         res.IsDuplicate,
				"is_waiting":           res.IsWaiting,
				"ready_messages_count": len(res.ReadyMessages),
			},
			"validation": "기본값_검증_통과",
		}
		helpers.SaveTestData(t, "message/order/result_builder_defaults.json", testData)
	})

	t.Run("WithProcessed", func(t *testing.T) {
		// 명세 요구사항: 처리됨 상태 설정 검증
		helpers.LogTestSection(t, "8.1.7", "결과 빌더 처리됨 상태 설정")

		helpers.LogDetail(t, "WithProcessed(true)로 빌더 설정")
		res := NewResultBuilder().WithProcessed(true).Build()

		require.True(t, res.IsProcessed, "WithProcessed(true)는 IsProcessed=true로 설정해야 함")
		helpers.LogSuccess(t, "처리됨 상태 설정 성공")
		helpers.LogDetail(t, "IsProcessed: %v", res.IsProcessed)

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"WithProcessed(true) 메서드 호출 성공",
			"IsProcessed = true로 설정됨",
			"빌더 패턴이 올바르게 동작함",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "8.1.7_결과_빌더_처리됨_상태",
			"setting": map[string]interface{}{
				"method": "WithProcessed",
				"value":  true,
			},
			"result": map[string]interface{}{
				"is_processed": res.IsProcessed,
			},
		}
		helpers.SaveTestData(t, "message/order/result_builder_with_processed.json", testData)
	})

	t.Run("WithDuplicate", func(t *testing.T) {
		// 명세 요구사항: 중복 상태 설정 검증
		helpers.LogTestSection(t, "8.1.8", "결과 빌더 중복 상태 설정")

		helpers.LogDetail(t, "WithDuplicate(true)로 빌더 설정")
		res := NewResultBuilder().WithDuplicate(true).Build()

		require.True(t, res.IsDuplicate, "WithDuplicate(true)는 IsDuplicate=true로 설정해야 함")
		helpers.LogSuccess(t, "중복 상태 설정 성공")
		helpers.LogDetail(t, "IsDuplicate: %v", res.IsDuplicate)

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"WithDuplicate(true) 메서드 호출 성공",
			"IsDuplicate = true로 설정됨",
			"중복 메시지 표시 기능 동작 확인",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "8.1.8_결과_빌더_중복_상태",
			"setting": map[string]interface{}{
				"method": "WithDuplicate",
				"value":  true,
			},
			"result": map[string]interface{}{
				"is_duplicate": res.IsDuplicate,
			},
		}
		helpers.SaveTestData(t, "message/order/result_builder_with_duplicate.json", testData)
	})

	t.Run("WithWaiting", func(t *testing.T) {
		// 명세 요구사항: 대기 상태 설정 검증
		helpers.LogTestSection(t, "8.1.9", "결과 빌더 대기 상태 설정")

		helpers.LogDetail(t, "WithWaiting(true)로 빌더 설정")
		res := NewResultBuilder().WithWaiting(true).Build()

		require.True(t, res.IsWaiting, "WithWaiting(true)는 IsWaiting=true로 설정해야 함")
		helpers.LogSuccess(t, "대기 상태 설정 성공")
		helpers.LogDetail(t, "IsWaiting: %v", res.IsWaiting)

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"WithWaiting(true) 메서드 호출 성공",
			"IsWaiting = true로 설정됨",
			"메시지 대기 상태 표시 기능 동작 확인",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "8.1.9_결과_빌더_대기_상태",
			"setting": map[string]interface{}{
				"method": "WithWaiting",
				"value":  true,
			},
			"result": map[string]interface{}{
				"is_waiting": res.IsWaiting,
			},
		}
		helpers.SaveTestData(t, "message/order/result_builder_with_waiting.json", testData)
	})

	t.Run("WithReadyMessages", func(t *testing.T) {
		// 명세 요구사항: 준비된 메시지 목록 설정 검증
		helpers.LogTestSection(t, "8.1.10", "결과 빌더 준비된 메시지 설정")

		now := time.Now()
		seq1 := uint64(1)
		nonce1 := "a"
		seq2 := uint64(2)
		nonce2 := "b"

		helpers.LogDetail(t, "테스트 메시지 헤더 생성:")
		helpers.LogDetail(t, "  메시지 1 - seq=%d, nonce=%s", seq1, nonce1)
		helpers.LogDetail(t, "  메시지 2 - seq=%d, nonce=%s", seq2, nonce2)

		head1 := &mockHeader{seq: seq1, nonce: nonce1, timestamp: now}
		head2 := &mockHeader{seq: seq2, nonce: nonce2, timestamp: now.Add(time.Second)}
		expected := []message.ControlHeader{head1, head2}

		helpers.LogDetail(t, "WithReadyMessages()로 메시지 목록 설정")
		res := NewResultBuilder().WithReadyMessages(expected).Build()

		require.Equal(t, expected, res.ReadyMessages, "WithReadyMessages는 슬라이스를 올바르게 설정해야 함")
		helpers.LogSuccess(t, "준비된 메시지 목록 설정 성공")
		helpers.LogDetail(t, "ReadyMessages 개수: %d", len(res.ReadyMessages))

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"2개의 메시지 헤더 생성 완료",
			"WithReadyMessages() 메서드 호출 성공",
			"ReadyMessages 목록이 올바르게 설정됨",
			"메시지 순서가 유지됨",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "8.1.10_결과_빌더_준비된_메시지",
			"messages": []map[string]interface{}{
				{
					"sequence":  seq1,
					"nonce":     nonce1,
					"timestamp": now.Format(time.RFC3339Nano),
				},
				{
					"sequence":  seq2,
					"nonce":     nonce2,
					"timestamp": now.Add(time.Second).Format(time.RFC3339Nano),
				},
			},
			"result": map[string]interface{}{
				"ready_messages_count": len(res.ReadyMessages),
			},
		}
		helpers.SaveTestData(t, "message/order/result_builder_with_ready_messages.json", testData)
	})

	t.Run("ChainedSettings", func(t *testing.T) {
		// 명세 요구사항: 체이닝된 빌더 메서드 검증
		helpers.LogTestSection(t, "8.1.11", "결과 빌더 메서드 체이닝")

		now := time.Now()
		seq := uint64(3)
		nonce := "c"

		helpers.LogDetail(t, "테스트 메시지 헤더:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Nonce: %s", nonce)

		head := &mockHeader{seq: seq, nonce: nonce, timestamp: now}

		helpers.LogDetail(t, "빌더 메서드 체이닝:")
		helpers.LogDetail(t, "  WithProcessed(true)")
		helpers.LogDetail(t, "  WithDuplicate(true)")
		helpers.LogDetail(t, "  WithWaiting(true)")
		helpers.LogDetail(t, "  WithReadyMessages([1개 메시지])")

		res := NewResultBuilder().
			WithProcessed(true).
			WithDuplicate(true).
			WithWaiting(true).
			WithReadyMessages([]message.ControlHeader{head}).
			Build()

		require.True(t, res.IsProcessed)
		require.True(t, res.IsDuplicate)
		require.True(t, res.IsWaiting)
		require.Len(t, res.ReadyMessages, 1)
		require.Equal(t, head, res.ReadyMessages[0])

		helpers.LogSuccess(t, "메서드 체이닝 성공")
		helpers.LogDetail(t, "최종 결과:")
		helpers.LogDetail(t, "  IsProcessed: %v", res.IsProcessed)
		helpers.LogDetail(t, "  IsDuplicate: %v", res.IsDuplicate)
		helpers.LogDetail(t, "  IsWaiting: %v", res.IsWaiting)
		helpers.LogDetail(t, "  ReadyMessages 개수: %d", len(res.ReadyMessages))

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"빌더 패턴 체이닝이 올바르게 동작",
			"모든 상태 플래그가 true로 설정됨",
			"ReadyMessages에 1개 메시지 포함",
			"메서드 호출 순서가 결과에 영향 없음",
			"빌더 패턴의 유연성 검증 완료",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "8.1.11_결과_빌더_메서드_체이닝",
			"chained_methods": []string{
				"WithProcessed(true)",
				"WithDuplicate(true)",
				"WithWaiting(true)",
				"WithReadyMessages([1개])",
			},
			"result": map[string]interface{}{
				"is_processed":         res.IsProcessed,
				"is_duplicate":         res.IsDuplicate,
				"is_waiting":           res.IsWaiting,
				"ready_messages_count": len(res.ReadyMessages),
			},
			"message": map[string]interface{}{
				"sequence":  seq,
				"nonce":     nonce,
				"timestamp": now.Format(time.RFC3339Nano),
			},
		}
		helpers.SaveTestData(t, "message/order/result_builder_chained.json", testData)
	})
}
