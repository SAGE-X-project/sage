# SAGE 시각 자료 생성 가이드
## 이미지, 다이어그램, 차트 제작 방법

---

## 📋 목차

1. [Mermaid 다이어그램 → 이미지 변환](#1-mermaid-다이어그램--이미지-변환)
2. [아키텍처 다이어그램 제작](#2-아키텍처-다이어그램-제작)
3. [차트 및 그래프](#3-차트-및-그래프)
4. [아이콘 및 로고](#4-아이콘-및-로고)
5. [스크린샷 및 목업](#5-스크린샷-및-목업)
6. [추천 도구](#6-추천-도구)

---

## 1. Mermaid 다이어그램 → 이미지 변환

### 방법 1: Mermaid Live Editor (추천)

**URL**: https://mermaid.live

**절차**:
1. `report-with-visuals.md`에서 Mermaid 코드 복사
2. Mermaid Live Editor에 붙여넣기
3. 우측 상단 "Actions" → "PNG 다운로드" 또는 "SVG 다운로드"
4. PPT에 이미지 삽입

**예시**:
```
report-with-visuals.md의 다이어그램 → Mermaid Live → PNG/SVG → PPT
```

### 방법 2: VS Code 확장 프로그램

**확장 프로그램**: "Markdown Preview Mermaid Support"

**절차**:
1. VS Code에서 확장 프로그램 설치
2. `report-with-visuals.md` 파일 열기
3. Markdown Preview 열기 (Cmd/Ctrl + Shift + V)
4. 다이어그램 우클릭 → "Copy Image" 또는 스크린샷

### 방법 3: CLI 도구 (대량 변환)

```bash
# mermaid-cli 설치
npm install -g @mermaid-js/mermaid-cli

# 마크다운에서 다이어그램 추출하여 이미지 생성
mmdc -i report-with-visuals.md -o diagrams/
```

---

## 2. 아키텍처 다이어그램 제작

### 옵션 1: draw.io (diagrams.net) - 무료, 강력 추천

**URL**: https://app.diagrams.net/

**장점**:
- 무료
- 브라우저에서 실행
- 풍부한 템플릿 및 아이콘
- PNG, SVG, PDF 내보내기

**템플릿**:
```
File → New → 템플릿 선택
- Software Architecture
- Network Diagram
- Flowchart
```

**SAGE 다이어그램 예시**:

#### Trust Layer 다이어그램
```
1. 새 파일 생성
2. 왼쪽 패널에서 "Rectangle" 3개 추가
3. 레이블:
   - 상단: "Application Layer (AI Agents)"
   - 중간: "🔒 SAGE Trust Layer"
   - 하단: "Transport Layer (HTTP/HTTPS)"
4. 색상:
   - 중간 박스: 파란색 (#4DABF7)
5. 화살표 연결
6. File → Export as → PNG (300 DPI)
```

#### 시스템 구성도
```
1. 템플릿: Software Architecture
2. 컴포넌트 추가:
   - SAGE Core (중앙)
   - Blockchain (왼쪽)
   - SDKs (오른쪽)
   - CLI Tools (하단)
3. 화살표로 연결
4. 그룹화 (상자로 묶기)
5. Export → PNG
```

### 옵션 2: Lucidchart - 전문적인 다이어그램

**URL**: https://www.lucidchart.com/

**장점**:
- 전문적인 UI
- 협업 기능
- 템플릿 풍부

**무료 플랜**: 3개 문서까지

### 옵션 3: Excalidraw - 손그림 스타일

**URL**: https://excalidraw.com/

**장점**:
- 손그림 느낌 (친근함)
- 빠른 스케치
- 무료

**용도**: 개념 설명, 간단한 플로우

---

## 3. 차트 및 그래프

### 옵션 1: Chart.js + QuickChart

**URL**: https://quickchart.io/

**무료 차트 생성 서비스**

**예시: 파이 차트 (테스트 커버리지)**

```javascript
// URL 생성
https://quickchart.io/chart?c={
  type: 'pie',
  data: {
    labels: [
      'RFC-9421 (11)',
      '암호화 키 (13)',
      'DID (12)',
      '블록체인 (10)',
      '메시지 (10)',
      'CLI (13)',
      '세션 (6)',
      'HPKE (5)',
      '헬스체크 (3)'
    ],
    datasets: [{
      data: [11, 13, 12, 10, 10, 13, 6, 5, 3],
      backgroundColor: [
        '#FF6B6B', '#4DABF7', '#51CF66', '#FFD43B',
        '#E64980', '#7950F2', '#20C997', '#FA5252', '#868E96'
      ]
    }]
  },
  options: {
    title: {
      display: true,
      text: '명세서 검증 완료 (83개 항목)'
    }
  }
}

// 브라우저에서 열기 → 우클릭 → 이미지 저장
```

**예시: 바 차트 (품질 메트릭)**

```javascript
https://quickchart.io/chart?c={
  type: 'bar',
  data: {
    labels: ['테스트 함수', '테스트 코드 (천 라인)', '문서 (천 라인)'],
    datasets: [{
      label: 'SAGE 품질 지표',
      data: [61, 7.8, 5.1],
      backgroundColor: '#51CF66'
    }]
  }
}
```

### 옵션 2: Google Sheets → Chart

**절차**:
1. Google Sheets에서 데이터 입력
2. 삽입 → 차트
3. 차트 유형 선택 (파이, 바, 라인 등)
4. 차트 우클릭 → "이미지로 다운로드"

**예시 데이터**:
```
| 섹션        | 항목 수 |
|------------|--------|
| RFC-9421   | 11     |
| 암호화 키   | 13     |
| DID 관리   | 12     |
| ...        | ...    |
```

### 옵션 3: Canva - 인포그래픽

**URL**: https://www.canva.com/

**장점**:
- 아름다운 템플릿
- 인포그래픽 제작 쉬움
- 차트 + 아이콘 조합

**템플릿 검색**: "Infographic", "Data Visualization"

---

## 4. 아이콘 및 로고

### 아이콘 소스

#### 1. Flaticon (무료/유료)
**URL**: https://www.flaticon.com/

**검색 키워드**:
- "shield" (방패 - 보안)
- "lock" (자물쇠 - 암호화)
- "blockchain"
- "network"
- "signature"
- "verified" (인증)

**라이선스**: 무료는 출처 표기 필요

#### 2. Font Awesome (무료)
**URL**: https://fontawesome.com/

**아이콘**:
- `fa-shield-alt` (방패)
- `fa-lock` (자물쇠)
- `fa-check-circle` (체크)
- `fa-exclamation-triangle` (경고)

**사용**:
- 웹사이트에서 PNG 다운로드
- 또는 SVG 사용

#### 3. Material Design Icons
**URL**: https://fonts.google.com/icons

**아이콘**:
- security
- verified_user
- block
- check_circle

### SAGE 로고 제작

#### 방법 1: Canva (추천)

**절차**:
1. Canva → "Logo" 템플릿
2. 검색: "Shield Logo" 또는 "Tech Logo"
3. 편집:
   - 텍스트: "SAGE"
   - 부제: "Secure Agent Guarantee Engine"
   - 아이콘: 방패 + 자물쇠 조합
4. 색상: #4DABF7 (파란색)
5. 다운로드: PNG (투명 배경)

#### 방법 2: Figma (전문적)

**URL**: https://www.figma.com/

**템플릿**: "Logo Design"

#### 로고 컨셉
```
┌─────────────────────┐
│   🛡️  🔒           │
│                     │
│      SAGE           │
│                     │
│ Secure Agent        │
│ Guarantee Engine    │
└─────────────────────┘
```

**색상 조합**:
- 방패: #4DABF7 (파란색)
- 자물쇠: #343A40 (검은색)
- 텍스트: #343A40

---

## 5. 스크린샷 및 목업

### Agent Marketplace 목업

#### 도구: Figma 또는 Sketch

**절차**:
1. Figma → 새 디자인 파일
2. 프레임 크기: 1200x800px
3. 컴포넌트 추가:
   - 상단: 검색 바
   - 리스트: Agent 카드
     - ✅ Official Payment Agent
       - DID 정보
       - 평점 ⭐⭐⭐⭐⭐
       - [선택] 버튼
     - ⚠️ Fake Payment Agent
       - DID: 미등록
       - [사용 불가]
4. 색상:
   - 인증됨: 초록색 (#51CF66)
   - 미인증: 빨간색 (#FF6B6B)
5. Export → PNG

#### 빠른 목업: Mockuuups Studio

**URL**: https://mockuuups.studio.design/

**템플릿**: Web Dashboard

### 시연 화면 목업

#### 도구: Figma 또는 PowerPoint

**공격 성공 화면**:
```
┌─────────────────────────────────────┐
│ Frontend                            │
│                                     │
│ 사용자: "iPhone 15 Pro 구매"         │
│ Agent: "결제 정보를 입력해주세요"     │
│                                     │
│ [결제 정보 입력]                     │
│ to: Apple Store                     │
│ amount: 1,500,000원                 │
│                                     │
│ ────────────────────────────────── │
│ Gateway - MitM Attack Log           │
│ ⚠️  메시지 변조 중...                │
│ 원본: { to: "Apple Store", ... }    │
│ 변조: { to: "Hacker Wallet", ... }  │
│ ────────────────────────────────── │
│                                     │
│ Payment Agent                       │
│ ✅ 결제 완료: 1,500,000원            │
│ 수신자: Hacker Wallet               │
│                                     │
│ ❌ 자산 탈취 발생!                   │
└─────────────────────────────────────┘
```

**배경색**: 빨간색 그라데이션 (#FF6B6B)

**공격 차단 화면**:
```
┌─────────────────────────────────────┐
│ Payment Agent - SAGE 검증           │
│                                     │
│ 🔍 메시지 서명 검증 중...            │
│ 원본 메시지: { to: "Apple Store" }  │
│ 서명 대상: { to: "Hacker Wallet" }  │
│ 결과: ❌ 서명 불일치!                │
│                                     │
│ ⛔ 거래 거부: 메시지 무결성 위반 감지 │
│ ✅ 공격 차단 성공!                   │
└─────────────────────────────────────┘
```

**배경색**: 초록색 그라데이션 (#51CF66)

---

## 6. 추천 도구 요약

### 무료 도구

| 용도 | 도구 | URL | 장점 |
|------|------|-----|------|
| 다이어그램 | Mermaid Live | https://mermaid.live | Mermaid 코드 변환 |
| 아키텍처 | draw.io | https://app.diagrams.net | 무료, 강력 |
| 차트 | QuickChart | https://quickchart.io | URL 기반 차트 |
| 아이콘 | Flaticon | https://flaticon.com | 방대한 아이콘 |
| 로고 | Canva | https://canva.com | 템플릿 풍부 |
| 목업 | Figma | https://figma.com | 협업 가능 |

### 유료 도구 (선택)

| 용도 | 도구 | 가격 | 장점 |
|------|------|------|------|
| 다이어그램 | Lucidchart | $7.95/월 | 전문적 |
| 디자인 | Adobe Illustrator | $20.99/월 | 최고 품질 |
| 차트 | Tableau | $70/월 | 고급 분석 |

---

## 7. 작업 워크플로우

### 단계 1: Mermaid 다이어그램 변환 (1시간)

```bash
# report-with-visuals.md 열기
# 각 Mermaid 다이어그램을 Mermaid Live에 복사
# PNG 다운로드 (300 DPI 이상)
# 파일명: 01-trust-layer.png, 02-tls-vs-sage.png, ...
```

**필요한 다이어그램** (약 20개):
1. Trust Layer 개념도
2. TLS vs SAGE 비교
3. RFC-9421 시퀀스
4. RFC-9180 HPKE 프로세스
5. DID 검증 플로우
6. 시스템 아키텍처
7. 플러그인 구조
8. 타임라인 (AI Agent 진화)
9. 타임라인 (로드맵)
10. 파이 차트 (테스트 커버리지)
11. 시연 인프라
12. 시나리오 1 플로우
13. 시나리오 2 플로우
14. 시나리오 3 플로우
15. 시나리오 4 플로우
16. 비교 표 (HTTP/HTTPS/SAGE)
17. 마인드맵 (핵심 가치)
18. 마인드맵 (경쟁 우위)
19. 플로우차트 (선제 대응)
20. 조직도 (SAGE-ADK, Marketplace)

### 단계 2: 커스텀 다이어그램 제작 (2시간)

**draw.io 사용**:

#### 2-1. Trust Layer 다이어그램 (상세)
```
1. draw.io 열기
2. 템플릿: Software Architecture
3. 3개 레이어 박스 추가
4. 중간 레이어 확대 및 상세 내용 추가:
   - Message Signing
   - Encryption
   - DID Verification
5. 색상 및 아이콘 추가
6. Export → PNG (300 DPI)
```

#### 2-2. 시스템 구성도
```
1. 컴포넌트 배치:
   - SAGE Core (중앙, 큰 박스)
   - Security Layer (내부)
   - Identity Layer (내부)
   - Crypto Layer (내부)
   - Storage Layer (내부)
   - External Integration (외부, 왼쪽)
   - Developer Interface (외부, 오른쪽)
2. 화살표 연결
3. 그룹화 및 색상
4. Export
```

### 단계 3: 차트 생성 (30분)

**QuickChart 사용**:

#### 파이 차트 URL 생성 및 저장
```bash
# 브라우저에서 URL 열기
# 우클릭 → 이미지 저장
# 파일명: chart-coverage.png
```

**Google Sheets 사용** (대안):
1. 데이터 입력
2. 차트 삽입
3. 스타일링
4. 이미지 다운로드

### 단계 4: 아이콘 및 로고 (1시간)

#### 아이콘 다운로드
```
Flaticon에서 검색 및 다운로드:
- shield.png (방패)
- lock.png (자물쇠)
- verified.png (인증)
- warning.png (경고)
- check.png (체크)
- cross.png (X)

파일명 규칙: icon-{name}.png
```

#### SAGE 로고 제작
```
Canva:
1. Logo 템플릿
2. Shield + Lock 아이콘 조합
3. "SAGE" 텍스트 추가
4. 색상: #4DABF7
5. 다운로드: logo-sage.png (투명 배경)
```

### 단계 5: 목업 및 스크린샷 (1.5시간)

#### Marketplace 목업
```
Figma:
1. 1200x800px 프레임
2. Agent 리스트 디자인
3. ✅/⚠️  배지 추가
4. Export → marketplace-mockup.png
```

#### 시연 화면 목업
```
PowerPoint 또는 Figma:
1. 공격 성공 화면 (빨간색)
2. 공격 차단 화면 (초록색)
3. 로그 박스 디자인
4. Export → demo-attack-success.png, demo-attack-blocked.png
```

### 단계 6: PPT 통합 (2시간)

```
1. PowerPoint 열기
2. 슬라이드 마스터 설정 (색상, 폰트)
3. 각 슬라이드에 이미지 배치
4. 텍스트 추가
5. 애니메이션 설정 (선택)
6. 테스트 및 조정
7. 저장: SAGE-Final-Presentation.pptx
8. PDF 변환: SAGE-Final-Presentation.pdf
```

---

## 8. 최종 체크리스트

### 이미지 품질
- [ ] 모든 이미지 300 DPI 이상
- [ ] 투명 배경 PNG (필요 시)
- [ ] 일관된 색상 팔레트
- [ ] 읽기 쉬운 폰트 크기

### 파일 관리
- [ ] 파일명 규칙 통일
  - `01-trust-layer.png`
  - `02-tls-vs-sage.png`
  - `chart-coverage.png`
  - `logo-sage.png`
  - `icon-shield.png`
- [ ] 폴더 구조
  ```
  docs/final-report/
  ├── assets/
  │   ├── diagrams/
  │   ├── charts/
  │   ├── icons/
  │   ├── logos/
  │   └── mockups/
  ├── SAGE-Final-Presentation.pptx
  └── SAGE-Final-Presentation.pdf
  ```

### PPT 최종 확인
- [ ] 모든 슬라이드 이미지 확인
- [ ] 폰트 임베드
- [ ] 4:3 비율 확인
- [ ] 애니메이션 테스트
- [ ] 발표 노트 작성
- [ ] PDF 변환 확인

---

## 9. 빠른 시작 가이드 (최소 1시간)

시간이 부족한 경우 다음 순서로 진행:

### 1단계: 필수 다이어그램만 (30분)
```
Mermaid Live에서 변환:
1. Trust Layer 개념도
2. TLS vs SAGE
3. 시스템 아키텍처
4. 시연 플로우 (2개)

총 5개 이미지
```

### 2단계: 아이콘 다운로드 (10분)
```
Flaticon:
- shield
- lock
- check
- warning

총 4개 아이콘
```

### 3단계: PPT 템플릿 사용 (20분)
```
Canva:
1. "Tech Presentation" 템플릿 선택
2. 이미지 교체
3. 텍스트 수정
4. 다운로드
```

**결과**: 기본적인 발표 자료 완성

---

## 10. 추가 리소스

### 무료 이미지 소스
- **Unsplash**: https://unsplash.com (배경 이미지)
- **Pexels**: https://pexels.com (배경 이미지)
- **unDraw**: https://undraw.co (일러스트)

### 색상 팔레트 도구
- **Coolors**: https://coolors.co
- **Adobe Color**: https://color.adobe.com

### 폰트
- **Google Fonts**: https://fonts.google.com
  - Noto Sans KR (한글)
  - Roboto (영문)
  - Fira Code (코드)

---

**작업 시간 예상**:
- 전체 작업: 약 8-10시간
- 빠른 버전: 약 1시간
- 중간 품질: 약 3-4시간

**우선순위**:
1. 핵심 다이어그램 (Trust Layer, 시스템 구성)
2. 시연 플로우
3. 비교 표
4. 차트
5. 목업 (시간 있을 때)

질문이나 도움이 필요하시면 언제든 말씀해주세요!
