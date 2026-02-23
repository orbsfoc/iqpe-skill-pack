---
doc_id: TECH-GO-INT-001
title: TECH-GO-INT-001 Golang Integration Guidance
doc_type: architecture
concern: integration
status: accepted
owner_role: principal_architect
accountable_role: platform_lead
source_of_truth: true
version: '1.0'
---

# Golang Integration Guidance

## Required patterns
1. Port/adapter boundaries between domain and external systems.
2. Context-first I/O with explicit timeout budgets.
3. Adapter-layer error mapping with correlation ID propagation.
4. Bounded retries only for transient faults and idempotent operations.
5. Contract/schema validation at integration boundaries.

## Authority
- Authoritative Source: Corporate architecture integration guidance
- Approval Owner: platform_lead
- Approval Status: APPROVED
