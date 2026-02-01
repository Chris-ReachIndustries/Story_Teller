# Stable Diffusion Models for Card Set Studio

This document explains how to set up high-quality Stable Diffusion models for generating card art.

## Recommended Models

### Option 1: JuggernautXL v9 (Recommended)

The default model configured in Card Set Studio. Excellent for all-purpose high-quality generation.

- **Download:** https://civitai.com/models/133005/juggernaut-xl
- **File:** `juggernautXL_v9Rundiffusion.safetensors`
- **Size:** ~6.5 GB
- **Type:** SDXL
- **Best for:** All-purpose fantasy art, portraits, landscapes

### Option 2: DreamShaper XL

Great for painterly, dreamlike fantasy art.

- **Download:** https://civitai.com/models/112902/dreamshaper-xl
- **File:** `dreamshaperXL.safetensors`
- **Size:** ~6.5 GB
- **Type:** SDXL
- **Best for:** Fantasy art with painterly style, ethereal imagery

### Option 3: Deliberate XL

Excellent for detailed illustrations with strong composition.

- **Download:** https://civitai.com/models/297778/deliberate-xl
- **File:** `deliberateXL.safetensors`
- **Size:** ~6.5 GB
- **Type:** SDXL
- **Best for:** Detailed fantasy illustrations, book cover art

## Installation

### Method 1: Docker Volume Mount (Recommended)

1. Download the model file from Civitai
2. Create a local directory for models:
   ```bash
   mkdir -p ./sd-models
   ```
3. Place the `.safetensors` file in this directory
4. Update `docker-compose.yml` to mount the directory:
   ```yaml
   stable-diffusion:
     volumes:
       - ./sd-models:/opt/stable-diffusion-webui/models/Stable-diffusion
   ```
5. Restart the containers:
   ```bash
   docker-compose down && docker-compose up -d
   ```

### Method 2: Using the SD WebUI

1. Start the containers: `docker-compose up -d`
2. Open the SD WebUI at http://localhost:7860
3. Use the WebUI's model download feature or manually upload via the interface

## Verifying Model Installation

After installation, verify the model is available:

```bash
curl http://localhost:7860/sdapi/v1/sd-models | jq
```

You should see your model listed. The system will automatically use the configured model.

## Hardware Requirements

SDXL models require more resources than SD 1.5 models:

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| GPU VRAM | 8 GB | 12+ GB |
| System RAM | 16 GB | 32 GB |
| Storage | 20 GB | 50 GB |
| GPU | RTX 3060 | RTX 3080/4070+ |

## Troubleshooting

### Model Not Loading

If the model fails to load:
1. Check VRAM usage - SDXL needs ~8GB minimum
2. Verify the file is not corrupted (re-download if needed)
3. Check container logs: `docker-compose logs stable-diffusion`

### Out of Memory Errors

If you encounter VRAM errors:
1. Use the "Fast" quality mode (smaller resolution)
2. Close other GPU-intensive applications
3. Consider using a smaller model (SD 1.5 based)

### Slow Generation

SDXL models are slower than SD 1.5. Expected times:
- Fast mode: 15-25 seconds per image
- Normal mode: 40-60 seconds per image
- High mode: 75-120 seconds per image

## Changing the Default Model

To use a different model, update the default in:
`backend/internal/storage/storage.go`

```go
func DefaultSDSettings() SDSettings {
    return SDSettings{
        Model: "your_model_name.safetensors",
        // ... other settings
    }
}
```

Rebuild and restart the containers after making changes.
