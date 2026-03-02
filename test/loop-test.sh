#!/usr/bin/env bash
# 死循环，每 30 秒打印当前时间（用于 agent-runtime 调试）
set -e
while true; do
  echo "$(date -Iseconds 2>/dev/null || date)"
  sleep 30
done
