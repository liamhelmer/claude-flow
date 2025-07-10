#!/bin/bash
# Development setup script for Claude Flow Swarm Operator

set -e

echo "🚀 Setting up Claude Flow Swarm Operator development environment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running inside container
if [ -f /.dockerenv ]; then
    echo -e "${GREEN}✓ Running inside development container${NC}"
    
    # Initialize kubebuilder project if not already done
    if [ ! -f "PROJECT" ]; then
        echo -e "${YELLOW}Initializing kubebuilder project...${NC}"
        kubebuilder init --domain claudeflow.io --repo github.com/claude-flow/swarm-operator
        echo -e "${GREEN}✓ Kubebuilder project initialized${NC}"
    else
        echo -e "${GREEN}✓ Kubebuilder project already initialized${NC}"
    fi
    
    # Check kubebuilder version
    echo -e "\n${YELLOW}Kubebuilder version:${NC}"
    kubebuilder version
    
else
    echo -e "${YELLOW}Not running inside development container${NC}"
    echo "Starting development environment..."
    
    # Build and start dev container
    make dev-build
    make dev-up
    
    echo -e "\n${GREEN}✓ Development environment ready!${NC}"
    echo -e "Run '${YELLOW}make dev-shell${NC}' to enter the container"
    echo -e "Then run '${YELLOW}./hack/setup-dev.sh${NC}' again inside the container"
fi

echo -e "\n${GREEN}✓ Setup complete!${NC}"
echo -e "\nNext steps:"
echo -e "1. Wait for CRD designs from Architecture Designer"
echo -e "2. Create APIs: ${YELLOW}kubebuilder create api --group swarm --version v1alpha1 --kind [ResourceName]${NC}"
echo -e "3. Implement controllers in ${YELLOW}controllers/${NC}"
echo -e "4. Run tests: ${YELLOW}make test${NC}"
echo -e "5. Run locally: ${YELLOW}make run${NC}"