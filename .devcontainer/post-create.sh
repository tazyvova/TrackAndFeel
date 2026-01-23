#!/bin/bash
set -e

# Install act for local GitHub Actions
curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Make sure we are in the workspace root
cd /workspace

# Backend setup (optional, can be done manually)
# cd backend && go mod download

# Frontend setup (optional, can be done manually)
# cd frontend && npm install
