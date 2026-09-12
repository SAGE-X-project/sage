import json,re
from pathlib import Path
O=Path('analysis/2026-09-12');G=O/'graphs';ROOT=Path.cwd()
# Analyst-adjudicated requirements, distinct from automated claim candidates.
rows=[
('R01','본문·대상 무결성','README.md','RFC 9421 HTTP Message','pkg/agent/core/rfc9421/verifier_http.go','func (v *HTTPVerifier) VerifyRequest','부분 구현','strict 모드의 본문·대상 coverage는 구현. 기본 옵션은 느슨하며 호출자 설정이 보안 경계다.'),
('R02','서명 재전송 방지','docs/handshake/hpke-based-handshake-en.md','requests reject duplicate','pkg/agent/core/rfc9421/verifier_http.go','func checkSignatureTimes','결함 재현','created/expires 누락을 strict에서도 허용하고 nonce TTL과 전체 수락 기간을 연결하지 않는다. S04.'),
('R03','MCP 예제 replay 보호','examples/mcp-integration/basic-demo/README.md','signature','examples/mcp-integration/basic-demo/main.go','verifier := rfc9421.NewHTTPVerifier()','결함 재현','요청마다 새 guard 생성. basic-tool도 동일. 저장소 수명과 Close 누락. S03.'),
('R04','개별 요청에 연결된 응답 인증','docs/refactoring/BACKLOG.md','B-09','pkg/agent/core/rfc9421/verifier_http.go','func StrictHTTPResponseVerificationOptions','부분 구현','SignResponse/ResponseSigner는 존재하나 strict 검증의 필수 집합에 요청 nonce·signature가 없다. S05.'),
('R05','방향별 AEAD와 replay window','README.md','Session Management','pkg/agent/session/session.go','func (s *SecureSession) open','구현·테스트 확인','seq를 AAD에 결합하고 인증 후 sliding window를 갱신한다. 단순 복호화 반복 취약점은 과거 상태다.'),
('R06','세션 자동 키 교체','README.md','Automatic key rotation','pkg/agent/session/session.go','func (s *SecureSession) aeadForSeq','부분 구현','메시지 세대별 HKDF 키 교체는 존재. 보관된 동일 seed에서 재파생하므로 PCS나 seed 유출 후 과거키 보호는 아니다.'),
('R07','HPKE 기반 1 RTT','README.md','Handshake Protocol','pkg/agent/hpke/server.go','func (s *Server) HandleMessage','구현·테스트 확인','DID 서명, exporter+ephemeral DH, ackTag가 실제 경로에 있다. 표준 HPKE primitive와 SAGE 전용 합성 프로토콜을 구별해야 한다.'),
('R08','전방향 비밀성','docs/handshake/hpke-based-handshake-en.md','resulting `combined` seed','pkg/agent/hpke/server.go','combined, err := combineSecrets','설계 구현·증명 미확인','추가 ephemeral DH가 정적 KEM 키 유출 위험을 줄임. 합성 AKE의 형식 증명·독립 감사는 확인하지 못했다.'),
('R09','서버 수신 DID 바인딩','docs/refactoring/BACKLOG.md','B-10','pkg/agent/hpke/server.go','pl.RespDID != s.DID','구현·테스트 확인','RespDID 검사가 추가되어 과거 감사 결함 해소.'),
('R10','DoS cookie 선행 검증','README.md','verify cookie early','pkg/agent/hpke/server.go','senderDID, _, err := s.verifySender','부분 구현','현재 DID 조회·서명 검증 뒤 cookie 검사. HPKE 비용만 줄이고 앞선 비용은 막지 못한다.'),
('R11','DID와 등록 소유자 결합','docs/did/did-en.md','DID','contracts/ethereum/contracts/AgentCardVerifyHook.sol','function _validateDIDFormat','결함 재현','다른 주소를 포함한 DID를 등록할 수 있음. 이름을 소유자 인증으로 해석하는 통합에서는 신뢰 경계 손상. S01.'),
('R12','등록 키 소유 증명','README.md','Multi-Key Support','contracts/ethereum/contracts/AgentCardRegistry.sol','function _verifyKeyOwnership','결함 재현','Ed25519는 길이만 검사. ECDSA는 등록 계정 서명을 검사하나 등록 keyData와 결합하지 않는다. S01.'),
('R13','폐기·비활성 신원 거부','docs/refactoring/BACKLOG.md','B-06','contracts/ethereum/contracts/AgentCardRegistry.sol','function activateAgent','결함 재현','Go의 verified/active 검사는 개선됐지만 비활성 agent를 비소유자가 재활성화 가능. S02.'),
('R14','온체인 verified 키만 선택','docs/refactoring/BACKLOG.md','B-06','pkg/agent/did/ethereum/key_policy.go','if !k.Verified','구현·테스트 확인','Go 필터 자체는 구현. verified를 만드는 계약의 증명 강도가 약하다는 별도 문제 존재.'),
('R15','AgentCard proof의 DID 신뢰점','docs/refactoring/BACKLOG.md','B-07','pkg/agent/did/a2a_proof.go','func VerifyA2ACardProofWithDID','부분 구현','체인 키로 proof 검증하는 API 존재. 자기서명 검증과 신뢰점 검증의 선택은 호출자가 책임진다.'),
('R16','3단계 CLI 등록 --key','README.md','Phase 1: Commit with stake','cmd/sage-did/commit.go','key file loading not implemented yet','미완료','문서의 --key 예제가 즉시 실패한다. --private-key 우회도 빈 Keys와 TBD DID를 생성하여 다음 단계와 맞지 않는다. S06.'),
('R17','스마트컨트랙트 등록 상태기계','contracts/ethereum/README.md','commit','contracts/ethereum/contracts/AgentCardRegistry.sol','function registerAgentWithParams','부분 구현','commit-reveal/지연·stake는 구현. 영구 비활성화·소유 증명 결함과 별개로 평가해야 한다.'),
('R18','RFC 8785 canonical JSON','docs/refactoring/BACKLOG.md','B-11','pkg/agent/crypto/jcs/jcs.go','func','구현·테스트 확인','JCS 모듈 및 A2A/HPKE 연결 존재. 언어 간 모든 수치/Unicode 벡터 통과를 의미하지 않는다.'),
('R19','RFC 9421 RSA-PSS 상호운용','docs/dev/security-design.md','rsa-pss-sha256','pkg/agent/core/rfc9421/verifier_http.go','rsa.VerifyPKCS1v15','규격 불일치','알고리즘 표기는 rsa-pss-sha256이나 실제 HTTP 경로는 PKCS1v15. 표준 상대와 상호운용하지 못한다. S07.'),
('R20','HTTP/WS 입력 제한','docs/refactoring/BACKLOG.md','B-12','pkg/agent/transport/http/server.go','MaxBytesReader','구현·테스트 확인','HTTP/WS 제한과 origin 검사가 존재. ResponseSigner는 별도 무제한 버퍼링·streaming 미지원 과제다.'),
('R21','HMAC session RFC 9421 바인딩','docs/handshake/hpke-based-handshake-en.md','`keyId=kid`','pkg/agent/core/rfc9421/verifier_http.go','func (v *HTTPVerifier) verifySignature','통합 미완료','HTTPVerifier는 공개키 검증 경로. session.SignCovered와 RFC 9421 middleware 및 kid 해석이 자동 연결된 것으로 볼 수 없다.'),
('R22','x-channel-binding 프로파일','docs/handshake/hpke-based-handshake-en.md','x-channel-binding','pkg/agent/hpke/common.go','func','문서 예시·통합 미확인','문서에는 설계·예시가 있으나 pkg/internal/examples의 해당 헤더 구현 검색 결과 없음.'),
('R23','도구별 권한·위임 정책','docs/dev/security-design.md','Policy Engine','pkg/oidc/auth0/auth0.go','func','미구현 명시','OIDC 유틸리티는 있지만 tool/argument/tenant별 정책 엔진과 강제 경로는 확인되지 않음.'),
('R24','불변 감사 로그','docs/dev/security-design.md','Audit Logger','pkg/telemetry/logger/logger.go','func','미구현 명시','구조화 로그는 존재. hash-chain/서명 영수증/외부 체크포인트와 동일하지 않다.'),
('R25','멀티 언어 SDK 상호운용','README.md','experimental and not yet','sdk/typescript/src/client.ts','getDID()','미완료 명시','Go handshake·RFC 9421과 호환되지 않음을 README가 인정. 동일 언어 왕복만으로 완료 판정 불가.'),
('R26','기술 명세와 test vectors','docs/refactoring/BACKLOG.md','F-','internal/vectors/vectors.go','func','부분 구현','현재 sage-vectors 생성기가 있음. 외부 독립 구현의 벡터 검증 및 spec와 양방향 conformance는 이 폴더만으로 확증 불가.'),
('R27','Solana production 지원','README.md','Solana** — In development','pkg/agent/did/solana/client.go','func (c *SolanaClient) Register','부분 구현','Go RPC 클라이언트는 존재. 이 폴더에는 대응 배포 program과 실제 chain 상호운용 증거 부족.'),
('R28','외부 의존성 없음','README.md','Zero External Dependencies','go.mod','require (','문서 불일치','go.mod에는 circl/go-ethereum 등 외부 의존성이 있다. core 범위를 재정의하거나 주장 수정 필요.'),
('R29','production 99% 성공률','docs/PERFORMANCE_BENCHMARKS.md','99%+ success rate','pkg/agent/did/ethereum/agentcard_benchmark_test.go','func Benchmark','실증 근거 부족','날짜/하드웨어·시행 횟수·실패 분모가 명확하지 않음. 이번 측정은 local primitive benchmark뿐이다.'),
('R30','문서·구현 경로 최신화','docs/ARCHITECTURE.md','Architecture','pkg/agent/core/core.go','package core','미완료','기존 감사, 4-phase 설명, 옛 폴더 구조와 현 코드가 혼재. 과거 결함을 현재 결함으로 재사용하지 않도록 버전 귀속 필요.')]
bs=json.loads((G/'document-blocks.json').read_text());syms=json.loads((G/'go-semantic/graph-filtered.json').read_text())['symbols'];nodes=[];edges=[];out=[]
def ref(path,needle):
 ls=Path(path).read_text().splitlines();hit=next((i+1 for i,s in enumerate(ls) if needle.lower() in s.lower()),None)
 if hit is None:raise ValueError((path,needle))
 return {'file':path,'line':hit,'excerpt':ls[hit-1],'url':str(ROOT/path)+':'+str(hit)}
for id,title,dp,dpat,cp,cpat,status,note in rows:
 d=ref(dp,dpat);c=ref(cp,cpat);row={'id':id,'title':title,'status':status,'assessment':note,'document':d,'code':c,'method':'analyst-reviewed current source; testing stated individually'};out.append(row)
 nodes.append({'id':id,'kind':'requirement','label':title,'status':status});blocks=[b for b in bs if b['file']==dp and b['line']<=d['line']<=b['end_line']]
 if blocks:edges.append({'source':blocks[0]['id'],'target':id,'kind':'claims'})
 target='file:'+cp;found=[s for s in syms if s['file']==cp and s['line']<=c['line']<s['line']+s['lines']]
 if found:target=found[0]['id']
 nodes.append({'id':target,'kind':'code_evidence','label':cp+':'+str(c['line'])});edges.append({'source':id,'target':target,'kind':'reviewed_evidence','status':status})
(G/'traceability.json').write_text(json.dumps({'requirements':out,'nodes':nodes,'edges':edges},ensure_ascii=False,indent=2))
md=['# 문서–구현 추적표','', '자동 후보와 구별한 핵심 요구사항 30건의 현재 상태. 각 행의 문서·코드는 실제 파일과 줄을 가리킨다.','', '| ID | 요구사항 | 판정 | 문서 | 구현 증거 | 분석 |','|---|---|---|---|---|---|']
for r in out:
 d=r['document'];c=r['code'];md.append(f"| {r['id']} | {r['title']} | {r['status']} | [{d['file']}:{d['line']}]({d['url']}) | [{c['file']}:{c['line']}]({c['url']}) | {r['assessment']} |")
(O/'TRACEABILITY.ko.md').write_text('\n'.join(md)+'\n');print('requirements',len(out))
