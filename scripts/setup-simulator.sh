#!/usr/bin/env bash
# setup-simulator.sh – Build the TTK4145 SimElevatorServer from source.
#
# Works on macOS (arm64 / x86_64) and Linux x86_64.
# Prerequisites that will be installed automatically if missing:
#   macOS : Homebrew (https://brew.sh) → dmd
#   Linux : apt → dmd (via d-apt repository)
#
# The compiled binary is placed at:
#   <repo-root>/simulator/SimElevatorServer

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SIM_DIR="$REPO_ROOT/simulator"
SIM_SRC_DIR="$SIM_DIR/Simulator-v2"
SIM_BIN="$SIM_DIR/SimElevatorServer"

echo "==> Setting up TTK4145 Simulator-v2"

# ---- 1. Install dmd if not present ----
if ! command -v dmd &>/dev/null; then
    echo "==> dmd not found – installing..."
    if [[ "$(uname -s)" == "Darwin" ]]; then
        if ! command -v brew &>/dev/null; then
            echo "ERROR: Homebrew is required on macOS."
            echo "Install it from https://brew.sh and re-run this script."
            exit 1
        fi
        brew install dmd
    elif [[ "$(uname -s)" == "Linux" ]]; then
        # Use the official d-apt repository
        sudo apt-get install -y curl gpg
        curl -fsSL https://netcologne.dl.sourceforge.net/project/d-apt/files/d-apt.list \
            -o /tmp/d-apt.list
        sudo cp /tmp/d-apt.list /etc/apt/sources.list.d/d-apt.list
        sudo apt-get update --allow-insecure-repositories
        sudo apt-get install -y --allow-unauthenticated dub dmd-bin
    else
        echo "ERROR: Unsupported OS '$(uname -s)'. Please install dmd manually from https://dlang.org/download.html"
        exit 1
    fi
fi

echo "==> dmd $(dmd --version | head -1)"

# ---- 2. Clone or update the simulator source ----
mkdir -p "$SIM_DIR"
if [[ -d "$SIM_SRC_DIR/.git" ]]; then
    echo "==> Updating Simulator-v2 source..."
    git -C "$SIM_SRC_DIR" pull --ff-only
else
    echo "==> Cloning Simulator-v2 source..."
    git clone https://github.com/TTK4145/Simulator-v2.git "$SIM_SRC_DIR"
fi

# ---- 3. Compile ----
echo "==> Compiling SimElevatorServer..."
dmd -w -g \
    "$SIM_SRC_DIR/src/sim_server.d" \
    "$SIM_SRC_DIR/src/timer_event.d" \
    -of"$SIM_BIN"

echo ""
echo "✅  SimElevatorServer built at: $SIM_BIN"
echo ""
echo "Next steps:"
echo "  1. Run the simulator (in its own terminal window):"
echo "       ./simulator/SimElevatorServer --port 15657 --numfloors 4"
echo ""
echo "  2. Build and run the elevator controller:"
echo "       make build"
echo "       ./heis -id elevator0"
echo ""
echo "  Or just use: make run"
