#!/bin/bash

# Install abigen for Go binding generation

echo "🔧 Installing abigen (go-ethereum tool)..."
echo "=" 
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed!"
    echo ""
    echo "Please install Go first:"
    echo "  macOS: brew install go"
    echo "  Linux: sudo apt install golang"
    echo "  Or download from: https://go.dev/dl/"
    exit 1
fi

echo "✅ Go is installed: $(go version)"
echo ""

# Install abigen
echo "📦 Installing abigen..."
go install github.com/ethereum/go-ethereum/cmd/abigen@latest

# Check if installation was successful
if command -v abigen &> /dev/null; then
    echo "✅ abigen installed successfully!"
    echo "   Version: $(abigen --version)"
else
    echo "⚠️  abigen installed but not in PATH"
    echo ""
    echo "Add this to your shell profile (.bashrc, .zshrc, etc.):"
    echo '  export PATH=$PATH:$(go env GOPATH)/bin'
    echo ""
    echo "Then reload your shell:"
    echo "  source ~/.zshrc  # or ~/.bashrc"
fi

echo ""
echo "✨ Installation complete!"
echo ""
echo "You can now generate Go bindings with:"
echo "  npm run generate:go"