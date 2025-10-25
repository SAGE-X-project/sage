
# SAGE(Secure Agent Guarantee Engine)
## 팀소개
- 추후에 추가
## 프로젝트 개요(소개)
  - SAGE Overview
    - ai를 사용하는 수많은 유저들이 Agent를 사용할때, Agent 간의 메시지를 블록체인의 DID기술, RFC-9421, RFC-9180을 지원하여 보안 -> 개인정보, 자산을보호
## 개발 밴경 및 목적
### 개발 배경
  - Ai 기술의 성장 -> 더 많은 Agent가 생성됨. (대 Agent의 시대, Appless 시대가 다가옴 -> smart phone 시대의 app이 ai 시대의 agent, mcp)
    - 누구나 Agent를 만들수 있도록 Agent Builder를 지원하는 서비스 증가. ( n8n, make, OpenAi, Gemini 등 )
    - 기업 내부에 자동화 시스템을 위해 Agent를 개발하고 도입하는 사례도 증가.
    - Agent를 통해 Stable Coin으로 결제를 지원하는 AP2 기술 등장.
  - 현재는 Ai Agent에서 보안에 대한 고민이 깊지 않음. (다음의 논문이 존재함. A Survey of LLM-Driven AI Agent Communication: Protocols,
Security Risks, and Defense Countermeasures, The Dark Side of LLMs:
Agent-based Attacks for Complete Computer Takeover )
  - Agent의 현재 보안은 TLS,HTTPS를 기본으로 OAuth2 등을 사용. 
  - TLS(HTTPS)는 종단간의 보안이 아닌, 연결 구간마다의 서로다른 여러 TLS 구간으로 셋업되는 한계점. 따라서 종단간(end-to-end)의 메시지 무결성을 보장하지 못함(미들웨어 어택이 가능한 지점이 발생). rfc-9421이 등장한 배경이기도함.
  - Agent의 할루시네이션 문제를 해결하기 위해 MCP를 연결해서 사용. (이를 위해 유저가 Agent에게 권한 부여)
  - sub-agent와 multi-agent로 확장을 위해 A2A 기술 등장(A2A 기술도 TLS,Https를 사용. OAuth2가 옵션.). 하지만 메시지의 종단간 무결성을 보장하지는 못하며, AgentCard만으로 Agent가 악의적인 목적인지 알기 어려움.
  - AP2 프로젝트는 Agent 등록 및 결제까지만의 한계가 있음 (결제 목적이라 개인정보 유출까지 막기 어려움)
  - Agent를 통해 많은 정보가 유출될 수 있으며, 개인정보 탈취, 자산 해킹 등의 더 큰 문제가 발생할 수 있는 기술이 Ai 시장임
  - 보안 사고가 발생한 후 뒷수습으로 인해 발생하는 사회적 비용이 매우 큼 (보안 사고 뒷수습으로 인해 발생한 사회적 혼란과 비용에 관한 자료를 예시로 들면 ? -> skt 해킹 사태등)

### 개발 목적
  - Agent 개발에서 보안을 위한 Trust Layer가 필요함. (지금 Agent가 Http 라면, Https가 필요.)
  - Agent가 통신하는 메시지는 종단간의 무결성을 가져야함. -> RFC-9421 적용
  - Agent가 사용하는 key는 투명한 환경에서 공개되어야함.(누군가 조작하지 못하도록 블록체인 기술을 이용) -> AgentCard와 PubKey를 블록체인에 등록. 누구에게나 투명하게 공개되어있는 블록체인 정보를 이용하여 Agent 정보 이용.
  - Agent가 통신하는 메시지가 노출되지 않고 숨겨져야 할 때가 있으므로 Agent 통신에 사용되는 메시지를 암호화할 수 있어야 한다. -> 핸드쉐이크 기반 메시지 암호화 통신 적용.
  - Ai 기술이 성장하는 지금 시점 부터, Agent를 개발함에 있어 보안 기술을 적용하고, 허위 Agent (피싱용 등)로 인하여 사람들이 피해를 보지 않도록, 블록체인 기술과 보안 기술을 적용한 오픈소스를 구현하여, 대 Agent 시대에 개인정보 유출, 해킹 등의 피해 사례가 발생하지 않도록 기여
## 프로젝트 구성 및 기능
  - SAGE :
    - SAGE 오픈소스의 기능에 대한 설명 필요.
    - SAGE의 구성 아키텍쳐 필요.
    - Agent 구현에 있어서 Layer를 구분했을때, Trust Layer 기술이라는 설명 필요. (Layer 아키텍쳐로 설명)
    - 확장성 :
        - 플러그인 기반 설계 (암호학, 멀티 블록체인)
        - 라이브러리 빌드 지원
        - java, python, rust, typescript sdk 지원
## 추후 활용 방안 및 계획
  - rs-sage-core : core만 라이브러리로 빌드할 수 있도록 지원
  - SAGE-ADK (Agent Development Kit) : 보안이 강화된 Agent 개발 지원
    - SAGE 기술과 구글의 A2A 를 적용
    - Agent를 쉽게 Building할 수 있도록 지원하는 Builder
    - 현재 개발 진행중
  - Agent-Dashboard (Market) : 신뢰할 수 있는 Agent 정보 제공
    - BlockChain 에 등록된 Agent들의 정보를 실시간 24/7 볼 수 있는 Dashboard(Market) 지원
  - MCP Integration
    - MCP 기술에도 SAGE 적용
    - MCP에는 이제 Agent도 연동할 수 있도록 진화중
  - Google(gemini), Openai(chatgpt), Claude 에서 안전한 Agent를 연결하고 사용할 수 있는 환경 구성을 위해, easy-integration opensource 지원
  
## 시연
  - 시연 준비 사항
    - Contract (조금 수정해야할 수 있는데 어쨌든, 이더리움 테스트넷(Sepolia))에 배포
    - RPC -> 알케미 rpc api 사용
    - gpt site frontend (chat gpt 같은 사이트) -> 배포 (vercel) , 도메인 구매
    - gateway server (MIM Attack 감염) -> 배포 (supabase or aws)
    - Agent without sage -> 배포 (supabase or aws)
    - Agent with sage -> 배포 (supabase or aws)
    - AP2 (stable token 결제) -> 배포 (supabase or aws)
  - 인프라 구성
    - frontend — Agent without(with) sage — gateway server(감염) — AP2(결제용 agent)
  - 시연 시나리오
    - frontend -> Agent 연결 (보안이 적용되지 않음) -> 제품을 구매 -> 해킹으로 인한 자산 탈취가 가능함을 설명
    - frontend -> Sage 적용 Agent 연결 -> 제품을 구매 -> 해킹 시도 불가능 (RFC-9180케이스)
    - frontend -> Sage 적용 Agent 연결 -> 제품을 구매 -> 해킹 시도를 결제용 Agent에서 인식 (RFC-9421) 하여 처리 거절
## Appendix
