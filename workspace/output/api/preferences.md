# API Specification Preferences

This document captures all user directives, decisions, and clarifications made during the API specification phase.

---

## Entry 1: gRPC Conventions
**Type**: Directive
**User Input**: "No specific preferences - proceed with standard gRPC conventions (snake_case, standard proto3)"
**Context**: User preference for gRPC API design conventions
**Impact**:
- Use proto3 syntax
- Field names use snake_case
- Standard protobuf message structures
- No custom metadata or correlation fields required
**Date**: 2025-12-23

---

## Summary of Key Decisions

### General API Standards
- **Protocol**: gRPC over HTTP/2
- **Serialization**: Protocol Buffers (proto3)
- **Field Naming**: snake_case (standard protobuf convention)
- **Package Versioning**: v1 suffix in package name

### Endpoint Design Patterns
- **Service Name**: `PricingService`
- **RPC Naming**: Verb + Noun (GetAsk, StreamBid)
- **Unary RPCs**: Get operations for single responses
- **Streaming RPCs**: Stream prefix for continuous updates

### Data Model Strategy
- **Decimal Values**: String representation for precision
- **Timestamps**: int64 epoch seconds
- **Optional Fields**: Use proto3 optional keyword
- **Enums**: Prefix with type name (CONTRACT_TYPE_CALL)

### Error Handling
- **Format**: Standard gRPC status codes
- **Messages**: Clear, actionable with field names
- **No Custom Details**: Standard status message sufficient

### Authentication
- **Type**: None (service-level security)
- **TLS**: Required in production
- **Rate Limiting**: Applied at load balancer

---

## Inherited Preferences

The following decisions from prior phases influenced API design:

| Source | Decision | API Impact |
|--------|----------|------------|
| PRD Entry 8 | Commission hidden | Not exposed in responses |
| PRD Entry 11 | Specific error codes | Mapped to gRPC status codes |
| PRD Entry 12 | Tick no fallback | Documented in stream behavior |
| Domain Entry 1 | Value objects | Flat message structures |

---

**End of Document**
