# TL-NAME: Requirement title

Status: proposed

## Purpose and canonical context

Explain the user need and link the relevant system-design and architecture sections. Identify whether this adds or changes existing behavior.

## Required behavior

Describe inputs, outputs, invariants, failure behavior, and non-goals. Specify CLI/JSON/configuration contracts where affected. Do not substitute a chosen library for a behavioral requirement.

## Acceptance criteria

- TL-NAME/AC-1: Given … when … then … . Include observable state and errors.
- TL-NAME/AC-2: Failure, cancellation, concurrency, and recovery behavior where relevant.

## Compatibility and performance

Describe existing records/configuration/schema compatibility, migration needs, resource limits, and benchmark workloads. State what is not yet measured.

## Implementation boundaries

Identify package ownership, shared contracts, dependencies, and independent worktree assignments. Link architectural decisions rather than duplicating them.

## Verification and status

Map each criterion to test level and, once implemented, exact test file/symbol. Record commands and results, unresolved criteria, and rollout constraints. Update status only when evidence supports it.
