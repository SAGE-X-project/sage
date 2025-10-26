# SAGE API Specification

This directory contains the OpenAPI specification and documentation for the SAGE HTTP API.

## Contents

### 📄 OpenAPI Specification

- **`openapi.yaml`** (351 lines)
  - Complete OpenAPI 3.0.3 specification for SAGE HTTP API
  - Defines endpoints: `/a2a:sendMessage`, `/protected`, `/health`, `/debug`
  - Includes request/response schemas for A2A messaging
  - Server configurations: Production, Staging, Development

### 📚 Usage Examples

The `examples/` directory contains detailed usage examples:

- **`authentication.md`** - HPKE-based authentication flow
- **`sessions.md`** - Session lifecycle management
- **`signatures.md`** - RFC 9421 HTTP message signatures

### 🗂️ Schemas

The `schemas/` directory is currently empty. All component schemas are defined inline in `openapi.yaml` under the `components.schemas` section.

**Available Schemas:**
- `A2AMessageRequest` - Agent-to-agent message request
- `A2AMessageResponse` - Agent-to-agent message response
- `AgentMetadata` - Agent metadata information
- `Error` - Standard error response

## Usage

### Viewing the Specification

**Local Development:**
```bash
# Start SAGE server (serves OpenAPI spec)
cd cmd/sage-server
go run main.go

# Access spec
curl http://localhost:8081/api/openapi.yaml
curl http://localhost:8081/api/openapi.json
```

**Swagger UI:**
```bash
# Using Docker
docker run -p 8080:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/api:/api swaggerapi/swagger-ui

# Open browser
open http://localhost:8080
```

**Redoc:**
```bash
npx @redocly/cli preview-docs api/openapi.yaml
```

### Validating the Specification

```bash
# Using openapi-generator
npx @openapitools/openapi-generator-cli validate -i api/openapi.yaml

# Using redocly
npx @redocly/cli lint api/openapi.yaml
```

### Generating Client SDKs

```bash
# Python client
npx @openapitools/openapi-generator-cli generate \
  -i api/openapi.yaml \
  -g python \
  -o generated/python-client

# TypeScript/JavaScript client
npx @openapitools/openapi-generator-cli generate \
  -i api/openapi.yaml \
  -g typescript-axios \
  -o generated/typescript-client

# Go client
npx @openapitools/openapi-generator-cli generate \
  -i api/openapi.yaml \
  -g go \
  -o generated/go-client
```

## API Endpoints

### A2A Messaging

```http
POST /a2a:sendMessage
Content-Type: application/json

{
  "sender_did": "did:sage:ethereum:0x123...",
  "receiver_did": "did:sage:ethereum:0x456...",
  "message": "eyJhbGciOiJIUEtFIn0...",
  "timestamp": 1234567890,
  "signature": "base64-encoded-signature"
}
```

### Protected Endpoint (RFC 9421)

```http
POST /protected
Signature-Input: sig1=("@method" "@path" "content-type");created=1234567890
Signature: sig1=:BASE64_SIGNATURE:
Content-Type: application/json

{
  "data": "example"
}
```

### Health Check

```http
GET /health

Response:
{
  "status": "healthy",
  "version": "1.3.0"
}
```

## Documentation

Full API documentation is available in:
- **[docs/API.md](../docs/API.md)** - Complete API reference (1139 lines)

The API.md file includes:
- Authentication mechanisms
- Session management
- HTTP signature verification
- Error handling
- Complete code examples

## Integration

The OpenAPI spec is referenced and served by:

1. **SAGE HTTP Server** (`cmd/sage-server`)
   - Serves spec at `/api/openapi.yaml` and `/api/openapi.json`
   - Implements all defined endpoints

2. **Documentation** (`docs/API.md`)
   - References examples in `api/examples/`
   - Provides detailed integration guides

3. **Future SDK Generation**
   - Spec can be used to auto-generate client SDKs
   - Ensures API consistency across languages

## Maintenance

### When to Update

Update `openapi.yaml` when:
- ✅ Adding new API endpoints
- ✅ Changing request/response schemas
- ✅ Modifying authentication requirements
- ✅ Adding new error codes

### Validation Checklist

Before committing changes:
- [ ] Run `npx @redocly/cli lint api/openapi.yaml`
- [ ] Verify examples still work
- [ ] Update docs/API.md if needed
- [ ] Test with Swagger UI

## Related Files

- `docs/API.md` - Full API documentation
- `cmd/sage-server/` - HTTP server implementation
- `pkg/agent/transport/http/` - HTTP transport layer
- `pkg/agent/core/rfc9421/` - HTTP signature verification

## Standards

- **OpenAPI:** 3.0.3
- **Authentication:** RFC 9421 (HTTP Message Signatures)
- **Encryption:** RFC 9180 (HPKE)
- **Identity:** W3C DID Core 1.0

## License

LGPL-3.0 (same as main project)

---

**Last Updated:** 2025-10-26
**API Version:** 1.0.0
**SAGE Version:** 1.3.0
