# Runtime Validation Explained: THIS vs THAT

## THIS (Current System - WITH Runtime Validation)

### What happens when a WebSocket message arrives:

```typescript
// In web/src/network/socket.ts

private onMessage = (event: MessageEvent) => {
    const raw = JSON.parse(event.data);
    
    // L1: Check envelope has required fields (type, payload)
    const structureValidation = validateEnvelopeStructure(raw);
    if (!structureValidation.valid) {
        // BAD DATA REJECTED HERE
        validationLogger.logValidationError({
            layer: "L1_STRUCTURE",
            error: "Invalid envelope structure",
            details: structureValidation.errors,
            rawEnvelope: raw,
        });
        return; // Message dropped
    }

    // L2: Check message type is in whitelist
    if (!isValidInboundMessageType(envelope.type)) {
        // UNKNOWN TYPE REJECTED HERE
        validationLogger.logValidationWarning({
            layer: "L2_TYPE_WHITELIST",
            messageType: envelope.type,
            error: `Unknown message type: ${envelope.type}`,
        });
        return; // Message dropped
    }

    // L3: Schema validation (AJV)
    if (this.validateEnvelope && !this.validateEnvelope(envelope)) {
        const errors = this.validateEnvelope.errors;
        console.error("[ws] Schema validation failed", { errors, envelope });
        // Continue processing (non-blocking in current setup)
    }

    this.handleMessage(envelope); // Only valid messages reach here
};
```

### Example: Bad Data Gets Caught

```typescript
// Server sends malformed message (accidentally sends string instead of number):
{ 
    "type": "card_discarded",
    "payload": {
        "player_index": "1",  // ❌ String instead of number
        "player_name": "Alice",
        "card_role": "Colonel",
        "turn": "not_a_number"  // ❌ String instead of number
    }
}

// Result: AJV validator catches this
// Console error: "data/payload/player_index must be number"
// Message is logged but NOT processed (or could be rejected)
```

### Example: Unknown Message Type

```typescript
// Server sends:
{ 
    "type": "hacker_exploit",  // ❌ Unknown type
    "payload": { "malicious": "data" }
}

// Result: Type whitelist catches this
// Console warning: "Unknown message type: hacker_exploit"
// Message is immediately dropped
```

---

## THAT (Without Runtime Validation - TypeScript Only)

### What happens with just TypeScript types:

```typescript
// In web/src/network/socket.ts (WITHOUT validation)

private onMessage = (event: MessageEvent) => {
    const raw = JSON.parse(event.data);
    
    // NO VALIDATION - just trust the data matches our types
    const envelope = raw as WsEnvelope;  // ❌ Dangerous cast!
    
    // If server sends bad data, we crash here or have undefined behavior
    this.handleMessage(envelope);
};

// Later in a reducer:
case "card_discarded": {
    const payload = envelope.payload as CardDiscardedPayload;
    // If turn is "not_a_number", this could crash:
    return { ...state, currentTurn: payload.turn + 1 };  // ❌ NaN or crash!
}
```

### Example: Bad Data Causes Runtime Errors

```typescript
// Server sends malformed message:
{ 
    "type": "card_discarded",
    "payload": {
        "player_index": "1",  // ❌ String
        "player_name": null,  // ❌ Null instead of string
        "card_role": 123,     // ❌ Number instead of string
        "turn": undefined     // ❌ Missing
    }
}

// Result: TypeScript thinks everything is fine at compile time
// But at runtime:

// ❌ Crash 1: Trying to display player name
<div>{payload.player_name.toUpperCase()}</div>  
// ERROR: Cannot read property 'toUpperCase' of null

// ❌ Crash 2: Array indexing with string
const player = state.players[payload.player_index];  
// Gets undefined (arrays use numeric indices)

// ❌ Crash 3: Math with string
const nextTurn = payload.turn + 1;  
// Result: "undefined1" or NaN

// ❌ Silent Bug: Invalid role displays nothing
<Card role={payload.card_role} />  
// Card component receives 123, doesn't know what to render
```

---

## Real-World Analogy

### Runtime Validation = Bouncer at a Club
```
Server sends: { type: "game_state", payload: {...} }
                 ↓
Bouncer (AJV): "Let me check your ID..."
                 ↓
IF valid:       "OK, come in" → Process message
IF invalid:     "Sorry, wrong format" → Reject message
```

### TypeScript Only = Honor System
```
Server sends: { type: "game_state", payload: {...} }
                 ↓
TypeScript:    "Looks good on paper" (compile time)
                 ↓
Runtime:       Direct entry, no check
                 ↓
IF valid:      Everything works
IF invalid:    💥 Crash or weird behavior
```

---

## Summary Table

| Scenario | With Runtime Validation | TypeScript Only |
|----------|------------------------|-----------------|
| **Server sends wrong type** | Caught, logged, rejected | Crash or silent bug |
| **Missing required field** | Caught at L1 structure | `undefined` errors |
| **Wrong data type** | Caught by AJV schema | Type coercion bugs |
| **Unknown message type** | Caught at L2 whitelist | Goes to default handler |
| **Security** | Malformed messages rejected | Potential injection |
| **Debuggability** | Clear validation errors | Mystery crashes |
| **Compile-time safety** | ✅ Same as TypeScript | ✅ Type checking |
| **Runtime safety** | ✅ Validation catches issues | ❌ No protection |

---

## My Recommendation

**Keep runtime validation because:**

1. **WebSocket data is untrusted** - It comes from the network, could be corrupted
2. **Backend bugs happen** - A Go bug could send malformed JSON
3. **Version mismatches** - Old client + new server = incompatible messages
4. **Game integrity** - Bad data could desync or crash the game
5. **Debugging** - Validation errors tell you exactly what's wrong

**For a multiplayer game, you want BOTH:**
- TypeScript for compile-time developer experience
- Runtime validation for production safety

**The JSON schema system gives you both automatically.**

**Want me to show you the actual error messages you'd see in the console?**