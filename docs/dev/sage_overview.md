# 프로젝트명

- 블록체인 DID 기반 AI 에이전트 신뢰 통신 프레임워크 ( **SAGE** | **S**ecure **A**gent **G**uarantee **E**ngine)

# 개발 목적

현존하는 AI 에이전트 간 통신(Agent-to-Agent, A2A) 프로토콜은 보안 측면의 표준이 부족하여, 중간자 공격(Man-in-the-Middle), 요청 위조, 데이터 변조 등의 위험에 노출되어 있습니다 [solo.io](https://www.solo.io/blog/deep-dive-mcp-and-a2a-attack-vectors-for-ai-agents#:~:text=The%20Model%20Context%20Protocol%20,a%20bumpy%20ride%20so%20far). 본 프로젝트의 목적은 **블록체인 기반 분산신원확인(DID)**과 **RFC 9421 표준에 따른 HTTP 부분 서명 기술**을 도입함으로써, 보호된 세션 통신을 통해 AI 에이전트 간 통신에서 발생할 수 있는 보안 취약점을 해결하는 것입니다. 이를 통해 각 에이전트의 신원을 **암호학적으로 검증**하고, 통신 메시지의 **무결성**과 **신뢰성**을 보장하고자 합니다.

# 프로젝트 소개

본 프로젝트는 오픈소스 형태로 개발되며, AI 에이전트들 사이의 안전한 통신을 위한 **A2A 통신 프로토콜 및 보안 모듈**을 구현합니다. 구체적으로 각 에이전트는 블록체인 기반 DID를 부여받아 **고유한 분산 ID와 공개키**를 가지며, 통신 전 핸드쉐이크를 통해 안전한 세션을 생성하고, 세션 내 통신은 암호화 되어 종단간 보안을 보장합니다. 상호 통신 시에는 **메시지에 디지털 서명**을 포함합니다. 서명 기술은 2024년 표준화된 **RFC 9421 (HTTP Message Signatures)**를 참고하여 구현되며, HTTP 메시지의 일부만으로도 서명이 가능하도록 지원합니다. 부분 서명을 활용하면, 중간 프록시나 게이트웨이를 거치는 동안에도 중요한 메시지 요소의 무결성을 유지할 수 있습니다 [datatracker.ietf.org](https://datatracker.ietf.org/doc/html/rfc9421#section-abstract-1%23:~:text=This%20document%20describes%20a%20mechanism,%C2%B6).
또한 A2A 프로토콜과 함께 **MCP (Model Context Protocol) 서버**를 연동하여, 에이전트가 툴이나 자원에 접근할 때도 동일한 DID 기반 인증과 서명을 사용하도록 확장합니다. 이런 구조를 통해 **요청자가 실제 신뢰할 수 있는 AI 에이전트인지를 검증**하고, **메시지가 전송 중 변조되지 않았음을 확인**할 수 있습니다. 모든 통신은 TLS 등 전송계층 암호화와 병행하여 이루어지며, **서명 검증 실패 시 요청 거부** 등의 정책을 적용하여 보안을 강화합니다.

# 기대효과

이 프로젝트의 결과물로, AI 에이전트 생태계에 다음과 같은 긍정적 효과가 기대됩니다:

- **에이전트 신원 보장**: 블록체인 DID를 통해 각 에이전트를 고유하게 식별하고 인증함으로써, 악의적인 에이전트의 **신분 위장 및 가장**을 방지합니다 [linkedin.com](https://www.linkedin.com/pulse/blockchain-secret-agentic-ai-success-benjamin-manning-tgmif#:~:text=Blockchain%20enhances%20AI%20security%20by,enabling). 이를 통해 오직 신뢰할 수 있는 에이전트들만이 통신에 참여하도록 제어할 수 있습니다.
- **메시지 무결성 향상**: RFC 9421 기반의 서명은 메시지가 생성된 이후 도착할 때까지 내용이 변경되지 않았음을 보장하여, 중간자 공격이나 데이터 변조 시도를 무력화합니다. 서명이 검증된 메시지만 처리함으로써 **오작동이나 오남용을 예방**할 수 있습니다.
- **종단간 보안**: 매 세션마다 새로운 임시 키를 생성하고 이를 기반으로 암호화 통신을 수행하여 메시지 기밀성과 무결성을 보장합니다. 세션 종료 시 키를 즉시 폐기해 키 재사용을 차단하고, **전방향 비밀성(PFS)**을 확보합니다.
- **보안 표준 선도**: 제안된 방식은 현재 A2A/MCP 표준에서 부족한 보안 요소를 보완하여 업계 표준으로 발전할 가능성이 있습니다 [solo.io](https://www.solo.io/blog/deep-dive-mcp-and-a2a-attack-vectors-for-ai-agents#:~:text=The%20Model%20Context%20Protocol%20,a%20bumpy%20ride%20so%20far). 오픈소스로 공개함으로써 많은 개발자들이 이를 활용하거나 기여하여 **멀티에이전트 시스템 보안 수준**을 전체적으로 끌어올릴 수 있습니다.
- **투명성과 신뢰성**: 모든 에이전트의 DID와 서명이 블록체인 및 검증 로그에 기록되므로, 에이전트들의 행동을 추적하고 감사할 수 있습니다 [house-of-communication.com](https://www.house-of-communication.com/de/en/brands/plan-net/landingpages/agentic-services/ai-agents-transparency-and-security.html#:~:text=claims%3F%20Serviceplan%20Group%20has%20developed,the%20reliability%20of%20the%20system). 이는 **에이전트 간 상호작용의 투명성**을 높이고, 문제 발생 시 **책임 소재를 규명**하는 데에도 도움을 줍니다 [house-of-communication.com](https://www.house-of-communication.com/de/en/brands/plan-net/landingpages/agentic-services/ai-agents-transparency-and-security.html#:~:text=claims%3F%20Serviceplan%20Group%20has%20developed,the%20reliability%20of%20the%20system). 나아가 기업 환경에서 멀티에이전트 시스템을 도입할 때 신뢰 구축에 기여할 것입니다.

# 요약

- **프로젝트명**: 블록체인 DID 기반 AI 에이전트 신뢰 통신 프레임워크

- **개발 목적 (200자 이내)**: AI 에이전트 간 통신에서 발생하는 중간자 공격, 위조, 변조 등 보안 위협을 막기 위해 블록체인 기반 DID와 RFC 9421 디지털 서명 기술을 도입합니다. 이를 통해 에이전트 신원을 검증하고 메시지 무결성을 보장하는 안전한 A2A 통신 프로토콜을 구축하는 것이 목표입니다.
- **프로젝트 소개 (300자 이내)**: 본 프로젝트는 오픈소스로 개발되며, AI 에이전트들이 상호 통신 시 각자의 블록체인 DID를 활용해 상호 인증하고, 주고받는 HTTP 요청/응답에 디지털 서명을 첨부하도록 합니다. DID가 제공하는 분산 신원확인 기술과 RFC 9421 표준에 따른 메시지 서명/검증 모듈을 결합하여, 에이전트 간 요청이 위변조되지 않고 신뢰할 수 있는지 실시간 검증하는 A2A 통신 프레임워크를 구현합니다.
- **기대효과 (500자 이내)**: 이 프레임워크를 도입하면 AI 에이전트 생태계의 **보안과 신뢰 수준이 크게 향상**됩니다. 첫째, DID 기반 인증으로 승인된 에이전트만이 통신에 참여하게 되어 **에이전트 가장 및 사칭을 방지**합니다. 둘째, 디지털 서명을 통해 **메시지 내용의 무결성**을 보장하여 데이터 변조나 중간자 개입 없이 안전한 정보 교환이 가능합니다. 이는 금융, 의료 등 민감한 데이터를 다루는 에이전트 활용 분야에서 **안전성 및 신뢰성** 확보에 기여합니다. 마지막으로 본 프로젝트를 오픈소스로 공개함으로써 관련 기술 표준화와 생태계 활성화를 촉진하고, 다양한 개발자들이 협업하여 **멀티에이전트 시스템 전반의 보안 수준 향상**에 이바지할 것으로 기대됩니다.

# 사용 기술 및 스택

- **Blockchain & DID**: 이더리움 등 퍼블릭 블록체인 또는 사이드체인 활용, W3C DID 표준, 분산 ID 관리

- **RFC 9421 (HTTP Message Signatures)**: HTTP 메시지 부분 서명/검증 구현 (예: 헤더 및 본문 일부 서명) [datatracker.ietf.org](https://datatracker.ietf.org/doc/html/rfc9421#section-abstract-1%23:~:text=This%20document%20describes%20a%20mechanism,%C2%B6)
- **AI Agent & A2A Protocol**: 오픈소스 A2A 프로토콜 스택, MCP 서버 연동, Agent Card 등 에이전트 정의 구조 활용
- **기타**: Cryptography (RSA/ECDSA/EdDSA 서명 알고리즘), Web Backend (Node.js 또는 Python, Golang 기반 서버), REST/HTTP, TLS 보안, GitHub 협업, CI/CD 등

⠀*(상기 기술 스택은 프로젝트 진행에 따라 유동적으로 조정될 수 있습니다.)*

# 참여 방법

- **GitHub 레포지토리**: 프로젝트 소스코드는 공개 GitHub 레포지토리에 관리됩니다. 이슈 등록, Pull Request를 통해 기여할 수 있습니다. (레포지토리 주소: https://github.com/SAGE-X-project/sage)

- **커뮤니케이션**: 프로젝트 디스코드 채널에 참여하여 실시간으로 의견을 나누고 질문을 할 수 있습니다. 중요한 공지나 회의 일정은 디스코드 및 GitHub wiki에 공유됩니다.
- **기여 방식**: 개발, 문서화, 테스트 등 다양한 형태로 참여 가능하며, 초기에는 핵심 모듈(DID 인증, 서명/검증 엔진 등) 개발을 위주로 진행됩니다. 기여자들은 주마다 진행 상황을 공유하고 코드를 리뷰하며 협업합니다.

# 활동 기간 및 개발 일정

- **아이데이션 & 설계 (1~2주차)**: 문제 정의, 요구사항 분석, 시스템 아키텍처 설계. DID 발급 방식, 서명 처리 흐름, 프로토콜 개략 설계를 확정.

- **개발 단계 1 (3~6주차)**: DID 관리 모듈 구현, 기본 A2A 통신 프로토콜 제작, 메시지 서명/검증 모듈 개발. MVP 형태로 에이전트 간 간단한 서명 교환 시연.
- **개발 단계 2 (7~10주차)**: MCP 서버 연동, 에이전트 등록/탐색 기능 개선(Agent Card 활용), 서명 정보의 블록체인 연계(서명 검증 로그를 블록체인에 기록 등).
- **테스트 및 문서화 (11~12주차)**: 시나리오별 통합 테스트 (중간자 공격 시나리오, 위조 메시지 검출 등) 진행. 결과 분석 및 성능 측정, 기술 문서와 사용자 가이드 작성.
- **최종 발표 및 제출**: 오픈소스 경진대회 일정에 맞춰 프로젝트 산출물 제출, 발표 준비 및 데모 시연.

⠀*(상기 일정은 예상이며, 프로젝트 진행에 따라 유동적으로 조정될 수 있습니다.)*

# 현실 가능성 분석

## 블록체인 DID를 통한 에이전트 식별 및 인증

**현실성 장점**: 블록체인 기반 DID 기술은 이미 W3C 표준으로 채택되어 있어, 분산 환경에서의 **신원 증명 수단으로 현실성**이 높습니다. 각 에이전트에 DID를 부여하면 중앙 권위 없이도 **고유한 식별자와 공개키**를 가질 수 있고, 상대 에이전트는 블록체인 또는 분산 ID 레지스트리 조회를 통해 **상대의 공개키를 신뢰성 있게 획득**할 수 있습니다 [house-of-communication.com](https://www.house-of-communication.com/de/en/brands/plan-net/landingpages/agentic-services/ai-agents-transparency-and-security.html#:~:text=claims%3F%20Serviceplan%20Group%20has%20developed,the%20reliability%20of%20the%20system). 실제로도 Masumi 프로토콜 등의 사례에서 모든 AI 에이전트에 DID를 부여하고 블록체인에 활동을 기록하여 **에이전트의 신원과 행위를 추적**하고 **책임을 부여**하는 시도가 있습니다 [house-of-communication.com](https://www.house-of-communication.com/de/en/brands/plan-net/landingpages/agentic-services/ai-agents-transparency-and-security.html#:~:text=claims%3F%20Serviceplan%20Group%20has%20developed,the%20reliability%20of%20the%20system). 또한 블록체인 자체의 탈중앙 특성 덕분에 **해킹이나 조작에 강인한 인프라**를 제공하여, 에이전트 신원 정보를 위변조하거나 삭제하기 어렵게 만듭니다 [house-of-communication.com](https://www.house-of-communication.com/de/en/brands/plan-net/landingpages/agentic-services/ai-agents-transparency-and-security.html#:~:text=But%20the%20technology%20can%20do,sources%20to%20perform%20their%20tasks). 이러한 특성은 다수의 에이전트가 참여하는 시스템에서 **신뢰 기반**을 형성하는 데 유용합니다.
**현실성 한계**: 그러나 DID를 실제 적용할 때 고려해야 할 요소들도 있습니다. 우선 **DID 발급과 관리의 복잡성**입니다. 모든 에이전트가 DID를 발급받고 블록체인에 등록하려면, 해당 프로세스의 비용(예: 퍼블릭 체인 거래 수수료)이나 속도 지연을 무시할 수 없습니다. 대규모 에이전트 시스템에서는 **DID 등록/조회로 인한 성능 이슈**도 검토해야 합니다. 또한 DID를 발급받았다고 해서 **에이전트의 신뢰도가 담보되는 것은 아닙니다**. 악성 에이전트도 DID를 발급받을 수 있으므로, 결국 DID는 “누구인지”를 증명할 뿐 “믿을 만한 존재인지”는 별도 문제입니다. 이를 보완하려면 초기 합의된 **신뢰할 수 있는 DID 발급자** 또는 화이트리스트, 혹은 행위 기반의 **신뢰 점수**와 같은 추가적인 메커니즘이 필요할 수 있습니다. 마지막으로, 상호 운용성 문제로서 여러 DID 메소드(예: did:ethr, did:key, did:ion 등)가 존재하는데, 어떤 방식을 채택할지, 또 **상대 에이전트가 동일한 DID 해석 체계를 지원하는지** 등의 이슈가 있습니다. 이러한 한계에도 불구하고, 현재 많은 프로젝트와 기업들이 DID를 **디지털 신원 증명의 핵심 솔루션**으로 연구하고 있어 적용 가능성은 날로 높아지고 있습니다 [linkedin.com](https://www.linkedin.com/pulse/blockchain-secret-agentic-ai-success-benjamin-manning-tgmif#:~:text=Blockchain%20enhances%20AI%20security%20by,enabling).

## RFC 9421을 활용한 메시지 무결성 유지

**현실성 장점**: RFC 9421 (HTTP Message Signatures)은 2024년에 표준화된 최신 규격으로, HTTP 메시지의 일부 혹은 전체에 디지털 서명을 부여하는 방법을 정의합니다 [datatracker.ietf.org](https://datatracker.ietf.org/doc/html/rfc9421#section-abstract-1%23:~:text=This%20document%20describes%20a%20mechanism,%C2%B6). 이 표준을 활용하면, 에이전트 간 주고받는 API 요청/응답의 **헤더, 본문, 메서드, 경로 등 필요한 요소만 선별적으로 서명**할 수 있어 효율적입니다 [datatracker.ietf.org](https://datatracker.ietf.org/doc/html/rfc9421#section-abstract-1%23:~:text=This%20document%20describes%20a%20mechanism,%C2%B6). 예를 들어 본문은 그대로 두고 헤더와 중요한 필드만 서명함으로써, **프록시 서버가 본문 압축을 변경하거나 헤더를 추가해도 서명 검증에 지장**이 없도록 할 수 있습니다. 이러한 부분 서명 기법은 **중간 경로 변형에 견디면서도 핵심 데이터의 무결성을 보존**하기에 매우 현실적입니다. 이미 Amazon, Digital Bazaar 등에서 해당 표준의 초안을 공동 작성했고, 일부 개발자 커뮤니티에서 RFC 9421 서명 구현을 진행하고 있어 실용 라이브러리나 툴링도 점차 등장하는 추세입니다. 또한 기존에 널리 쓰이는 JWT, OAuth 등의 토큰 기반 서명보다 **HTTP 메시지 레벨에서 세분화된 검증**을 할 수 있다는 점에서, 에이전트 통신 보안에 특화된 솔루션이 될 수 있습니다. 무엇보다 서명과 검증 연산은 RSA, ECDSA 등 검증된 암호화 알고리즘을 사용하므로 **기술적 신뢰성은 높다** 할 수 있습니다.
**현실성 한계**: 반면, 실제 적용 시 고려해야 할 부분은 **성능 및 구현 복잡도**입니다. 모든 요청/응답마다 서명/검증을 수행하면 암호화 연산 부담이 추가되며, 에이전트 간 통신이 빈번한 시나리오에서는 **지연(latency)**이 늘어날 수 있습니다. 특히 에이전트 수가 많고 통신이 폭주하면, 서명 키 관리나 검증 작업의 **스케일링 이슈**가 발생할 수 있습니다. 또한 RFC 9421은 비교적 새로운 표준이므로, **상용 환경에 검증된 구현체가 아직 적고**, 여러 언어/플랫폼에서 호환되게 만드는 데 노력이 필요합니다. 부분 서명 활용 시 **어떤 요소를 서명에 포함할지 정책 결정**도 중요합니다. 예를 들어 timestamp나 nonce 등을 서명에 포함하여 재전송 공격을 막는 등의 추가 설계를 해야 완벽한 보안을 보장할 수 있습니다. 마지막으로, 서명 그 자체는 무결성과 신원 확인을 제공하지만, **에이전트가 받은 데이터를 어떻게 활용하는지에 대한 보장은 별개**입니다. 예컨대, 서명이 유효하더라도 메시지 내용이 악의적일 경우(프롬프트 주입 등) 에이전트 행동을 해칠 수 있으므로, **콘텍스트 검증이나 권한 제어** 등의 보완이 필요합니다. 요약하면 RFC 9421 기반 서명은 A2A 통신의 **기술적 무결성 보장 수단으로 충분히 현실성 있고 효과적**이지만, 전반적 보안을 위해서는 시스템 아키텍처 전반의 최적화와 추가 대책이 함께 고려되어야 합니다.

## 제안 기술의 적용 가능성 및 한계 종합

제안한 DID 기반 인증과 메시지 서명 기법은 **AI 에이전트 보안 문제에 직접 적용 가능한 솔루션**으로 높은 잠재력을 지닙니다. 특히 에이전트 간 통신에서 가장 우려되는 **신원 위장**과 **데이터 변조**를 동시에 해결한다는 점에서, 현재 떠오르는 멀티에이전트 보안 이슈의 핵심을 찌르고 있습니다 [linkedin.com](https://www.linkedin.com/pulse/blockchain-secret-agentic-ai-success-benjamin-manning-tgmif#:~:text=Blockchain%20enhances%20AI%20security%20by,enabling)[house-of-communication.com](https://www.house-of-communication.com/de/en/brands/plan-net/landingpages/agentic-services/ai-agents-transparency-and-security.html#:~:text=But%20the%20technology%20can%20do,sources%20to%20perform%20their%20tasks). DID를 통해 에이전트의 **신원을 사전에 등록**해 두고 상호 검증하는 방식은, 사람 사회의 PKI 인증서 교환과 유사하게 **기계 간 신뢰 기반**을 형성해줄 것입니다. 또한 서명된 메시지만 처리하는 프로토콜 규칙을 둔다면, 공격자가 가로채 내용을 바꾸거나 가짜 요청을 보내도 검증에 실패하여 **실제 피해로 이어지지 않게 할 수 있습니다**.
다만, 이 접근법이 **만능 보안 솔루션은 아니라는 한계**도 존재합니다. 예를 들어 DID로 신원을 확인하더라도 **내부적으로 탈취된 에이전트 계정**이나 **프라이빗 키 유출**에는 취약할 수 있습니다. 결국 사람의 인증과 마찬가지로, **키 관리 문제**는 남습니다. 또한 서명이 유효하다고 해서 그 메시지의 **맥락까지 안전한 것은 아니므로**, 에이전트 스스로 악성 명령을 구분하거나 거부할 수 있는 **지능적인 보안 대책(예: 프롬프트 필터링)**도 병행되어야 합니다.
그럼에도 불구하고, 현재 AI 에이전트 보안 분야에서 DID와 메시지 서명은 **가장 구현 가능하고 직접적인 대응책**으로 평가됩니다. 블록체인과 암호화 기술의 성숙도, 그리고 AI 에이전트 기술의 발전 속도를 볼 때, 두 영역의 융합은 충분히 실현 가능하며 곧 사례가 등장할 것으로 보입니다. 특히 **오픈소스 생태계**에서 이러한 시도를 시작하면, 업계 표준이나 모범 사례로 확산될 수 있다는 점에서 실제 적용 가능성은 더욱 높아집니다 [solo.io](https://www.solo.io/blog/deep-dive-mcp-and-a2a-attack-vectors-for-ai-agents#:~:text=The%20Model%20Context%20Protocol%20,a%20bumpy%20ride%20so%20far). 결론적으로, 제안 기술들은 **현 시점에서 실효성 있는 보안 강화책이며, 다만 구현상의 복잡성과 운영상의 보완책을 함께 고려**해야 할 것입니다.
