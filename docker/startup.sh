#!/bin/sh
# Note: Not using set -e because we handle errors explicitly in the wait loops

# If using local AI provider, wait for the model to be available
if [ "$AI_PROVIDER" = "local" ] && [ "$AI_ENABLED" = "true" ]; then
    MODEL="${AI_MODEL:-llava:7b}"
    OLLAMA_URL="${LOCAL_AI_URL:-http://ollama:11434}"

    echo "Waiting for Ollama model '$MODEL' to be ready..."

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

    # Wait for model to be downloaded (up to 10 minutes)
    MAX_WAIT=600
    WAITED=0
    MODEL_READY=false
    while [ $WAITED -lt $MAX_WAIT ]; do
        if curl -sf "$OLLAMA_URL/api/tags" 2>/dev/null | grep -q "\"name\":\"$MODEL\""; then
            MODEL_READY=true
            break
        fi
        echo "  Model '$MODEL' not yet available, waiting... (${WAITED}s)"
        sleep 10
        WAITED=$((WAITED + 10))
    done

    if [ "$MODEL_READY" = "true" ]; then
        echo "Model '$MODEL' is ready!"
    else
        echo "WARNING: Model '$MODEL' not available after ${MAX_WAIT}s, starting anyway (will use fallback)"
    fi
fi

# Start supervisord
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
