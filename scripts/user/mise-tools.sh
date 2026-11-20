#!/usr/bin/env bash
# Ferramentas instaladas por usuário via mise (não são pacotes do sistema).
# Inspirado no install/user/mise.sh do Omarchy, enxugado. Idempotente.
set -euo pipefail

mise use -g node@lts
mise use -g python@latest

# CLIs de IA para programação: deixe só os que você usa
mise use -g npm:@anthropic-ai/claude-code
mise use -g npm:@openai/codex
mise use -g npm:@google/gemini-cli

mkdir -p "$HOME/Work"
