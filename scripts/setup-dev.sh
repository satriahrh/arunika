#!/bin/bash

# Arunika Development Setup Script
# This script sets up the development environment for the Arunika project

set -e

echo "🚀 Setting up Arunika development environment..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    echo "   Visit: https://golang.org/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if ! printf '%s\n%s' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo "❌ Go version $GO_VERSION is too old. Please upgrade to Go 1.21 or later."
    exit 1
fi

echo "✅ Go version $GO_VERSION is compatible"

# Check if GCC is installed
if ! command -v gcc &> /dev/null; then
    echo "❌ GCC is not installed. Please install GCC for C development."
    case "$(uname -s)" in
        Darwin*)
            echo "   macOS: xcode-select --install"
            ;;
        Linux*)
            echo "   Ubuntu/Debian: apt-get install build-essential"
            echo "   CentOS/RHEL: yum groupinstall 'Development Tools'"
            ;;
        *)
            echo "   Please install GCC for your platform"
            ;;
    esac
    exit 1
fi

echo "✅ GCC is available"

# Check if Make is installed
if ! command -v make &> /dev/null; then
    echo "❌ Make is not installed. Please install Make."
    exit 1
fi

echo "✅ Make is available"

# Setup Go server dependencies
echo "📦 Installing Go dependencies..."
cd server
go mod tidy
if [ $? -eq 0 ]; then
    echo "✅ Go dependencies installed successfully"
else
    echo "❌ Failed to install Go dependencies"
    exit 1
fi
cd ..

# Build firmware to test C compilation
echo "🔨 Testing C compilation..."
cd doll-m2
make clean
make all
if [ $? -eq 0 ]; then
    echo "✅ C compilation successful"
else
    echo "❌ C compilation failed"
    exit 1
fi
cd ..

# Create development environment file
echo "📝 Creating development environment file..."
cat > .env.development << EOF
# Arunika Development Environment
PORT=8080
JWT_SECRET=development-secret-key-change-in-production
LOG_LEVEL=debug
ENVIRONMENT=development

# AI Service Configuration (to be configured)
# OPENAI_API_KEY=your-openai-api-key
# GOOGLE_CLOUD_PROJECT=your-gcp-project
# AZURE_SPEECH_KEY=your-azure-speech-key

# Database Configuration (future)
# DB_HOST=localhost
# DB_PORT=3306
# DB_NAME=arunika
# DB_USER=arunika
# DB_PASSWORD=arunika
EOF

echo "✅ Development environment file created (.env.development)"

# Create VS Code workspace configuration
echo "📝 Creating VS Code workspace configuration..."
mkdir -p .vscode
cat > .vscode/settings.json << EOF
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.vetOnSave": "package",
    "go.lintOnSave": "package",
    "C_Cpp.default.cStandard": "c99",
    "C_Cpp.default.includePath": [
        "\${workspaceFolder}/doll-m2/include"
    ],
    "files.associations": {
        "*.h": "c",
        "*.c": "c"
    },
    "files.exclude": {
        "**/build": true,
        "**/bin": true,
        "**/*.o": true
    }
}
EOF

cat > .vscode/tasks.json << EOF
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Run Go Server",
            "type": "shell",
            "command": "go",
            "args": ["run", "cmd/main.go"],
            "options": {
                "cwd": "\${workspaceFolder}/server"
            },
            "group": {
                "kind": "build",
                "isDefault": true
            },
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": []
        },
        {
            "label": "Build Firmware",
            "type": "shell",
            "command": "make",
            "args": ["all"],
            "options": {
                "cwd": "\${workspaceFolder}/doll-m2"
            },
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": ["$gcc"]
        },
        {
            "label": "Test All",
            "type": "shell",
            "command": "./scripts/test-all.sh",
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        }
    ]
}
EOF

echo "✅ VS Code workspace configured"

# Create development scripts
echo "📝 Creating development scripts..."

# Test script
cat > scripts/test-all.sh << 'EOF'
#!/bin/bash

echo "🧪 Running all tests..."

# Test Go server
echo "Testing Go server..."
cd server
go test ./...
if [ $? -ne 0 ]; then
    echo "❌ Go tests failed"
    exit 1
fi
echo "✅ Go tests passed"
cd ..

# Test C firmware
echo "Testing C firmware..."
cd doll-m2
make test
if [ $? -ne 0 ]; then
    echo "❌ C tests failed"
    exit 1
fi
echo "✅ C tests passed"
cd ..

echo "🎉 All tests passed!"
EOF

chmod +x scripts/test-all.sh

# Development server script
cat > scripts/dev-server.sh << 'EOF'
#!/bin/bash

echo "🚀 Starting Arunika development server..."

# Load environment variables
if [ -f .env.development ]; then
    export $(cat .env.development | xargs)
fi

cd server
go run cmd/main.go
EOF

chmod +x scripts/dev-server.sh

# Build script
cat > scripts/build-all.sh << 'EOF'
#!/bin/bash

echo "🔨 Building all components..."

# Build Go server
echo "Building Go server..."
cd server
go build -o ../bin/arunika-server cmd/main.go
if [ $? -ne 0 ]; then
    echo "❌ Server build failed"
    exit 1
fi
echo "✅ Server built successfully"
cd ..

# Build C firmware
echo "Building C firmware..."
cd doll-m2
make clean && make all
if [ $? -ne 0 ]; then
    echo "❌ Firmware build failed"
    exit 1
fi
echo "✅ Firmware built successfully"
cd ..

echo "🎉 All components built successfully!"
EOF

chmod +x scripts/build-all.sh

echo "✅ Development scripts created"

# Create bin directory
mkdir -p bin

echo ""
echo "🎉 Arunika development environment setup complete!"
echo ""
echo "Next steps:"
echo "1. Start the development server: ./scripts/dev-server.sh"
echo "2. Run tests: ./scripts/test-all.sh"
echo "3. Build all components: ./scripts/build-all.sh"
echo ""
echo "Development files created:"
echo "- .env.development (environment variables)"
echo "- .vscode/ (VS Code configuration)"
echo "- scripts/ (development scripts)"
echo "- bin/ (build output directory)"
echo ""
echo "Happy coding! 🚀"
