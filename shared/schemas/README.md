# WebSocket Schema

# WebSocket Schema

This directory contains the canonical JSON Schema for the ElMakina WebSocket API.

- **`envelope.json`**: The machine-readable source of truth for all WebSocket messages.

## Automation Workflow

The schema is **NOT** edited manually. It is generated from the Backend Go code.

### 1. Source of Truth: Go Code
The Go structs in `backend/server/ws/messages.go` are the primary definition of the API.

### 2. Generation Cycle
To update the API contract, follow these steps:

1.  **Modify Go Structs**: Edit `backend/server/ws/messages.go`.
2.  **Run Generation**: Execute the following command from the project root:
    ```bash
    make gen-schema
    ```
    This command performs two actions:
    *   Runs `backend/scripts/generate-schema/` to generate `shared/schemas/envelope.json`.
    *   Runs `web/scripts/generate-ws-contract.mjs` to generate TypeScript types in `web/src/network/ws-contract.ts`.

### 3. Verification
The backend has a test suite that ensures the implementation complies with the schema:
- Run `backend/scripts/check-ws-contract.sh` to validate.

## Files
- `envelope.json`: The generated JSON Schema.
- `../../../backend/docs/ws_contract.md`: Human-readable documentation (manually maintained).

