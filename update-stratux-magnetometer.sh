#!/bin/bash
#
# Stratux Magnetometer Update Script
# Automatically fetches latest lordvampire/goflying code and rebuilds Stratux
#
# Usage:
#   sudo ./update-stratux-magnetometer.sh
#

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Stratux Magnetometer Update Script${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: Please run as root (sudo)${NC}"
    exit 1
fi

# Source bashrc for Go environment
source /root/.bashrc

# Fetch latest commit hash from GitHub
echo -e "${YELLOW}Fetching latest commit hash from GitHub...${NC}"
LATEST_HASH=$(curl -s https://api.github.com/repos/lordvampire/goflying/commits/stratux_master | jq -r '.sha')

if [ -z "$LATEST_HASH" ] || [ "$LATEST_HASH" = "null" ]; then
    echo -e "${RED}Error: Could not fetch latest commit hash from GitHub${NC}"
    echo -e "${YELLOW}Falling back to manual hash...${NC}"
    read -p "Enter commit hash: " LATEST_HASH
fi

echo -e "${GREEN}Latest commit hash: ${LATEST_HASH}${NC}"

# Change to stratux directory
cd /root/stratux

# Show current replace directive
echo -e "${YELLOW}Current go.mod replace:${NC}"
grep "lordvampire/goflying" go.mod || echo "No replace directive found"

# Clean Go module cache
echo -e "${YELLOW}Cleaning Go module cache...${NC}"
go clean -modcache

# Update replace directive
echo -e "${YELLOW}Updating replace directive with hash ${LATEST_HASH}...${NC}"
/root/go/bin/go mod edit -replace github.com/b3nn0/goflying=github.com/lordvampire/goflying@${LATEST_HASH}

# Remove go.sum to force redownload
echo -e "${YELLOW}Removing go.sum...${NC}"
rm -f go.sum

# Download modules
echo -e "${YELLOW}Downloading Go modules...${NC}"
go mod download

# Tidy modules
echo -e "${YELLOW}Tidying Go modules...${NC}"
go mod tidy

# Verify the debug logging is present
echo -e "${YELLOW}Verifying magnetometer debug code...${NC}"
if grep -q "Initializing AK09916" /root/go_path/pkg/mod/github.com/lordvampire/goflying@*/icm20948/icm20948.go 2>/dev/null; then
    echo -e "${GREEN}✓ Debug logging found in downloaded module${NC}"
else
    echo -e "${RED}✗ Warning: Debug logging not found - build may use old code${NC}"
fi

if grep -q "I2C_MST_ODR_CONFIG" /root/go_path/pkg/mod/github.com/lordvampire/goflying@*/icm20948/dmp_constants.go 2>/dev/null; then
    echo -e "${GREEN}✓ I2C_MST_ODR_CONFIG found (critical fix)${NC}"
else
    echo -e "${RED}✗ Warning: I2C_MST_ODR_CONFIG not found - build may use old code${NC}"
fi

# Show updated replace directive
echo -e "${YELLOW}Updated go.mod replace:${NC}"
grep "lordvampire/goflying" go.mod

# Copy patched sensors.go (fixes ICM-20948 Bank 0 detection)
if [ -f /home/pi/sensors.go ]; then
    echo -e "${YELLOW}Copying patched sensors.go...${NC}"
    cp /home/pi/sensors.go /root/stratux/main/sensors.go
    echo -e "${GREEN}✓ sensors.go patched${NC}"
else
    echo -e "${RED}✗ Warning: /home/pi/sensors.go not found - using original${NC}"
fi

# Clean build
echo -e "${YELLOW}Cleaning previous build...${NC}"
make clean

# Build Stratux
echo -e "${YELLOW}Building Stratux (this may take a few minutes)...${NC}"
make

# Install
echo -e "${YELLOW}Installing Stratux...${NC}"
make install

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"

# Ask to restart stratux
read -t 10 -n 10000 discard 2>/dev/null || true
read -p "Restart Stratux service now? [y/n] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Restarting Stratux...${NC}"
    systemctl restart stratux
    echo -e "${GREEN}Stratux restarted!${NC}"
    echo -e "${YELLOW}Monitor logs with: journalctl -u stratux -f${NC}"
else
    echo -e "${YELLOW}Restart manually with: systemctl restart stratux${NC}"
fi

echo -e "${GREEN}Done!${NC}"
