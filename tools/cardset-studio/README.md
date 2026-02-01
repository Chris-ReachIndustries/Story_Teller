# Card Set Studio

A standalone tool for creating Dixit card sets using AI-generated images.

## Features

- **AI Concept Generation**: Uses Ollama (llama3.2) to generate creative card concepts based on themes
- **Image Generation**: Uses Stable Diffusion (A1111) to generate card images
- **Review Workflow**: Approve, reject, or regenerate individual cards
- **Git LFS Export**: Export completed sets ready for the main Dixit repository

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Card Set Studio                       │
│  ┌─────────────────┐      ┌─────────────────┐          │
│  │  Next.js        │      │   Go Backend    │          │
│  │  Frontend       │◄────►│   API Server    │          │
│  │  (Port 3002)    │      │   (Port 3001)   │          │
│  └─────────────────┘      └────────┬────────┘          │
│                                    │                    │
│              ┌─────────────────────┼─────────────────┐  │
│              │                     │                 │  │
│              ▼                     ▼                 │  │
│  ┌─────────────────┐    ┌─────────────────┐         │  │
│  │     Ollama      │    │ Stable Diffusion│         │  │
│  │   (llama3.2)    │    │    (A1111)      │         │  │
│  │   Port 11434    │    │   Port 7860     │         │  │
│  └─────────────────┘    └─────────────────┘         │  │
└─────────────────────────────────────────────────────────┘
```

## Requirements

- Docker with Docker Compose
- NVIDIA GPU with 8GB+ VRAM (recommended)
- Git LFS installed (for exporting card sets)

## Quick Start

1. **Start all services:**
   ```bash
   cd tools/cardset-studio
   docker compose up -d
   ```

2. **Wait for services to be healthy:**
   ```bash
   docker compose ps
   ```

   The first run will download models:
   - Ollama: llama3.2:3b (~2GB)
   - Stable Diffusion: Base model (~4GB)

3. **Access the UI:**
   - Studio UI: http://localhost:3001
   - Stable Diffusion WebUI: http://localhost:7860 (optional)

## Workflow

### 1. Create a New Set

1. Click "New Set" on the home page
2. Enter a name (e.g., "Fantasy Ocean")
3. Enter a theme description (e.g., "underwater fantasy creatures, bioluminescent scenes")
4. Choose number of cards (default: 100)

### 2. Generate Concepts

1. Click "Generate Concepts"
2. AI will create titles and prompts for each card
3. Review and edit prompts as needed
4. Click "Generate Images" when ready

### 3. Generate Images

1. Click "Start Generation"
2. Watch progress as images are generated
3. Pause/resume at any time
4. Generation continues in background

### 4. Review Cards

1. Browse all generated cards
2. Click a card to open detail view
3. **Approve**: Card is ready for export
4. **Reject**: Remove from export
5. **Regenerate**: Create new image with same or modified prompt

### 5. Export

1. Click "Export Set"
2. Review summary of approved cards
3. Click "Export X Cards"
4. Follow the provided git commands to commit

## Git LFS Commands

After export, run these commands in the Dixit repository root:

```bash
# Track PNG files in the new set
git lfs track "cards/<set-name>/images/*.png"

# Add and commit
git add cards/<set-name>/
git commit -m "feat: add <set-name> card set"

# Push to remote
git push
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3001 | Backend API port |
| `OLLAMA_URL` | http://ollama:11434 | Ollama API URL |
| `SD_URL` | http://stable-diffusion:7860 | Stable Diffusion API URL |
| `OUTPUT_PATH` | /app/output | Working directory for sets |
| `EXPORT_PATH` | /app/dixit-cards | Export destination |

### Stable Diffusion Settings

Edit generation settings in the Generate page:
- Model selection
- Steps (default: 30)
- CFG Scale (default: 7)
- Sampler (default: DPM++ 2M Karras)
- Image size (default: 512x768)

## Development

### Backend (Go)

```bash
cd backend
go run ./cmd/studio
```

### Frontend (Next.js)

```bash
cd frontend
npm install
npm run dev
```

### Full Stack (Docker)

```bash
docker compose up --build
```

## Troubleshooting

### Services not starting

Check logs:
```bash
docker compose logs -f studio
docker compose logs -f ollama
docker compose logs -f stable-diffusion
```

### GPU not detected

Ensure NVIDIA Container Toolkit is installed:
```bash
nvidia-smi
docker run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi
```

### Slow generation

- Reduce image steps (20-30 is usually enough)
- Use a smaller model
- Check GPU memory usage

### Out of memory

- Reduce batch size
- Use lower resolution images
- Close other GPU-intensive applications

## License

Part of the Dixit project.
