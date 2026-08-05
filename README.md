<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/badge/Charrade-E11D48?style=for-the-badge&labelColor=1a1a2e&color=E11D48">
    <source media="(prefers-color-scheme: light)" srcset="https://img.shields.io/badge/Charrade-E11D48?style=for-the-badge&labelColor=fff1f2&color=E11D48">
    <img src="https://img.shields.io/badge/Charrade-E11D48?style=for-the-badge&color=E11D48" alt="Charrade">
  </picture>
</p>

<h3 align="center">Act it out. Guess it right. Beat the clock.</h3>

<p align="center">
  A real-time, browser-based <strong>charades</strong> party game.<br>
  One player silently acts out a word on camera while teammates race to guess it before time runs out.
</p>

---

## How it works

1. **Create a room** — Pick a name and avatar, then share the room code with friends.
2. **Act it out** — The actor mimes a secret word on camera. No talking allowed!
3. **Guess & score** — Teammates type guesses in real time. Correct guesses earn points for both the guesser and the actor.

The game cycles through players as actors so everyone gets a turn, and a final scoreboard wraps it up.

## Features

- **Real-time gameplay** via WebSockets — guess text appears instantly for all players
- **Live video** — actor's camera is streamed to guessers using peer-to-peer WebRTC (no server relay needed)
- **Auto-advancing rounds** — the next actor is picked automatically after each round ends
- **Responsive, animated UI** — built with React and Motion for smooth transitions
- **No accounts required** — just a name, an avatar, and a room code

## Tech stack

| Layer     | Technology                              |
| --------- | --------------------------------------- |
| Backend   | Go, `net/http`, `gorilla/websocket`     |
| Frontend  | React 18, TypeScript, Vite              |
| Real-time | WebSocket (state sync) + WebRTC (video) |
| Animation | Motion (Framer Motion)                  |
| Monorepo  | Turborepo, pnpm workspaces              |

## Project structure

```
charrade/
├── apps/
│   ├── backend/               # Go server
│   │   ├── cmd/server/         # Entrypoint (main.go)
│   │   └── internal/
│   │       ├── config/         # Env-driven configuration
│   │       ├── game/           # Pure game logic (Room, Round, Guess, Words)
│   │       └── server/         # HTTP API, WebSocket hub, signaling relay
│   │           ├── server.go   # Wiring + New()
│   │           ├── handler.go  # HTTP handlers
│   │           ├── dispatch.go # WS message routing + scheduling
│   │           ├── signal.go   # WebRTC relay
│   │           ├── middleware.go # CORS, health
│   │           ├── hub.go      # WebSocket connection hub
│   │           ├── store.go    # In-memory room store
│   │           └── types.go    # Request/response types, constants
│   └── web/                    # React + TypeScript frontend
│       └── src/
│           ├── pages/          # Home, Start, Room, Score
│           ├── hooks/          # useGame, useRoomVideo
│           ├── lib/            # game types, API client, WS client, WebRTC, reducer
│           └── components/    # AvatarSelector, PlayerList
├── Dockerfile                  # Multi-stage build
├── docker-compose.yml          # Server + TURN server
├── lefthook.yml                # Git hooks (lint, test, conventional commits)
├── .golangci.yml               # Go linter configuration
├── turbo.json                  # Turborepo task config
└── package.json                # Root scripts (dev, build, test, lint)
```

## Getting started

### Prerequisites

- **Go** 1.22+ (used: 1.26)
- **Node.js** 18+
- **pnpm** 9.12+

### Install

```sh
git clone https://github.com/hey-amanthakur/charrade.git
cd charrade
pnpm install
```

### Run

```sh
# Start both backend and frontend
pnpm dev

# Or start individually
pnpm dev --filter=backend   # Go server on :8080
pnpm dev --filter=web       # Vite dev server on :5173
```

The Vite dev server proxies `/api/v1` requests to the Go backend automatically.

### Other commands

| Command       | Description                             |
| ------------- | --------------------------------------- |
| `pnpm build`  | Build backend binary + frontend bundle  |
| `pnpm test`   | Run Go tests + Vitest                   |
| `pnpm lint`   | `go vet` + ESLint                       |
| `pnpm format` | Prettier (TS/CSS/JSON); Go uses `gofmt` |

## API overview

| Method | Endpoint                          | Description                                |
| ------ | --------------------------------- | ------------------------------------------ |
| `POST` | `/api/v1/rooms`                   | Create a new room                          |
| `GET`  | `/api/v1/rooms/{id}`              | Get room state                             |
| `POST` | `/api/v1/rooms/{id}/players`      | Join an existing room                      |
| `GET`  | `/api/v1/rooms/{id}/ws?playerId=` | WebSocket connection for real-time updates |
| `GET`  | `/api/v1/health`                  | Health check                               |

### WebSocket message types

**Client → Server**

| Type         | Description                          |
| ------------ | ------------------------------------ |
| `start`      | Host starts the game                 |
| `startRound` | Host starts the next round           |
| `endRound`   | Host ends the current round early    |
| `guess`      | Submit a guess (`{ text: "..." }`)   |
| `signal`     | WebRTC signaling (`{ to, payload }`) |

**Server → Client**

| Type     | Description                                  |
| -------- | -------------------------------------------- |
| `state`  | Full room state (word hidden for non-actors) |
| `error`  | Error message                                |
| `signal` | Relayed WebRTC signaling                     |

## Environment variables

| Variable               | Where    | Description                                   |
| ---------------------- | -------- | --------------------------------------------- |
| `ADDR`                 | Backend  | Server address (default `:8080`)              |
| `ROUND_DURATION`       | Backend  | Round duration (default `60s`)                |
| `NEXT_ROUND_DELAY`     | Backend  | Delay between rounds (default `4s`)           |
| `VITE_TURN_URL`        | Frontend | TURN server URL (optional, for NAT traversal) |
| `VITE_TURN_USERNAME`   | Frontend | TURN credentials                              |
| `VITE_TURN_CREDENTIAL` | Frontend | TURN credentials                              |

## License

This project is proprietary software. All rights are reserved by the author. See [LICENSE](./LICENSE) for details.
