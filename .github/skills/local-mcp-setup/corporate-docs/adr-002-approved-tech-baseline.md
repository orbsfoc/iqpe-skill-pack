---
doc_id: ADR-POC-002
title: ADR-POC-002 Approved Technology Baseline and Integration Policy
doc_type: decision
concern: architecture
status: accepted
owner_role: principal_architect
accountable_role: platform_lead
source_of_truth: true
version: '1.0'
---

# ADR-POC-002 Approved Technology Baseline and Integration Policy

## Decision
- Backend runtime: Go 1.21+
- Primary persistence: PostgreSQL 15+
- Cache/session store: Redis 7.4
- API contract format: OpenAPI 3.1

## Authority
- Authoritative Source: Corporate architecture technology baseline + platform ADR governance
- Approval Owner: platform_lead
- Approval Status: APPROVED

## Consequences
- Teams may auto-approve technical decisions referencing this ADR.
- Local technical/adaptor choices outside this baseline remain PROPOSED or BLOCKED pending corporate approval.
