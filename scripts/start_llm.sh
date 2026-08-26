#!/bin/bash
# Start llama-server with Qwen2.5-3B-Instruct for PatientTriage.ai caution-flag service
LLAMA_SERVER="/home/kernelghost/Desktop/GO/llama.cpp/build/bin/llama-server"
MODEL_PATH="/home/kernelghost/Desktop/GO/models/qwen2.5-3b-instruct-q4_k_m.gguf"

if [ ! -f "$LLAMA_SERVER" ]; then
    echo "ERROR: llama-server not found at $LLAMA_SERVER"
    exit 1
fi

if [ ! -f "$MODEL_PATH" ]; then
    echo "ERROR: Model not found at $MODEL_PATH"
    exit 1
fi

exec "$LLAMA_SERVER" \
    --model "$MODEL_PATH" \
    --port 8080 \
    --ctx-size 2048 \
    --threads $(nproc) \
    --log-disable
