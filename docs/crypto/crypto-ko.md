# SAGE Crypto Package

SAGE (Secure Agent Guarantee Engine) 프로젝트의 암호화 기능을 제공하는 Go 패키지입니다.

## 주요 기능

- **키 쌍 생성**: Ed25519, Secp256k1 알고리즘 지원
- **키 내보내기/가져오기**: JWK (JSON Web Key), PEM 형식 지원
- **안전한 키 저장소**: 메모리 및 파일 기반 저장소
- **키 회전**: 자동 키 회전 및 이력 관리
- **메시지 서명 및 검증**: 디지털 서명 생성 및 검증
- **블록체인 통합**: Ethereum, Solana 주소 생성 및 검증

## 설치

```bash
go get github.com/sage-x-project/sage/crypto
```

## 아키텍처

### 패키지 구조

```
crypto/
├── types.go              # 핵심 인터페이스 정의
├── keys/                 # 키 생성 및 관리
│   ├── ed25519.go       # Ed25519 구현
│   └── secp256k1.go     # Secp256k1 구현
├── formats/              # 키 형식 변환
│   ├── jwk.go           # JWK 형식
│   └── pem.go           # PEM 형식
├── storage/              # 키 저장소
│   ├── memory.go        # 메모리 저장소
│   └── file.go          # 파일 저장소
├── rotation/             # 키 회전
│   └── rotator.go       # 키 회전 관리
└── chain/               # 블록체인 통합
    ├── types.go         # Chain Provider 인터페이스
    ├── registry.go      # Provider 레지스트리
    ├── ethereum/        # Ethereum 지원
    └── solana/          # Solana 지원
```

## 빌드 방법

### CLI 도구 빌드

```bash
# 프로젝트 루트에서 실행
go build -o sage-crypto ./cmd/sage-crypto

# 또는 go install 사용
go install ./cmd/sage-crypto
```

### 테스트 실행

```bash
# 모든 테스트 실행
go test ./crypto/...

# 상세 출력과 함께 테스트
go test -v ./crypto/...

# 특정 패키지 테스트
go test ./crypto/keys
go test ./crypto/formats
go test ./crypto/storage
go test ./crypto/rotation
go test ./crypto/chain
go test ./crypto/chain/ethereum
go test ./crypto/chain/solana
```

## 사용 방법

### 1. 프로그래밍 방식 사용

#### 키 쌍 생성

```go
package main

import (
    "fmt"
    "github.com/sage-x-project/sage/crypto/keys"
)

func main() {
    // Ed25519 키 쌍 생성
    ed25519Key, err := keys.GenerateEd25519KeyPair()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Ed25519 Key ID: %s\n", ed25519Key.ID())

    // Secp256k1 키 쌍 생성
    secp256k1Key, err := keys.GenerateSecp256k1KeyPair()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Secp256k1 Key ID: %s\n", secp256k1Key.ID())
}
```

#### 키 내보내기/가져오기

```go
import (
    "github.com/sage-x-project/sage/crypto"
    "github.com/sage-x-project/sage/crypto/formats"
)

// JWK 형식으로 내보내기
exporter := formats.NewJWKExporter()
jwkData, err := exporter.Export(keyPair, crypto.KeyFormatJWK)

// JWK 형식에서 가져오기
importer := formats.NewJWKImporter()
importedKey, err := importer.Import(jwkData, crypto.KeyFormatJWK)

// PEM 형식으로 내보내기
pemExporter := formats.NewPEMExporter()
pemData, err := pemExporter.Export(keyPair, crypto.KeyFormatPEM)
```

#### 키 저장소 사용

```go
import "github.com/sage-x-project/sage/crypto/storage"

// 메모리 저장소 생성
memStorage := storage.NewMemoryKeyStorage()

// 파일 저장소 생성
fileStorage, err := storage.NewFileKeyStorage("./keys")

// 키 저장
err = fileStorage.Store("my-key", keyPair)

// 키 로드
loadedKey, err := fileStorage.Load("my-key")

// 키 목록 조회
keyIDs, err := fileStorage.List()
```

#### 메시지 서명 및 검증

```go
// 메시지 서명
message := []byte("Hello, SAGE!")
signature, err := keyPair.Sign(message)

// 서명 검증
err = keyPair.Verify(message, signature)
if err == nil {
    fmt.Println("Signature verified!")
}
```

### 2. CLI 도구 사용

#### 키 생성

```bash
# Ed25519 키 생성 (JWK 형식 출력)
./sage-crypto generate --type ed25519 --format jwk

# Secp256k1 키 생성하여 파일로 저장
./sage-crypto generate --type secp256k1 --format pem --output mykey.pem

# 키 생성하여 저장소에 저장
./sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id mykey
```

#### 메시지 서명

```bash
# JWK 키 파일로 서명
./sage-crypto sign --key mykey.jwk --message "Hello, World!"

# PEM 키로 파일 서명
./sage-crypto sign --key mykey.pem --format pem --message-file document.txt

# 저장소의 키로 서명
./sage-crypto sign --storage-dir ./keys --key-id mykey --message "Test message"

# stdin에서 메시지 읽어 서명 (base64 출력)
echo "Message to sign" | ./sage-crypto sign --key mykey.jwk --base64
```

#### 서명 검증

```bash
# 공개키와 base64 서명으로 검증
./sage-crypto verify --key public.jwk --message "Hello, World!" --signature-b64 "base64sig..."

# 서명 파일로 검증
./sage-crypto verify --key mykey.pem --format pem --message-file document.txt --signature-file sig.json
```

#### 키 회전

```bash
# 키 회전 (이전 키 삭제)
./sage-crypto rotate --storage-dir ./keys --key-id mykey

# 키 회전 (이전 키 보관)
./sage-crypto rotate --storage-dir ./keys --key-id mykey --keep-old
```

#### 키 목록 조회

```bash
# 저장소의 모든 키 목록
./sage-crypto list --storage-dir ./keys
```

#### 블록체인 주소 생성

```bash
# Ed25519 키로 Solana 주소 생성
./sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id alice-sol
./sage-crypto address generate --storage-dir ./keys --key-id alice-sol --chain solana

# Secp256k1 키로 Ethereum 주소 생성
./sage-crypto generate --type secp256k1 --format storage --storage-dir ./keys --key-id alice-eth
./sage-crypto address generate --storage-dir ./keys --key-id alice-eth --chain ethereum

# 모든 호환 가능한 블록체인 주소 생성
./sage-crypto address generate --storage-dir ./keys --key-id alice-eth --all
```

#### 블록체인 주소 파싱

```bash
# Ethereum 주소 파싱 및 검증
./sage-crypto address parse 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80

# Solana 주소 파싱 및 검증
./sage-crypto address parse 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM
```

## 블록체인 지원

### 지원되는 블록체인

| 블록체인 | 필요한 키 타입 | 주소 형식 | 공개키 복구 |
|---------|--------------|----------|------------|
| Ethereum | Secp256k1 | 0x로 시작하는 40자 hex | ❌ |
| Solana | Ed25519 | Base58 인코딩 | ✅ |

### 프로그래밍 방식으로 블록체인 주소 사용

```go
import (
    "github.com/sage-x-project/sage/crypto/chain"
    "github.com/sage-x-project/sage/crypto/keys"
)

// Ethereum 주소 생성
secp256k1Key, _ := keys.GenerateSecp256k1KeyPair()
ethProvider, _ := chain.GetProvider(chain.ChainTypeEthereum)
ethAddress, _ := ethProvider.GenerateAddress(
    secp256k1Key.PublicKey(), 
    chain.NetworkEthereumMainnet,
)

// Solana 주소 생성
ed25519Key, _ := keys.GenerateEd25519KeyPair()
solProvider, _ := chain.GetProvider(chain.ChainTypeSolana)
solAddress, _ := solProvider.GenerateAddress(
    ed25519Key.PublicKey(),
    chain.NetworkSolanaMainnet,
)

// 키로부터 모든 호환 주소 생성
addresses, _ := chain.AddressFromKeyPair(secp256k1Key)
for chainType, address := range addresses {
    fmt.Printf("%s: %s\n", chainType, address.Value)
}

// 주소 파싱 및 검증
parsedAddr, _ := chain.ParseAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80")
fmt.Printf("Chain: %s, Network: %s\n", parsedAddr.Chain, parsedAddr.Network)

// Solana 주소에서 공개키 복구
solPubKey, _ := solProvider.GetPublicKeyFromAddress(
    ctx, 
    "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
    chain.NetworkSolanaMainnet,
)
```

### 새로운 블록체인 추가하기

새로운 블록체인을 지원하려면 `ChainProvider` 인터페이스를 구현하세요:

```go
type MyChainProvider struct{}

func (p *MyChainProvider) ChainType() chain.ChainType {
    return "mychain"
}

func (p *MyChainProvider) GenerateAddress(publicKey crypto.PublicKey, network chain.Network) (*chain.Address, error) {
    // 주소 생성 로직 구현
}

// 다른 메서드들 구현...

// Provider 등록
func init() {
    chain.RegisterProvider(&MyChainProvider{})
}
```

## 실제 사용 예제

### 1. 전체 워크플로우 예제 (블록체인 포함)

```bash
# 1. 키 저장소 디렉토리 생성
mkdir -p ./my-keys

# 2. Ed25519 키 생성 및 저장
./sage-crypto generate --type ed25519 --format storage \
    --storage-dir ./my-keys --key-id alice-key

# 3. 키 목록 확인
./sage-crypto list --storage-dir ./my-keys

# 4. 메시지 서명
echo "Important message from Alice" | ./sage-crypto sign \
    --storage-dir ./my-keys --key-id alice-key \
    --output alice-signature.json

# 5. 서명 검증
./sage-crypto verify --storage-dir ./my-keys --key-id alice-key \
    --message "Important message from Alice" \
    --signature-file alice-signature.json

# 6. 블록체인 주소 생성
./sage-crypto address generate --storage-dir ./my-keys --key-id alice-key --all

# 7. 키 회전
./sage-crypto rotate --storage-dir ./my-keys --key-id alice-key --keep-old
```

### 2. JWK 형식 사용 예제

```bash
# JWK 키 생성
./sage-crypto generate --type ed25519 --format jwk --output alice.jwk

# JWK 키로 서명
./sage-crypto sign --key alice.jwk --message "Test message" --output signature.json

# 서명 검증
./sage-crypto verify --key alice.jwk --message "Test message" --signature-file signature.json
```

### 3. PEM 형식 사용 예제

```bash
# PEM 키 생성
./sage-crypto generate --type secp256k1 --format pem --output bob.pem

# PEM 키로 파일 서명
echo "Document content" > document.txt
./sage-crypto sign --key bob.pem --format pem --message-file document.txt --base64

# Base64 서명을 직접 검증
./sage-crypto verify --key bob.pem --format pem --message-file document.txt \
    --signature-b64 "MEUCIQDx..."
```

### 4. 블록체인 통합 예제

```bash
# Ethereum과 Solana를 위한 키 생성
./sage-crypto generate --type secp256k1 --format storage --storage-dir ./keys --key-id eth-key
./sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id sol-key

# 각 키에 대한 블록체인 주소 생성
./sage-crypto address generate --storage-dir ./keys --key-id eth-key --all
./sage-crypto address generate --storage-dir ./keys --key-id sol-key --all

# 특정 체인의 주소만 생성
./sage-crypto address generate --storage-dir ./keys --key-id eth-key --chain ethereum
./sage-crypto address generate --storage-dir ./keys --key-id sol-key --chain solana

# 주소 검증
./sage-crypto address parse 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80
./sage-crypto address parse 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM

# JSON 형식으로 주소 내보내기
./sage-crypto address generate --storage-dir ./keys --key-id eth-key --all --output addresses.json
```

## 보안 고려사항

1. **키 파일 권한**: 생성된 키 파일은 자동으로 `0600` 권한으로 설정되어 소유자만 읽고 쓸 수 있습니다.

2. **키 회전**: 정기적인 키 회전을 권장합니다. `--keep-old` 옵션을 사용하면 이전 키를 보관할 수 있습니다.

3. **저장소 보안**: 파일 기반 저장소를 사용할 때는 디렉토리 권한을 적절히 설정하세요.

## 지원되는 알고리즘

- **Ed25519**: 빠르고 안전한 EdDSA 서명 알고리즘
- **Secp256k1**: Bitcoin과 Ethereum에서 사용되는 타원 곡선

## 지원되는 형식

- **JWK (JSON Web Key)**: JSON 기반의 표준 키 형식
- **PEM (Privacy Enhanced Mail)**: Base64로 인코딩된 텍스트 형식

## 문제 해결

### 키를 찾을 수 없음
```
Error: key not found
```
저장소 디렉토리와 키 ID를 확인하세요.

### 잘못된 서명
```
❌ Signature verification FAILED
```
올바른 키와 메시지를 사용하고 있는지 확인하세요.

### 권한 오류
```
failed to create key storage directory: permission denied
```
디렉토리에 대한 쓰기 권한이 있는지 확인하세요.

### 잘못된 키 타입
```
Error: invalid public key: Ethereum requires secp256k1 keys
```
블록체인에 맞는 키 타입을 사용하세요:
- Ethereum: Secp256k1
- Solana: Ed25519

### 지원되지 않는 블록체인
```
Error: unsupported chain: bitcoin
```
현재 Ethereum과 Solana만 지원됩니다. 새로운 블록체인은 ChainProvider를 구현하여 추가할 수 있습니다.

## 라이선스

SAGE 프로젝트의 일부로 제공됩니다.