#!/bin/bash
# Checkpoint management for swarm tasks

CHECKPOINT_DIR="/swarm-state"
CHECKPOINT_FILE="$CHECKPOINT_DIR/checkpoint.json"

# Function to save checkpoint
save_checkpoint() {
    local step=$1
    local data=$2
    
    echo "💾 Saving checkpoint at step: $step"
    
    # Create checkpoint JSON
    cat > "$CHECKPOINT_FILE" << EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "step": "$step",
  "data": $data,
  "environment": {
    "task_name": "$TASK_NAME",
    "swarm_id": "$SWARM_ID",
    "pwd": "$(pwd)",
    "git_branch": "$(git branch --show-current 2>/dev/null || echo 'none')"
  }
}
EOF
    
    # Also save workspace state
    if [ -d "/workspace" ]; then
        tar -czf "$CHECKPOINT_DIR/workspace-$step.tar.gz" -C /workspace .
    fi
    
    echo "✅ Checkpoint saved"
}

# Function to load checkpoint
load_checkpoint() {
    if [ ! -f "$CHECKPOINT_FILE" ]; then
        echo "❌ No checkpoint found"
        return 1
    fi
    
    echo "📂 Loading checkpoint..."
    cat "$CHECKPOINT_FILE"
    
    # Extract step
    local step=$(jq -r '.step' "$CHECKPOINT_FILE")
    
    # Restore workspace if exists
    if [ -f "$CHECKPOINT_DIR/workspace-$step.tar.gz" ]; then
        echo "📦 Restoring workspace state..."
        cd /workspace
        tar -xzf "$CHECKPOINT_DIR/workspace-$step.tar.gz"
    fi
    
    echo "✅ Checkpoint loaded from step: $step"
    return 0
}

# Function to clean checkpoints
clean_checkpoints() {
    echo "🧹 Cleaning old checkpoints..."
    find "$CHECKPOINT_DIR" -name "workspace-*.tar.gz" -mtime +7 -delete
    echo "✅ Cleanup complete"
}

# Main execution
case "$1" in
    save)
        save_checkpoint "$2" "$3"
        ;;
    load)
        load_checkpoint
        ;;
    clean)
        clean_checkpoints
        ;;
    *)
        echo "Usage: $0 {save|load|clean} [step] [data]"
        exit 1
        ;;
esac