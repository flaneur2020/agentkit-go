# agentkit-go

A thin Go wrapper for agent CLIs, starting with Claude.

## Design Principles

- Thin wrappers over over-designed abstractions.
- Explicit types over `any`.
- Typed, testable protocol handling over `io.Reader` / `io.Writer`.
- Spec-driven behavior with practical tests (including real CLI interaction).

## Non-Goals

- Having abstractions and "universal" interfaces for different agents.
