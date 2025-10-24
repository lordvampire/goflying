#!/bin/bash
#
# Restore Working Magnetometer Configuration
# Stellt den funktionierenden Stand vom Backup wieder her
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Restore Working Magnetometer Setup${NC}"
echo -e "${GREEN}========================================${NC}"

# Find latest backup
BACKUP_DIR=$(ls -td /root/backup-* | head -1)

if [ -z "$BACKUP_DIR" ]; then
    echo -e "${RED}Error: No backup found${NC}"
    exit 1
fi

echo -e "${YELLOW}Using backup: $BACKUP_DIR${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: Please run as root (sudo)${NC}"
    exit 1
fi

# Stop Stratux
echo -e "${YELLOW}Stopping Stratux...${NC}"
systemctl stop stratux 2>/dev/null || true

# Restore sensors.go
echo -e "${YELLOW}Restoring sensors.go with MagHeading fix...${NC}"
if [ -f "$BACKUP_DIR/sensors.go.working" ]; then
    cp "$BACKUP_DIR/sensors.go.working" /root/stratux/main/sensors.go
    echo -e "${GREEN}✓ sensors.go restored${NC}"
else
    echo -e "${RED}✗ sensors.go.working not found in backup${NC}"
fi

# Restore go.mod
echo -e "${YELLOW}Restoring go.mod...${NC}"
if [ -f "$BACKUP_DIR/go.mod.working" ]; then
    cp "$BACKUP_DIR/go.mod.working" /root/stratux/go.mod
    echo -e "${GREEN}✓ go.mod restored${NC}"
else
    echo -e "${RED}✗ go.mod.working not found in backup${NC}"
fi

# Update goflying with the critical fix
cd /root/stratux

echo -e "${YELLOW}Applying critical USER_CTRL fix to goflying...${NC}"

# Clean module cache
go clean -modcache

# Remove go.sum
rm -f go.sum

# Download and tidy
go mod download
go mod tidy

# Find and patch the USER_CTRL register
GOFLYING_PATH=$(find /root/go_path/pkg/mod/github.com -name "*goflying*" -type d | grep -E "(lordvampire|b3nn0)" | head -1)

if [ -n "$GOFLYING_PATH" ]; then
    DMP_FILE="$GOFLYING_PATH/icm20948/dmp_constants.go"

    # Make it writable
    chmod -R u+w "$GOFLYING_PATH" 2>/dev/null || true

    if [ -f "$DMP_FILE" ]; then
        # Check current value
        if grep -q "ICMREG_USER_CTRL.*0x6A" "$DMP_FILE"; then
            echo -e "${YELLOW}Fixing USER_CTRL from 0x6A to 0x03...${NC}"
            sed -i 's/ICMREG_USER_CTRL.*=.*0x6A/ICMREG_USER_CTRL          = 0x03 \/\/ Bank 0: CRITICAL FIX! Was 0x6A (MPU-9250), should be 0x03 (ICM-20948)/' "$DMP_FILE"
            echo -e "${GREEN}✓ USER_CTRL patched to 0x03${NC}"
        elif grep -q "ICMREG_USER_CTRL.*0x03" "$DMP_FILE"; then
            echo -e "${GREEN}✓ USER_CTRL already set to 0x03${NC}"
        fi
    else
        echo -e "${RED}✗ dmp_constants.go not found${NC}"
    fi
else
    echo -e "${RED}✗ goflying module not found${NC}"
fi

# Rebuild Stratux
echo -e "${YELLOW}Rebuilding Stratux...${NC}"
make clean
make
make install

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Restore completed!${NC}"
echo -e "${GREEN}========================================${NC}"

read -p "Start Stratux now? [y/n] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    systemctl start stratux
    echo -e "${GREEN}Stratux started!${NC}"
else
    echo -e "${YELLOW}Start manually with: systemctl start stratux${NC}"
fi
