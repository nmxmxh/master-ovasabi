#!/bin/bash

# setup_ai_environment.sh
# Creates a virtual environment and installs requirements for the OVASABI AI system

set -e  # Exit on any error

echo "🚀 Setting up OVASABI AI Python Environment"
echo "=========================================="

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="/Users/okhai/Desktop/OVASABI STUDIOS/master-ovasabi"
AI_DIR="$PROJECT_ROOT/internal/ai/python"

cd "$AI_DIR"

# Check if Python 3.11 is available (required for TensorFlow compatibility)
if ! command -v python3.11 &> /dev/null; then
    echo "❌ Python 3.11 is required but not found."
    echo "TensorFlow and other ML libraries need Python 3.8-3.12"
    if command -v pyenv &> /dev/null; then
        echo "Using pyenv to find Python 3.11..."
        if pyenv versions | grep -q "3.11"; then
            PYTHON_CMD="pyenv exec python"
            PYTHON_VERSION=$(pyenv exec python --version | cut -d' ' -f2)
            echo "✅ Found Python $PYTHON_VERSION via pyenv"
        else
            echo "Please install Python 3.11 via pyenv: pyenv install 3.11.9"
            exit 1
        fi
    else
        echo "Please install Python 3.11 and try again."
        exit 1
    fi
else
    PYTHON_CMD="python3.11"
    PYTHON_VERSION=$(python3.11 --version | cut -d' ' -f2)
    echo "✅ Found Python $PYTHON_VERSION"
fi

# Create virtual environment
VENV_NAME="ovasabi_ai_env"
if [ -d "$VENV_NAME" ]; then
    echo "📁 Virtual environment '$VENV_NAME' already exists"
    read -p "Do you want to recreate it? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🗑️  Removing existing virtual environment..."
        rm -rf "$VENV_NAME"
    else
        echo "Using existing virtual environment"
    fi
fi

if [ ! -d "$VENV_NAME" ]; then
    echo "🔧 Creating virtual environment '$VENV_NAME' with $PYTHON_CMD..."
    $PYTHON_CMD -m venv "$VENV_NAME"
fi

# Activate virtual environment
echo "⚡ Activating virtual environment..."
source "$VENV_NAME/bin/activate"

# Upgrade pip first
echo "📦 Upgrading pip..."
pip install --upgrade pip

# Install NumPy first with specific version to avoid conflicts
echo "🔢 Installing NumPy <2.0 to avoid compatibility issues..."
pip install "numpy<2.0"

# Install requirements
echo "📚 Installing requirements from requirements.txt..."
if [ -f "requirements.txt" ]; then
    pip install -r requirements.txt
else
    echo "⚠️  requirements.txt not found, installing essential packages..."
    pip install \
        "numpy<2.0" \
        "pandas>=2.0.0" \
        "scikit-learn" \
        "transformers" \
        "torch" \
        "sentence-transformers" \
        "pydantic" \
        "typer" \
        "pytest" \
        "pytest-asyncio"
fi

# Fix any potential compatibility issues
echo "🔧 Fixing potential compatibility issues..."

# Downgrade JAX if it exists and conflicts with NumPy
if pip show jax &> /dev/null; then
    echo "📉 Checking JAX compatibility..."
    pip install "jax[cpu]" --upgrade --force-reinstall || echo "⚠️  JAX install issues - will work in fallback mode"
fi

# Install or fix ml-dtypes compatibility
echo "🔧 Ensuring ml-dtypes compatibility..."
pip install "ml-dtypes>=0.2.0,<0.6.0" || echo "⚠️  ml-dtypes compatibility issues - will work in fallback mode"

# Test the installation
echo "🧪 Testing the installation..."
$VENV_NAME/bin/python -c "
import sys
print(f'Python version: {sys.version}')

try:
    import numpy as np
    print(f'✅ NumPy {np.__version__} - OK')
except Exception as e:
    print(f'❌ NumPy: {e}')

try:
    import transformers
    print(f'✅ Transformers {transformers.__version__} - OK')
except Exception as e:
    print(f'⚠️  Transformers: {e}')

try:
    import sentence_transformers
    print(f'✅ Sentence-Transformers {sentence_transformers.__version__} - OK')
except Exception as e:
    print(f'⚠️  Sentence-Transformers: {e}')

try:
    import torch
    print(f'✅ PyTorch {torch.__version__} - OK')
except Exception as e:
    print(f'⚠️  PyTorch: {e}')

try:
    import jax
    print(f'✅ JAX {jax.__version__} - OK')
except Exception as e:
    print(f'⚠️  JAX: {e}')
"

echo ""
echo "🎉 Environment setup complete!"
echo ""
echo "To activate the environment in the future, run:"
echo "  cd '$AI_DIR'"
echo "  source $VENV_NAME/bin/activate"
echo ""
echo "To test the AI system, run:"
echo "  python test_robust_imports.py"
echo ""
echo "To run in offline mode:"
echo "  export HF_HUB_OFFLINE=true"
echo "  export TRANSFORMERS_OFFLINE=true"
echo "  python test_robust_imports.py"
