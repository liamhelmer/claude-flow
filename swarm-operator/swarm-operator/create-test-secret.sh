#!/bin/bash

echo "📝 Creating test GitHub secret..."
echo ""
echo "⚠️  WARNING: This creates a placeholder secret."
echo "You'll need to update it with a real GitHub token for the job to succeed."
echo ""

# Create placeholder secret
kubectl create secret generic github-credentials \
    --from-literal=username="liamhelmer" \
    --from-literal=token="ghp_PLACEHOLDER_TOKEN" \
    --from-literal=email="liamhelmer@users.noreply.github.com" \
    --namespace=default \
    --dry-run=client -o yaml | kubectl apply -f -

# Create Claude placeholder secret
kubectl create secret generic claude-credentials \
    --from-literal=api_key="sk-placeholder" \
    --from-literal=model="claude-3-opus-20240229" \
    --namespace=default \
    --dry-run=client -o yaml | kubectl apply -f -

echo "✅ Placeholder secrets created"
echo ""
echo "To update with real GitHub token later:"
echo "kubectl edit secret github-credentials"
echo ""
echo "Or delete and recreate:"
echo "kubectl delete secret github-credentials"
echo "kubectl create secret generic github-credentials \\"
echo "  --from-literal=username=liamhelmer \\"
echo "  --from-literal=token=YOUR_REAL_TOKEN \\"
echo "  --from-literal=email=your-email@example.com"