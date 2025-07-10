#!/bin/bash
# Resume logic for interrupted swarm tasks

CHECKPOINT_FILE="/swarm-state/checkpoint.json"

if [ ! -f "$CHECKPOINT_FILE" ]; then
    echo "❌ No checkpoint file found, starting fresh"
    exit 0
fi

echo "🔄 Resuming swarm task from checkpoint..."

# Load checkpoint data
CHECKPOINT_STEP=$(jq -r '.step' "$CHECKPOINT_FILE")
TASK_NAME=$(jq -r '.environment.task_name' "$CHECKPOINT_FILE")
SWARM_ID=$(jq -r '.environment.swarm_id' "$CHECKPOINT_FILE")

echo "📊 Resume Information:"
echo "  Task: $TASK_NAME"
echo "  Swarm: $SWARM_ID"
echo "  Last Step: $CHECKPOINT_STEP"

# Export environment variables
export TASK_NAME
export SWARM_ID
export RESUME_FROM_STEP="$CHECKPOINT_STEP"

# Load the checkpoint
/scripts/checkpoint.sh load

echo "✅ Ready to resume execution from step: $CHECKPOINT_STEP"