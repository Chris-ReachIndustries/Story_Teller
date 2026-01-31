#!/bin/sh
# Note: Not using set -e because we handle errors explicitly in the wait loops

# If using local AI provider, wait for both models to be available
if [ "$AI_PROVIDER" = "local" ] && [ "$AI_ENABLED" = "true" ]; then
    VISION_MODEL="${AI_MODEL:-llava:7b}"
    TEXT_MODEL="${AI_TEXT_MODEL:-llama3.2:3b}"
    OLLAMA_URL="${LOCAL_AI_URL:-http://ollama:11434}"

    echo "Waiting for Ollama models to be ready..."
    echo "  Vision model: $VISION_MODEL"
    echo "  Text model: $TEXT_MODEL"

    # Wait for Ollama server (up to 2 minutes)
    SERVER_WAIT=0
    while [ $SERVER_WAIT -lt 120 ]; do
        if curl -sf "$OLLAMA_URL/api/tags" > /dev/null 2>&1; then
            echo "  Ollama server is up"
            break
        fi
        echo "  Waiting for Ollama server at $OLLAMA_URL... (${SERVER_WAIT}s)"
        sleep 5
        SERVER_WAIT=$((SERVER_WAIT + 5))
    done

    # Wait for vision model (up to 10 minutes)
    MAX_WAIT=600
    WAITED=0
    VISION_READY=false
    while [ $WAITED -lt $MAX_WAIT ]; do
        if curl -sf "$OLLAMA_URL/api/tags" 2>/dev/null | grep -q "\"name\":\"$VISION_MODEL\""; then
            VISION_READY=true
            break
        fi
        echo "  Vision model '$VISION_MODEL' not yet available, waiting... (${WAITED}s)"
        sleep 10
        WAITED=$((WAITED + 10))
    done

    if [ "$VISION_READY" = "true" ]; then
        echo "Vision model '$VISION_MODEL' is ready!"
    else
        echo "WARNING: Vision model '$VISION_MODEL' not available after ${MAX_WAIT}s"
    fi

    # Wait for text model (up to 10 minutes)
    WAITED=0
    TEXT_READY=false
    while [ $WAITED -lt $MAX_WAIT ]; do
        if curl -sf "$OLLAMA_URL/api/tags" 2>/dev/null | grep -q "\"name\":\"$TEXT_MODEL\""; then
            TEXT_READY=true
            break
        fi
        echo "  Text model '$TEXT_MODEL' not yet available, waiting... (${WAITED}s)"
        sleep 10
        WAITED=$((WAITED + 10))
    done

    if [ "$TEXT_READY" = "true" ]; then
        echo "Text model '$TEXT_MODEL' is ready!"
    else
        echo "WARNING: Text model '$TEXT_MODEL' not available after ${MAX_WAIT}s"
    fi

    if [ "$VISION_READY" = "true" ] && [ "$TEXT_READY" = "true" ]; then
        echo "All models ready!"
    else
        echo "WARNING: Some models not available, starting anyway (may use fallback)"
    fi
fi

# Start supervisord
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
