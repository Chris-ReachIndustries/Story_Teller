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
- NVIDIA GPU (see hardware requirements below)
- Git LFS installed (for exporting card sets)

### Hardware Requirements

For high-quality SDXL image generation:

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| GPU VRAM | 8 GB | 12+ GB |
| System RAM | 16 GB | 32 GB |
| Storage | 20 GB | 50 GB |
| GPU | RTX 3060 | RTX 3080/4070+ |

### Expected Generation Times

| Quality Mode | Time per Image | Resolution |
|--------------|----------------|------------|
| Fast | 15-25 seconds | 512x768 |
| Normal | 40-60 seconds | 768x1024 |
| High | 75-120 seconds | 896x1152 |

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
3. Enter a detailed theme description (see tips below)
4. Choose number of cards (default: 100)

**Theme Description Tips:**

Detailed themes produce better visual coherence across all cards:

- **Good:** "Underwater fantasy world with bioluminescent creatures, ancient coral castles, ethereal jellyfish, deep ocean blues and vibrant teals, mystical atmosphere"
- **Less effective:** "ocean fantasy"

The theme influences:
- Color palette across all cards
- Art style consistency
- Mood and atmosphere
- Visual coherence between cards

### 2. Generate Concepts

1. Click "Generate Concepts"
2. AI will create titles and prompts for each card
3. Review and edit prompts as needed
4. Click "Generate Images" when ready

### 3. Generate Images

1. **Select quality mode:**
   - **Fast:** Quick previews, lower resolution (~20s/image)
   - **Normal:** Good balance (~45s/image)
   - **High:** Best quality, recommended for final output (~90s/image)
2. Click "Start Generation"
3. Watch progress as images are generated
4. Pause/resume or cancel at any time
5. All cards in a set share a consistent visual style derived from the theme

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

Default high-quality settings (configurable per quality mode):
- Model: JuggernautXL v9 (SDXL)
- Steps: 45 (High mode), 30 (Normal), 20 (Fast)
- CFG Scale: 8.0
- Sampler: DPM++ 2M Karras
- Resolution: 896x1152 (High), 768x1024 (Normal), 512x768 (Fast)

See [MODELS.md](MODELS.md) for model installation instructions.

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

High-quality SDXL generation is intentionally slower for better results:

- Use "Fast" quality mode for quick previews
- Switch to "High" mode only for final output
- Expected times: 20-120 seconds per image depending on mode
- Check GPU memory usage with `nvidia-smi`

### Out of memory

- Reduce batch size
- Use lower resolution images
- Close other GPU-intensive applications

## License

Part of the Dixit project.
