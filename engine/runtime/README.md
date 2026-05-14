<!-- File: engine/runtime/README.md -->
<!-- Purpose: Explains the runtime orchestration boundary and its role in the architecture. -->
# Engine Runtime

`engine/runtime` is the home for turn orchestration, prompts, emitted events, and other live
match coordination concerns.

This package boundary lets transport adapters talk to runtime services while the runtime drives
the transport-agnostic core engine.
