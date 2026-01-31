# Dixit Online

A multiplayer online Dixit-style game with real-time gameplay via WebSockets.

## Quick Start

```bash
# Build and run with Docker Compose
docker compose up --build

# Open in browser
# http://localhost:8080
```

For testing with 3 players (instead of 4):
```bash
MIN_PLAYERS=3 docker compose up --build
```

## Architecture

### Single Container Design

Everything runs in a single Docker container using supervisord:

```
┌─────────────────────────────────────────────────┐
│                Docker Container                 │
│                                                 │
│  ┌─────────────────┐    ┌──────────────────┐   │
│  │   Go Backend    │    │  Next.js Frontend │   │
│  │   Port 8080     │◄───│    Port 3000      │   │
│  │                 │    │                   │   │
│  │  /api/*         │    │   React App       │   │
│  │  /ws            │    │                   │   │
│  │  /cards/*       │    │                   │   │
│  │  /* (proxy)     │    │                   │   │
│  └─────────────────┘    └──────────────────┘   │
│         │                                       │
│         ▼                                       │
│  ┌─────────────────┐                           │
│  │   cards/        │                           │
│  │   ├── cards.json│                           │
│  │   └── images/   │                           │
│  └─────────────────┘                           │
└─────────────────────────────────────────────────┘
```

- **Go Backend** (port 8080): Handles API, WebSocket, static files, and reverse-proxies to Next.js
- **Next.js Frontend** (port 3000): React-based game UI
- **Cards Directory**: Runtime-loaded card definitions and images

### Tech Stack

- **Backend**: Go 1.21+ with gorilla/websocket
- **Frontend**: Next.js 14, React 18, TypeScript, Tailwind CSS
- **Process Manager**: supervisord
- **Testing**: Go testing, Jest, Playwright

## How to Play

1. Open the game in your browser
2. Enter your name
3. Create a new room or join an existing one with a 4-letter code
4. Share the room code with 3-5 friends (4-6 players total)
5. Host starts the game when ready

### Game Flow

1. **Storyteller Phase**: The storyteller selects a card and gives a creative clue
2. **Submission Phase**: Other players select cards that match the clue
3. **Voting Phase**: Everyone votes for which card they think is the storyteller's
4. **Scoring**: Points are awarded based on correct/incorrect guesses

### Scoring Rules

- If **everyone** or **nobody** guesses the storyteller's card:
  - Storyteller: 0 points
  - Everyone else: 2 points
- Otherwise:
  - Storyteller: 3 points
  - Each correct guesser: 3 points
- **Bonus**: +1 point for each vote your submitted card receives

### Room Configuration

The host can customize game settings when creating a room or in the lobby:

| Setting | Default | Range | Description |
|---------|---------|-------|-------------|
| **Score to Win** | 30 | 10-100 | First player to reach this score wins |
| **Hand Size** | 6 | 4-10 | Number of cards each player holds |
| **Card Set** | default | - | Which card set to use for this room |

### Per-Room Card Set Selection

Each room can use a different card set! When creating or configuring a room:

1. The host selects which card set to use from available sets
2. All players in the room see which set is selected in the lobby
3. Once the game starts, the card set cannot be changed
4. Each card set maintains its own images and metadata

This allows different rooms to run simultaneously with different themes.

### Reconnection

If you get disconnected mid-game:

1. Your reconnect token is saved in localStorage
2. Navigate back to the room URL within 10 minutes
3. You'll be automatically reconnected with your full game state restored

The server keeps disconnected players for 10 minutes before removing them. If a game is in progress and too many players disconnect, the game may end early.

## AI Bot Players

The game supports AI bot players that can participate in all game phases: acting as storyteller (creating clues), submitting cards, and voting. You can choose between two AI providers:

- **OpenAI** (default): Uses OpenAI's Vision API (requires API key, best quality)
- **Local**: Uses Ollama with vision models (runs offline, no API key needed)

### Adding Bots to a Game

1. Create or join a room as the host
2. Click the "Add AI Bot" button in the lobby
3. Add up to 5 bots (6 players total including yourself)
4. Start the game when ready

### How Bots Work

1. **As Storyteller**: Bot analyzes its hand of cards and creates a creative clue
2. **Submitting Cards**: Bot selects the card that best matches the storyteller's clue
3. **Voting**: Bot votes for the card it thinks is the storyteller's (cannot vote for its own card)

Bots have a small random delay (600-1200ms) before taking actions to feel more natural.

### OpenAI Mode (Default)

Uses OpenAI's GPT-4o Vision API for high-quality bot decisions.

```bash
# Linux/Mac
AI_ENABLED=true OPENAI_API_KEY=sk-your-key-here docker compose up --build

# Windows PowerShell
$env:AI_ENABLED="true"
$env:OPENAI_API_KEY="sk-your-key-here"
docker compose up --build
```

**OpenAI Configuration:**

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_ENABLED` | false | Enable AI bot support |
| `AI_PROVIDER` | openai | Provider selection (openai or local) |
| `OPENAI_API_KEY` | - | Your OpenAI API key (required for openai provider) |
| `AI_MODEL` | gpt-4o | OpenAI model (gpt-4o, gpt-4o-mini, etc.) |
| `AI_TIMEOUT_MS` | 12000 | Request timeout in milliseconds |
| `AI_MAX_TOKENS` | 250 | Maximum tokens for responses |
| `AI_TEMPERATURE` | 0.7 | Response creativity (0.0-2.0) |
| `AI_THUMB_WIDTH` | 320 | Thumbnail width for card images |

**Cost Considerations:**

Each bot decision sends card images to OpenAI's Vision API. Costs depend on:
- Number of bots in the game
- Number of rounds played
- Model used (gpt-4o-mini is cheaper than gpt-4o)

For cost-effective testing, use `AI_MODEL=gpt-4o-mini`.

### Local AI Mode (Offline)

Run AI bots entirely locally using Ollama. No OpenAI API key required.

The local AI uses a **two-stage pipeline** for better decision-making:
1. **Vision model** (LLaVA): Describes card images in natural language
2. **Text model** (llama3.2): Makes game decisions based on those descriptions

This approach plays to each model's strengths - LLaVA excels at image understanding, while llama3.2 follows game instructions reliably.

```bash
# Linux/Mac
AI_ENABLED=true AI_PROVIDER=local docker compose --profile local-ai up --build

# Windows PowerShell
$env:AI_ENABLED="true"
$env:AI_PROVIDER="local"
docker compose --profile local-ai up --build
```

**First run downloads two models (~6.5GB total: ~4.5GB for llava:7b + ~2GB for llama3.2:3b).**

**Local AI Configuration:**

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_PROVIDER` | openai | Set to `local` for Ollama |
| `LOCAL_AI_URL` | http://ollama:11434 | Ollama endpoint |
| `AI_MODEL` | llava:7b | Vision model for image description |
| `AI_TEXT_MODEL` | llama3.2:3b | Text model for game decisions |
| `AI_TIMEOUT_MS` | 180000 | Request timeout (3 min for local inference) |

**Available Vision Models:**

| Model | Size | Quality | Speed |
|-------|------|---------|-------|
| `llava:7b` | 4.5GB | Good | Fastest |
| `llava:13b` | 8GB | Better | Medium |
| `llama3.2-vision:11b` | 7GB | Best | Medium |

**Available Text Models:**

| Model | Size | Quality | Speed |
|-------|------|---------|-------|
| `llama3.2:3b` | 2GB | Good | Fastest |
| `llama3.2:8b` | 4.7GB | Better | Medium |
| `mistral:7b` | 4GB | Good | Medium |

To use different models:
```bash
AI_ENABLED=true AI_PROVIDER=local AI_MODEL=llava:13b AI_TEXT_MODEL=llama3.2:8b docker compose --profile local-ai up --build
```

**System Requirements for Local AI:**

| Setup | RAM | Response Time |
|-------|-----|---------------|
| CPU only | 8GB+ (16GB recommended) | 15-45 seconds |
| GPU (8GB+ VRAM) | 16GB+ | 3-8 seconds |

**Troubleshooting Local AI:**

- **Slow first startup**: Model download takes 5-10 minutes. Subsequent starts are fast.
- **Out of memory**: Use `llava:7b` (smallest) or increase Docker memory limit.
- **No GPU detected**: Install NVIDIA Container Toolkit for GPU support. CPU mode works but is slower.
- **Connection refused**: Ensure Ollama service is healthy: `docker compose --profile local-ai logs ollama`

## Automatic Card Generation

The game can automatically generate 100 unique Dixit-style card images using OpenAI's DALL-E 3 API on first startup.

### How It Works

On server startup, the system checks if a valid deck exists (100 images + cards.json). If not:

1. **Concept Generation**: GPT-4o generates 100 unique card concepts with titles, descriptions, and tags
2. **Image Generation**: DALL-E 3 creates images for each concept in portrait format (1024×1557, matching real Dixit card proportions)
3. **Persistence**: Cards are saved to the mounted `./cards` directory and persist across restarts

### Enabling Card Generation

```bash
AI_ENABLED=true OPENAI_API_KEY=sk-your-key docker compose up --build
```

The first startup will take several minutes while generating 100 images. Progress is logged:
```
[CardGen] Generating 100 card concepts via GPT-4o...
[CardGen] Progress: 5/100 images generated
...
[CardGen] Card generation complete! 100 cards ready.
```

### Generation Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_ENABLED` | false | Must be true for card generation |
| `OPENAI_API_KEY` | - | Required for generation |
| `CARD_SET_NAME` | default | Name of the card set (allows multiple sets) |
| `CARDS_REGENERATE` | false | Force regenerate even if valid deck exists |
| `DALLE_MODEL` | dall-e-3 | DALL-E model version |
| `DALLE_SIZE` | 1024x1792 | Portrait mode for card aspect ratio |
| `DALLE_QUALITY` | standard | Image quality (standard or hd) |
| `DALLE_CONCURRENCY` | 3 | Parallel API calls |

### Named Card Sets

You can create and manage multiple card sets, each stored in its own folder:

```bash
# Generate a new set named "fantasy"
CARD_SET_NAME=fantasy AI_ENABLED=true OPENAI_API_KEY=sk-... docker compose up --build

# Generate another set named "ocean"
CARD_SET_NAME=ocean AI_ENABLED=true OPENAI_API_KEY=sk-... docker compose up --build

# Switch to using the "fantasy" set (no generation if it exists)
CARD_SET_NAME=fantasy docker compose up
```

Card sets are stored in `./cards/<set_name>/`:
```
./cards/
├── default/           # Default card set
│   ├── cards.json
│   └── images/
├── fantasy/           # "fantasy" card set
│   ├── cards.json
│   └── images/
└── ocean/             # "ocean" card set
    ├── cards.json
    └── images/
```

### Cost Estimate

One-time generation costs approximately $4.15:
- GPT-4o for 100 concepts: ~$0.15
- DALL-E 3 × 100 images: ~$4.00

### Force Regeneration

To regenerate all cards (replacing existing ones):

```bash
CARDS_REGENERATE=true AI_ENABLED=true OPENAI_API_KEY=sk-... docker compose up
```

### Using Manual Cards (No AI)

To use pre-made cards without AI generation:

1. Create a folder for your set: `./cards/<set_name>/` (e.g., `./cards/custom/`)
2. Place your images in `./cards/<set_name>/images/` (card-001.png through card-100.png)
3. Create `./cards/<set_name>/cards.json` with card metadata
4. Run with `CARD_SET_NAME=<set_name> AI_ENABLED=false`

Example for a custom set:
```bash
# Create folder structure
mkdir -p ./cards/custom/images
# Add your images and cards.json, then:
CARD_SET_NAME=custom docker compose up
```

If cards are missing and AI is disabled, the server will show an error explaining what to do.

### Status Endpoint

Check generation progress: `GET /api/cardgen/status`

```json
{
  "deckValid": true,
  "targetCards": 100,
  "existingCards": 100
}
```

During generation:
```json
{
  "deckValid": false,
  "generation": {
    "inProgress": true,
    "conceptsGenerated": true,
    "imagesCompleted": 42,
    "imagesTotal": 100
  }
}
```

## Adding New Cards

Cards are loaded at runtime - no rebuild needed!

### Step 1: Edit cards.json

Add a new entry to `cards/cards.json`:

```json
{
  "id": "card-021",
  "title": "My New Card",
  "image": "/cards/images/card-021.svg",
  "tags": ["custom", "art"]
}
```

### Step 2: Add the Image

Place your image in `cards/images/`:
- SVG recommended (any format works)
- Suggested size: 300x400 pixels (3:4 aspect ratio)

### Step 3: Refresh

Just refresh the browser - the new card will appear in the game!

## Development

### Prerequisites

If developing locally (not in Docker):
- Go 1.21+
- Node.js 20+
- npm

### Backend Development

```bash
cd backend

# Run tests
go test ./...

# Format code
gofmt -w .
```

### Frontend Development

```bash
cd frontend

# Install dependencies
npm install

# Run dev server
npm run dev

# Run tests
npm test

# Run linting
npm run lint
```

### Running Tests

```bash
# Backend tests (in Docker)
docker compose exec dixit sh -c "cd /app/backend && go test ./..."

# Or build and test locally if Go is installed
cd backend && go test ./...

# Frontend tests (requires Node.js)
cd frontend && npm test

# E2E tests (requires Docker running)
cd frontend && npx playwright test
```

## Project Structure

```
dixit/
├── backend/
│   ├── cmd/server/         # Entry point
│   └── internal/
│       ├── game/           # Game logic, state machine, scoring
│       ├── ws/             # WebSocket handling
│       ├── cards/          # Card loading and thumbnails
│       ├── ai/             # AI client (OpenAI integration)
│       ├── bot/            # Bot controller for AI players
│       └── http/           # HTTP server, reverse proxy
├── frontend/
│   ├── app/                # Next.js pages
│   ├── components/         # React components
│   ├── lib/                # Types, WebSocket client, context
│   ├── __tests__/          # Jest tests
│   └── e2e/                # Playwright E2E tests
├── cards/
│   └── <set_name>/         # Named card sets (e.g., "default", "fantasy")
│       ├── cards.json      # Card definitions
│       └── images/         # Card images (PNG/SVG)
├── docker/
│   ├── supervisord.conf    # Process manager config
│   └── startup.sh          # Container startup script (waits for AI models)
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Game Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Backend port |
| `CARDS_PATH` | /app/cards | Base path for card sets |
| `CARD_SET_NAME` | default | Name of the card set to use |
| `NEXTJS_URL` | http://localhost:3000 | Next.js URL for proxy |
| `MIN_PLAYERS` | 4 | Minimum players to start (3-6) |
| `AI_ENABLED` | false | Enable AI bot support and card generation |
| `AI_PROVIDER` | openai | AI provider (openai or local) |
| `OPENAI_API_KEY` | - | OpenAI API key (for openai provider) |
| `AI_MODEL` | gpt-4o / llava:7b | Vision model (OpenAI or Ollama) |
| `AI_TEXT_MODEL` | llama3.2:3b | Text model for decisions (local AI only) |
| `AI_TIMEOUT_MS` | 12000 / 180000 | Request timeout (OpenAI / local) |
| `CARDS_REGENERATE` | false | Force regenerate card deck |
| `DALLE_MODEL` | dall-e-3 | DALL-E model for cards |
| `DALLE_QUALITY` | standard | Image quality (standard/hd) |

## API Endpoints

- `GET /api/health` - Health check
- `GET /api/cards` - List all cards (with metadata)
- `GET /api/card-sets` - List all available card sets
- `GET /api/card-sets/:id/cards` - Get all cards for a specific set
- `GET /api/cardgen/status` - Card generation status
- `GET /cards/<set_name>/images/*` - Static card images (e.g., `/cards/default/images/card-001.png`)
- `WS /ws` - WebSocket for real-time game state

### Card Sets API

**List all card sets:**
```
GET /api/card-sets
```

Response:
```json
{
  "sets": [
    {
      "id": "default",
      "name": "Default",
      "cardCount": 100,
      "previewUrls": ["/cards/default/images/card-001.png", ...]
    },
    {
      "id": "fantasy",
      "name": "Fantasy",
      "cardCount": 100,
      "previewUrls": ["/cards/fantasy/images/card-001.png", ...]
    }
  ]
}
```

**Get cards for a specific set:**
```
GET /api/card-sets/fantasy/cards
```

Response:
```json
{
  "cards": [
    {
      "id": "card-001",
      "title": "The Dragon's Lair",
      "image": "/cards/fantasy/images/card-001.png",
      "tags": ["dragon", "cave", "treasure"]
    }
  ],
  "total": 100
}

## WebSocket Protocol

### Client → Server

```json
{"type": "hello", "payload": {"name": "Player"}}
{"type": "hello", "payload": {"name": "Player", "reconnectToken": "...", "roomCode": "ABCD"}}
{"type": "create_room", "payload": {"config": {"scoreToWin": 30, "handSize": 6, "deckSetId": "fantasy"}}}
{"type": "join_room", "payload": {"code": "ABCD"}}
{"type": "update_config", "payload": {"config": {"scoreToWin": 25, "handSize": 5, "deckSetId": "default"}}}
{"type": "start_game", "payload": {}}
{"type": "add_bot", "payload": {}}
{"type": "story_submit", "payload": {"clue": "...", "cardId": "card-001"}}
{"type": "submit_card", "payload": {"cardId": "card-002"}}
{"type": "vote", "payload": {"submissionIndex": 2}}
```

### Server → Client

```json
{"type": "state", "payload": {
  "room": {"code": "ABCD", "hostId": "...", "config": {"scoreToWin": 30, "handSize": 6, "deckSetId": "fantasy"}},
  "you": {"id": "...", "name": "Player", "isHost": true, "reconnectToken": "..."},
  "phase": "LOBBY",
  "yourSubmissionIndex": 2
}}
{"type": "error", "payload": {"message": "..."}}
```

**Note**: `reconnectToken` is only sent once when first joining/creating a room. `yourSubmissionIndex` is only present during the VOTING phase to identify your own card.

## License

MIT
