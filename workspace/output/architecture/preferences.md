# Architecture Phase Preferences

This document captures all user directives, decisions, and clarifications made during the architecture specification phase for the Digital Call/Put Options Pricing Service.

---

## Entry 1: Scaffolding Tool
**Type**: Directive
**User Input**: "I want package structure of the service to be create using go-templates"
**Context**: User specified scaffolding tool for project structure
**Impact**:
- Use `github.com/junbon-deriv/go-templates` for project scaffolding
- Follow the template's standard package layout
- Service template type selected
- Module path: `github.com/regentmarkets/service-pricer-digitalcallput`
**Date**: 2025-12-23

---

## Summary of Key Decisions

### Service Boundaries
- **Single Service**: One microservice (digitalcallput) - no multi-service architecture
- **Domain Alignment**: Service implements entire Pricing Domain
- **Stateless Design**: No persistent storage, no database

### Internal Structure
- **Complexity Score**: 6/10 (Standard) → Light modular organization
- **Scaffolding**: go-templates with service template
- **Package Layout**: cmd, config, internal (api, pricing, feed, types), proto

### Communication Patterns
- **Client Protocol**: gRPC (proto3)
- **Dependency Protocol**: gRPC (service-feed)
- **Stream Support**: Server streaming for real-time updates

### Data Strategy
- **No Persistence**: All data is transient
- **Configuration**: YAML files, environment variable overrides
- **Market Data**: Subscribed from service-feed (not stored)

### Technology Preferences
- **Language**: Go 1.21+
- **Protocol**: gRPC/HTTP2
- **Serialization**: Protocol Buffers (proto3)
- **Containerization**: Docker
- **Metrics**: Prometheus-compatible
- **Logging**: Structured JSON

---

## Entry 2: Implementation Rules (STRICT)
**Type**: Directive
**User Input**: Added strict implementation rules for roo-pe to follow
**Context**: Ensuring clean architecture patterns during implementation
**Impact**:

1. **Handler Responsibility Rule**
   - gRPC handlers ONLY receive requests and delegate to higher-level modules
   - Handlers MUST NOT fetch data from multiple sources and assemble responses
   
2. **Error Handling Rule**
   - Always return standard gRPC error codes (InvalidArgument, FailedPrecondition, OutOfRange, Unavailable, Internal)
   - Never create custom error types for API responses

3. **Dependency Direction Rule**
   - Dependencies MUST point towards the core (`pricing`)
   - `pricing` package MUST NOT depend on any other internal package
   - Only stdlib and proto dependencies allowed in core

4. **No Shared Types Package Rule**
   - Do NOT create `models/`, `types/`, `common/`, or `shared/` packages
   - Domain types belong in the core (`pricing`) package

5. **Interface Definition Rule**
   - Interfaces MUST be defined where they are consumed
   - NOT where they are implemented

**Date**: 2025-12-23

---

## Inherited Preferences

The following decisions from prior phases influenced architecture:

| Source | Decision | Architecture Impact |
|--------|----------|---------------------|
| PRD Entry 6 | Service name: digitalcallput | Repository naming, module path |
| PRD Entry 3 | Config in files | config package, YAML format |
| Domain Entry 1 | Simple value objects | internal/types package |
| PRD Section 11.1 | go-templates scaffolding | Project structure |

---

**End of Document**
