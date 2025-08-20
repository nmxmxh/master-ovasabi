#!/bin/sh
set -e


GODOT_VERSION=${1:-4.4.1}
OUT_DIR=${2:-godot/bin}
OUT_FILE="$OUT_DIR/Godot_v${GODOT_VERSION}-stable_linux.x86_64"

mkdir -p "$OUT_DIR"
wget -O /tmp/godot.zip "https://github.com/godotengine/godot/releases/download/${GODOT_VERSION}-stable/Godot_v${GODOT_VERSION}-stable_linux.x86_64.zip"
unzip -j /tmp/godot.zip -d "$OUT_DIR"
mv "$OUT_DIR/Godot_v${GODOT_VERSION}-stable_linux.x86_64" "$OUT_FILE"
rm /tmp/godot.zip

echo "Godot binary downloaded to $OUT_FILE"
