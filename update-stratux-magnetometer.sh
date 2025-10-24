#!/bin/bash
#
# Stratux Magnetometer Update Script
# Automatically fetches latest lordvampire/goflying-b3nn0 code and rebuilds Stratux
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
echo -e "${YELLOW}Fetching latest commit from lordvampire/goflying (stratux_master branch)...${NC}"
LATEST_HASH=$(curl -s https://api.github.com/repos/lordvampire/goflying/commits/stratux_master | jq -r '.sha')

if [ -z "$LATEST_HASH" ] || [ "$LATEST_HASH" = "null" ]; then
    echo -e "${YELLOW}Warning: Could not fetch latest commit, using hardcoded hash...${NC}"
    LATEST_HASH="e7543f71af1e29d95d062be02310559aa6488156"
fi

# Get commit timestamp for pseudo-version
COMMIT_TIME=$(curl -s https://api.github.com/repos/lordvampire/goflying/commits/$LATEST_HASH | jq -r '.commit.committer.date' | sed 's/[-:]//g' | sed 's/T/ /' | awk '{print substr($1,1,8) substr($2,1,6)}')
if [ -z "$COMMIT_TIME" ]; then
    COMMIT_TIME="20251024121001"
fi

# Create pseudo-version (v0.0.0-yyyymmddhhmmss-abcdefabcdef)
PSEUDO_VERSION="v0.0.0-${COMMIT_TIME}-${LATEST_HASH:0:12}"

echo -e "${GREEN}Latest commit: ${LATEST_HASH}${NC}"
echo -e "${GREEN}Pseudo-version: ${PSEUDO_VERSION}${NC}"

# Change to stratux directory
cd /root/stratux

# Show current replace directive
echo -e "${YELLOW}Current go.mod replace:${NC}"
grep "goflying" go.mod || echo "No replace directive found"

# Clean Go module cache
echo -e "${YELLOW}Cleaning Go module cache...${NC}"
go clean -modcache

# Update replace directive to use lordvampire/goflying
echo -e "${YELLOW}Updating replace directive with ${PSEUDO_VERSION}...${NC}"
/root/go/bin/go mod edit -dropreplace github.com/b3nn0/goflying
/root/go/bin/go mod edit -replace github.com/b3nn0/goflying=github.com/lordvampire/goflying@${PSEUDO_VERSION}

# Remove go.sum to force redownload
echo -e "${YELLOW}Removing go.sum...${NC}"
rm -f go.sum

# Download modules
echo -e "${YELLOW}Downloading Go modules...${NC}"
go mod download

# Tidy modules
echo -e "${YELLOW}Tidying Go modules...${NC}"
go mod tidy

# Verify the critical USER_CTRL fix is present
echo -e "${YELLOW}Verifying critical USER_CTRL fix (0x03 instead of 0x6A)...${NC}"
GOFLYING_PATH=$(find /root/go_path/pkg/mod/github.com/lordvampire -name "goflying@*" -type d | head -1)
if [ -n "$GOFLYING_PATH" ]; then
    if grep -q "ICMREG_USER_CTRL.*0x03" "$GOFLYING_PATH/icm20948/dmp_constants.go" 2>/dev/null; then
        echo -e "${GREEN}✓ USER_CTRL fix (0x03) found - magnetometer will work!${NC}"
    elif grep -q "ICMREG_USER_CTRL.*0x6A" "$GOFLYING_PATH/icm20948/dmp_constants.go" 2>/dev/null; then
        echo -e "${RED}✗ CRITICAL: USER_CTRL is still 0x6A - magnetometer will NOT work!${NC}"
        echo -e "${RED}   Please push the fix to lordvampire/goflying (stratux_master branch) first!${NC}"
        exit 1
    else
        echo -e "${YELLOW}⚠ Warning: Could not verify USER_CTRL value${NC}"
    fi

    # Verify magnetometer init code
    if grep -q "Initializing AK09916" "$GOFLYING_PATH/icm20948/icm20948.go" 2>/dev/null; then
        echo -e "${GREEN}✓ Magnetometer init code found${NC}"
    else
        echo -e "${YELLOW}⚠ Warning: Magnetometer init code not found${NC}"
    fi
else
    echo -e "${RED}✗ Warning: Could not find downloaded lordvampire/goflying module${NC}"
fi

# Show updated replace directive
echo -e "${YELLOW}Updated go.mod replace:${NC}"
grep "goflying" go.mod

# Apply sensors.go patch for MagHeading
echo -e "${YELLOW}Applying MagHeading patch to sensors.go...${NC}"
if grep -q "mySituation.AHRSMagHeading = ahrs.Invalid" /root/stratux/main/sensors.go; then
    echo -e "${YELLOW}Found hardcoded Invalid - patching...${NC}"

    # Create backup
    cp /root/stratux/main/sensors.go /root/stratux/main/sensors.go.backup

    # Apply patch using sed
    sed -i '/\/\/TODO westphae: until magnetometer calibration is performed, no mag heading/,/mySituation.AHRSMagHeading = ahrs.Invalid/c\
\t\t\t\t// Get magnetometer heading from AHRS\
\t\t\t\tmagHeading := s.MagHeading()\
\t\t\t\tmySituation.AHRSMagHeading = magHeading\
\t\t\t\tif !isAHRSInvalidValue(magHeading) {\
\t\t\t\t\tmySituation.AHRSMagHeading /= ahrs.Deg\
\t\t\t\t}' /root/stratux/main/sensors.go

    echo -e "${GREEN}✓ MagHeading patch applied${NC}"
else
    echo -e "${GREEN}✓ MagHeading already patched or different version${NC}"
fi

# Verify patch was applied
if grep -q "s.MagHeading()" /root/stratux/main/sensors.go; then
    echo -e "${GREEN}✓ Verified: MagHeading() call found in sensors.go${NC}"
else
    echo -e "${RED}✗ Warning: MagHeading() call not found - patch may have failed${NC}"
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
echo -e "${GREEN}Applied changes:${NC}"
echo -e "${GREEN}  1. Updated lordvampire/goflying with USER_CTRL fix (0x03)${NC}"
echo -e "${GREEN}  2. Patched sensors.go for MagHeading integration${NC}"
echo -e "${GREEN}========================================${NC}"

# Ask to restart stratux
read -t 10 -n 10000 discard 2>/dev/null || true
read -p "Restart Stratux service now? [y/n] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Restarting Stratux...${NC}"
    systemctl restart stratux
    sleep 5
    echo -e "${GREEN}Stratux restarted!${NC}"
    echo
    echo -e "${YELLOW}Testing magnetometer...${NC}"
    sleep 10
    if curl -s http://localhost/getSituation | grep -q "AHRSMagHeading"; then
        MAG_VALUE=$(curl -s http://localhost/getSituation | python3 -c "import sys, json; print(json.load(sys.stdin)['AHRSMagHeading'])")
        if [ "$MAG_VALUE" != "3276.7" ]; then
            echo -e "${GREEN}✓ Magnetometer working! MagHeading = $MAG_VALUE${NC}"
        else
            echo -e "${RED}✗ MagHeading still shows Invalid ($MAG_VALUE)${NC}"
        fi
    fi
    echo
    echo -e "${YELLOW}Monitor logs with: journalctl -u stratux -f${NC}"
else
    echo -e "${YELLOW}Restart manually with: systemctl restart stratux${NC}"
fi

echo -e "${GREEN}Done!${NC}"
