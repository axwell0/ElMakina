# ElMakina Frontend

Realtime UI for the ElMakina game. Built with React + TypeScript + Vite and styled with Tailwind.

## Quick start
- Install dependencies
- Run the dev server
- Ensure the backend WebSocket is available (default: `ws://localhost:8080/ws`)

## Mock UI mode
For UI-only development without the backend, you can enable mock fixtures:
- Env: `NEXT_PUBLIC_USE_MOCKS=true`
- URL: add `?mock=1` to the dev server URL

### Mock scenarios
Use `?mock=<scenario>` to force a specific screen:
- `?mock=game` (default)
- `?mock=lobby`
- `?mock=reveal`
- `?mock=gameover`

You can also set `NEXT_PUBLIC_MOCK_SCENARIO` to one of the above.
## Scripts
- `npm run dev` - start Vite dev server
- `npm run build` - typecheck and build
- `npm run lint` - lint the codebase
- `npm run preview` - preview production build

## Project layout
- `src/App.tsx` - top-level app switch (connecting/lobby/game)
- `src/components` - UI components and overlays
- `src/components/game` - game view subcomponents (layout, HUD)
- `src/store` - game reducer and context
- `src/network` - WebSocket client
- `src/App.css` - custom animation and effects

## Notes
- UI is optimized for smooth animation; prefer `transform` and `opacity` transitions.
- See `docs/ui-guidelines.md` for layout and motion conventions.
