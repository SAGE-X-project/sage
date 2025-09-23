# SAGE Handshak Packae

기존 A2A 프로토콜 중간에 핸드쉐이크(handshake) 레이어를 추가하여 종단간 보안을 제공하는 Go 패키지 입니다.

## 주요 기능

핸드쉐이크는 다음 4단계로 수행됩니다.
![E2EE request lifecycle Diagram](../assets/SAGE-handshake.png)

1. Invitation
2. Request
3. Response
4. Complete
