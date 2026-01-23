#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting Production Simulation Check...${NC}"

# 1. Cleanup
echo "Cleaning up previous processes..."
pgrep -f vds-backend | xargs -r kill
pgrep -f "caddy run --config Caddyfile.vds" | xargs -r kill || pgrep -f "caddy" | xargs -r kill

# 2. Build Frontend
echo "Building Frontend..."
cd frontend
npm install > /dev/null 2>&1
npm run build > /dev/null 2>&1
cd ..

# 3. Build Backend
echo "Building Backend..."
COMMIT_SHA=$(git rev-parse HEAD 2>/dev/null || echo "local-dev")
/usr/local/go/bin/go build -C backend -ldflags="-X main.Commit=${COMMIT_SHA}" -o vds-backend ./cmd/server

# 4. Setup VDS Simulation Folder
echo "Syncing artifacts to vds_sim/..."
mkdir -p vds_sim
rm -rf vds_sim/frontend
cp -r frontend/dist vds_sim/frontend
cp backend/vds-backend vds_sim/
echo "${COMMIT_SHA}" > vds_sim/frontend/version.txt

# Create dedicated Caddyfile for this script if it doesn't exist
cat <<EOF > vds_sim/Caddyfile.vds
{
  admin off
}
:8081 {
  handle /api/* {
    reverse_proxy localhost:8080
  }
  handle /healthz {
    reverse_proxy localhost:8080
  }
  handle /version {
    reverse_proxy localhost:8080
  }
  handle /frontend-version {
    root * /workspace/vds_sim/frontend
    rewrite * /version.txt
    file_server
  }
  handle {
    root * /workspace/vds_sim/frontend
    try_files {path} /index.html
    file_server
  }
}
EOF

# 5. Launch
echo "Launching Simulation..."
cd vds_sim
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=training ./vds-backend > backend.log 2>&1 &
caddy run --config Caddyfile.vds > caddy.log 2>&1 &
cd ..

# 6. Verify
echo "Waiting for services to warm up..."
sleep 4

SUCCESS=true

echo -n "Checking Health... "
if curl -s localhost:8081/healthz | grep -q "ok"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    SUCCESS=false
fi

echo -n "Checking Backend Version... "
if curl -s localhost:8081/version | grep -q "${COMMIT_SHA}"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL (Expected ${COMMIT_SHA})${NC}"
    SUCCESS=false
fi

echo -n "Checking Frontend Version... "
if curl -s localhost:8081/frontend-version | grep -q "${COMMIT_SHA}"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL (Expected ${COMMIT_SHA})${NC}"
    SUCCESS=false
fi

if [ "$SUCCESS" = true ]; then
    echo -e "${GREEN}✅ ALL CHECKS PASSED. Ready to commit/push!${NC}"
    exit 0
else
    echo -e "${RED}❌ SOME CHECKS FAILED. Check vds_sim/backend.log and vds_sim/caddy.log${NC}"
    exit 1
fi
