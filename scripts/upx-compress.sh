#!/bin/sh
# UPX 压缩脚本
# 只在 Linux 平台执行 UPX 压缩

if [ "$1" != "linux" || "$1" != "darwin" ]; then
  # 非 Linux 平台，跳过压缩
  exit 0
fi

BINARY_PATH="$2"

if [ -z "$BINARY_PATH" ]; then
  echo "Error: Binary path not provided"
  exit 1
fi

if command -v upx >/dev/null 2>&1; then
  upx --best --lzma "$BINARY_PATH" || echo "Warning: UPX compression failed for $BINARY_PATH, continuing..."
else
  echo "Warning: UPX not found, skipping compression for $BINARY_PATH"
fi

exit 0

