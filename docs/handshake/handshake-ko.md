# SAGE Handshak Packae

SAGE (Secure Agent Guarantee Engine) 프로젝트에서 Secure 세션 통신을 위한 사전 합의를 제공하는 Go 패키지입니다.

## 주요 기능

기존 [A2A 프로토콜](https://a2a-protocol.org/latest/topics/what-is-a2a/#a2a-request-lifecycle)의 확장 모듈로 grpc로 핸드쉐이크를 수행합니다.

<img src="../assets/SAGE-handshake.png" width="450" height="500"/>

- **DID 서명 검증 & 부트스트랩 암호화**: 메타데이터의 DID와 Ed25519 서명을 검증하고, Request/Response 페이로드는 상대 DID 공개키로 암호화해 중간자 공격을 차단합니다.
- **Ephemeral 키 합의와 세션 수립**: X25519 임시 교환으로 공유 비밀(shared secret)을 얻고 세션내에서 사용할 서명 키와 암호화 키를 파생합니다. 이 키로 세션 내 메세지를 암호화 및 서명하여 메세지를 보호합니다.
- **세션·논스 관리**: 생성·조회·만료가 정책에 따라 자동 관리되고, 요청마다 고유 식별 정보(예: 키 ID·논스)를 점검해 재사용(Replay)을 막습니다. 세션이 끝나면 관련 자료는 안전하게 폐기됩니다.
- **이벤트 기반 확장성**: `Events` 인터페이스를 통해 OnInvitation/OnRequest/OnComplete 등 단계별 훅과 키 식별자 발급, outbound 응답 자동화를 구현할 수 있습니다.

**핸드쉐이크 4단계**

요청 에이전트는 [A2A의 Agent Discovery](https://a2a-protocol.org/latest/topics/agent-discovery/) 로 DID를 알고 있으며, 두 에이전트의 DID는 모두 블록체인에 등록되어 있다고 가정합니다.
DID Document를 통해 상대의 공개키를 조회(Resolve)하며, 신원 서명 검증과 부트스트랩 암호화에 사용합니다.

1. Invitation(agent A -> agent B):
   - 요청 에이전트 A가 자신의 DID와 함께 세션 수립 의사를 전송합니다.
   - 상대 에이전트 B는 A의 DID를 resolve하여 공개키를 얻고, 서명 검증을 통해 유효한 요청임을 확인합니다.
2. Request(agent A -> agent B):
   - 요청 에이전트 A는 ephemeral 공개키(X25519)를 생성하여 상대 에이전트 B에게 보냅니다. 데이터는 B의 DID 공개키로 암호화되며, A의 신원키(Ed25519)로 서명됩니다.
   - 상대 에이전트 B는 서명을 검증하고 복호화한 뒤, 요청 에이전트 A의 ephemeral 공개키를 보관합니다
   - 암호화 되어 전송되므로 복호화 키를 가진 상대 에이전트 외에는 데이터를 확인할 수 없습니다.
3. Response(agent B -> agent A):
   - 상대 에이전트 B는 ephemeral 공개키(X25519)를 생성하여 요청 에이전트 A에게 보냅니다. 데이터는 A의 DID 공개키로 암호화되며, B의 신원키(Ed25519)로 서명됩니다.
   - 요청 에이전트 A는 서명을 검증하고 복호화한 뒤, 요청 에이전트 B의 ephemeral 공개키를 보관합니다
   - 암호화 되어 전송되므로 복호화 키를 가진 상대 에이전트 외에는 데이터를 확인할 수 없습니다.
4. Complete(agent A -> agent B)
   - 두 에이전트는 shared secret 을 갖게 되었으므로, 요청 에이전트 A는 complete를 전송합니다.
   - 두 에이전트는 shared secret을 이용해 만든 의사 난수를 seed로하여 세션 아이디를 계산하며, 요청 에이전트와 상대 에이전트는 동일한 세션 아이디를 갖는 세션을 생성합니다.
   - 세션은 무작위 문자열 키 식별자 kid에 바인딩되며, 상대 에이전트 B는 complete 응답으로 kid를 요청 에이전트에게 전송합니다. 요청 에이전트 A는 B로부터 수신한 kid를 세션에 바인딩합니다. 이는 이후 HTTP Message Signatures(RFC 9421)의 keyId 필드에 들어가 두 에이전트가 메세지 송수신 시 서명 검증 시 세션 조회에 사용됩니다.

## 설치

```bash
go get github.com/sage-x-project/sage/handshake
```

## 아키텍처

```bash
├── client.go           # 요청 에이전트
├── server.go           # 상대 에이전트
├── session             # 세션 및 논스 관리
│   ├── manager.go      # 세션 생성 및 삭제
│   ├── metadata.go     # 세션 상태 및 만료 관리
│   ├── nonce.go        # 논스 관리
│   ├── session.go      # 세션 키 관리
│   └── types.go        # 세션 인터페이스
├── types.go            # 핸드 쉐이크 인터페이스
└── utils.go
```

### 세션 관리 구성요소

- `handshake/session/manager.go`: 세션 생성·조회·만료 등 세션 관리 담당하며, 백그라운드 정리를 위해 주기적 클린업 루프를 운용합니다.
- `handshake/session/nonce.go`: 세션 별로 사용된 nonce를 TTL 기반으로 저장해 재전송 공격을 탐지합니다.
- `handshake/session/session.go`: HKDF로 파생한 키를 이용해 ChaCha20-Poly1305 및 HMAC-SHA256 연산을 수행하고, 키 자료를 안전하게 폐기하는 로직을 포함합니다.
- `handshake/session/metadata.go`: 세션 메타데이터 ID, 생성/만료 시각, 상태를 생성하며, 외부 감사나 관측 시스템과 연동 시 활용할 수 있습니다.

## 빌드 방법

**CLI 도구 빌드**

```bash
# 프로젝트 루트에서 실행
go build -o sage-crypto ./cmd/sage-crypto

# 또는 go install 사용
go install ./cmd/sage-crypto
```

## 사용 방법

**요청 에이전트**

```go
package main

import (
   "fmt"
   "github.com/sage-x-project/sage/handshake"
   "github.com/sage-x-project/sage/core/message"
   "github.com/sage-x-project/sage/crypto"
)

// 요청 에이전트 생성
agentA := handshake.NewClient(conn, clientKeypair)

// Invitation
inv := handshake.InvitationMessage{
   BaseMessage: message.BaseMessage{
      ContextID: ctxID,
   },
}
if _, err := agentA.Invitation(ctx, inv, string(myDID)); err != if err != nil {
   panic(err)
}

// Request
eph := mustX25519()
jwk := must(formats.NewJWKExporter().ExportPublic(eph, crypto.KeyFormatJWK))

reqMsg := handshake.RequestMessage{
   BaseMessage: message.BaseMessage{
      ContextID: ctxID,
   },
   EphemeralPubKey: json.RawMessage(jwk),
}
if _, err := agentA.Request(ctx, reqMsg, serverPub, string(myDID));

err != if err != nil {
   panic(err)
}

// Completea
comMsg := handshake.CompleteMessage{
   BaseMessage: message.BaseMessage{
      ContextID: ctxID,
   },
}
if _, err := agentA.Complete(ctx, comMsg, string(myDID)); if err != nil {
   panic(err)
}
```

## 보안 고려사항

1. **Ephemeral 키 관리**: `Events.AskEphemeral`은 32바이트 X25519 공개키(raw)와 JWK 버전을 반환하지만 개인키는 애플리케이션이 소유합니다. 이벤트 구현에서 개인키를 안전하게 보관하고, 재사용 없이 세션마다 새 키를 생성하세요.
2. **부트스트랩 암호화**: Request/Response 단계는 `keys.EncryptWithEd25519Peer`를 사용해 상대 DID 공개키로 암호화합니다. 상대 DID Document가 최신 상태인지, 회전된 신원 키가 반영되어 있는지 주기적으로 확인해야 합니다.
3. **세션 키 폐기**: `session.SecureSession.Close()`는 AEAD 키·HMAC 키·HKDF 시드를 모두 0으로 덮어씁니다. 세션 만료 시 반드시 `Manager.RemoveSession` 또는 `Close`를 호출해 키가 메모리에 남지 않도록 합니다.
4. **논스 재사용 방지**: `session.NonceCache`는 `kid`-`nonce` 조합을 TTL 기반으로 추적합니다. HTTP Message Signatures에 nonce를 채우고, 각 메시지 처리 시 `Seen` 결과를 검사하여 재전송 공격을 차단하세요.
5. **미완료 컨텍스트 정리**: 서버는 Request 수신 시 `pending` 맵에 상대 ephemeral 키를 보관합니다. Complete가 도착하지 않으면 `cleanupLoop`가 만료된 컨텍스트를 제거하므로, TTL 및 정리 주기를 서비스 정책에 맞춰 조정하고 모니터링하세요.

## 오류 처리

### 일반적인 오류

**DID가 없음**

```bash
missing did
```

- 메타데이터에 DID 필드가 없을 때 발생합니다. `signStruct` 호출 시 DID를 반드시 전달하고, 중간 프록시가 gRPC 메타데이터를 제거하지 않는지 확인하세요.

**서명 검증 실패**

```bash
signature verification failed
```

- DID Document의 공개키와 실제 서명키가 다르거나 메시지가 변조된 경우입니다. DID resolve 구성과 time 동기화를 점검하고, Invitation/request 모두 동일한 TaskID와 ContextID를 사용하세요.

**복호화 실패**

```bash
request decrypt: .../ response decrypt: ...
```

- 부트스트랩 복호화 실패입니다. 상대 DID 공개키가 최신인지, Base64URL 인코딩이 손상되지 않았는지, 동일한 키 포맷(Ed25519)을 사용했는지 검토하세요.

**세션 키 생성 실패**

```bash
ask ephemeral: ..
```

- 이벤트 레이어에서 새 ephemeral 키 생성에 실패한 경우입니다. 키 관리 서비스 접근 권한을 확인하고, 장애 시 재시도·대체 경로를 마련하세요.

**세션 만료**

```bash
session expired
```

- `SecureSession`이 `MaxAge`, `IdleTimeout`, `MaxMessages` 정책을 위반한 상태입니다. 트래픽 패턴에 맞춰 세션 구성을 조정하거나 새 핸드쉐이크를 강제하세요.

## 정책 및 가시성

- **세션 정책**: `session.Config`를 통해 MaxAge(절대 만료), IdleTimeout(유휴 만료), MaxMessages(허용 메시지 수)를 설정하여 장시간 연결·폭주를 제어합니다.
- **로그 및 감사**: `handshake.Events`의 OnInvitation/OnRequest/OnComplete 콜백에서 DID 검증 결과, ephemeral 키 메타데이터, 세션 파라미터를 로깅하면 추후 감사 추적이 가능합니다.
- **모니터링 지표**: 세션 생성/만료율, nonce 재사용 탐지 횟수, 서명 검증 실패율 등을 수집하면 보안 사고 감지와 성능 튜닝에 도움이 됩니다.

## 고급 기능

- **KeyID 자동 발급**: 이벤트 구현이 `KeyIDBinder`를 함께 구현하면 서버가 Complete 이후 `IssueKeyID`를 호출해 `kid`를 즉시 응답에 포함시킵니다. 이후 HTTP Message Signatures의 `keyId`와 연동해 검증을 단순화할 수 있습니다.
- **아웃바운드 응답 흐름**: `NewServer`에 outbound gRPC 클라이언트를 주입하면 Request 수신 직후 `sendRes.ponseToPeer`로 Response를 푸시할 수 있습니다. 상대가 NAT 뒤에 있거나 비동기 협상이 필요한 경우 유용합니다.
- **세션 파생 단순화**: `session.Manager.EnsureSessionWithParams`는 shared secret과 컨텍스트 정보만으로 양쪽에서 동일한 세션 ID와 키를 생성합니다. 동일 세션 중복 생성을 방지하고 레이스를 줄입니다.
- **재전송 창 제어**: `session.Config`의 `IdleTimeout`/`MaxMessages`를 업무 패턴에 맞춰 조정하고 `NonceCache` TTL을 메시지 수명보다 짧게 두면 재전송·폭주 공격을 세밀하게 제어할 수 있습니다.
- **메타데이터/감사 연동**: `Events.OnRequest`와 `OnComplete` 콜백에 DID 검증 결과나 세션 파라미터를 기록해 감사 로그, SIEM, 정책 엔진과 쉽게 연동할 수 있습니다.
