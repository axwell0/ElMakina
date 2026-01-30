# server

## Knowledge Context
Prerequisites:
- **`backend/engine`**: Generates the games that this server manages.
- **Concurrency in Go**: Mutexes and goroutines.

## Core Responsibility
The `server` package is the "Lobby Manager". It sits above the game engine and manages the lifecycle of:
1.  **Lobbies**: Rooms where players gather before a game.
2.  **Sessions**: Active game instances running in the engine.
3.  **Persistence**: Saving/loading lobby state (via `store`).
It is the binding layer that maps "User ID (String)" to "Player Index (Int)" and ensures concurrent access safety.

## Inner Workings and Logic
- **Lobby Management**: `Lobby` struct acts as a state machine (Open -> Playing -> Finished). It maintains a list of "Players" (Server concept) which are effectively just names and IDs at this stage.
- **Session Mapping**: Once a game starts, `GameSession` is created. It holds a mapping of `PlayerID -> EngineIndex`. This is crucial because the Engine only knows about `Player 0`, `Player 1`, while the Server knows about `User "alice"`, `User "bob"`.
- **Concurrency**: A heavy reliance on `sync.RWMutex` protects the lobby list and individual lobby states.

## Key Architectural Patterns
- **Repository Pattern (Partial)**: `LobbyStore` abstracts the storage of lobbies, allowing for in-memory or database backends (though simple fs/json might be used).
- **Adapter/Facade**: The package acts as a facade over the `engine`, simplifying the interface for the transport layer (`ws`).

## Critical Components
- **`LobbyManager`**: The singleton-like entity creating and retrieving lobbies.
- **`Lobby`**: Represents a room.
- **`GameSession`**: The bridge object binding a live engine instance to a lobby.
- **`LobbyStore`**: Interface for persistence.

## External Dependencies
- **`backend/engine`**: To start `NewGame`.
- **`backend/models`**: Shared types.
