#!/bin/bash
#
# ICM20948 Magnetometer Test - Direct on Raspberry Pi
#
# This script runs directly on the Raspberry Pi
# Compiles and runs the test program using the local Go installation
#
# Usage:
#   1. Copy this script to Pi: scp test-direct-on-pi.sh root@<pi-ip>:/root/
#   2. Run on Pi: sudo /root/test-direct-on-pi.sh
#

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}ICM20948 Magnetometer Test${NC}"
echo -e "${GREEN}Direct Test on Raspberry Pi${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: Please run as root (sudo)${NC}"
    exit 1
fi

# Source bashrc for Go environment
if [ -f /root/.bashrc ]; then
    source /root/.bashrc
fi

# Try to find Go
GO_BIN=""
if [ -x "/root/go/bin/go" ]; then
    GO_BIN="/root/go/bin/go"
    echo -e "${GREEN}✓ Found Go at: ${GO_BIN}${NC}"
elif command -v go &> /dev/null; then
    GO_BIN=$(command -v go)
    echo -e "${GREEN}✓ Found Go at: ${GO_BIN}${NC}"
else
    echo -e "${RED}Error: Go not found!${NC}"
    echo -e "${YELLOW}Tried:${NC}"
    echo -e "  - /root/go/bin/go"
    echo -e "  - go in PATH"
    exit 1
fi

# Show Go version
echo -e "${YELLOW}Go version:${NC}"
${GO_BIN} version

# Create temp directory
WORKDIR=$(mktemp -d)
cd "$WORKDIR"
echo -e "${YELLOW}Working directory: ${WORKDIR}${NC}"

# Fetch latest commit hash from GitHub
echo -e "${YELLOW}Fetching latest commit hash from GitHub...${NC}"
LATEST_HASH=$(curl -s https://api.github.com/repos/lordvampire/goflying/commits/stratux_master | jq -r '.sha' 2>/dev/null || echo "")

if [ -z "$LATEST_HASH" ] || [ "$LATEST_HASH" = "null" ]; then
    echo -e "${RED}Warning: Could not fetch latest commit hash${NC}"
    echo -e "${YELLOW}Enter commit hash manually (or press Enter for local test):${NC}"
    read -r MANUAL_HASH
    if [ -n "$MANUAL_HASH" ]; then
        LATEST_HASH="$MANUAL_HASH"
    else
        echo -e "${RED}No hash provided, exiting${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}Using commit hash: ${LATEST_HASH}${NC}"

# Create go.mod
echo -e "${YELLOW}Creating test module...${NC}"
cat > go.mod <<EOF
module test-icm20948

go 1.16

require github.com/b3nn0/goflying v0.0.0

replace github.com/b3nn0/goflying => github.com/lordvampire/goflying ${LATEST_HASH}
EOF

# Create test program
echo -e "${YELLOW}Creating test program...${NC}"
cat > main.go <<'GOEOF'
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/b3nn0/goflying/icm20948"
)

func main() {
	log.Println("===========================================")
	log.Println("ICM20948 Magnetometer Standalone Test")
	log.Println("===========================================")

	i2cbus := 1
	gyroRange := 2000
	accelRange := 16
	updateFreq := 50
	enableMag := true
	enableDMP := false

	log.Printf("Initializing ICM20948 on I2C bus %d...\n", i2cbus)
	log.Printf("Settings: Gyro=±%d°/s, Accel=±%dg, Freq=%dHz, Mag=%v\n",
		gyroRange, accelRange, updateFreq, enableMag)

	mpu, err := icm20948.NewICM20948(i2cbus, gyroRange, accelRange, updateFreq, enableMag, enableDMP)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize ICM20948: %v\n", err)
	}

	log.Println("ICM20948 initialized successfully!")
	log.Println("===========================================")
	log.Println("Reading magnetometer data...")
	log.Println("Press Ctrl+C to exit")
	log.Println("===========================================")

	// Setup signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	readCount := 0
	zeroCount := 0

	for {
		select {
		case <-sigChan:
			log.Println("===========================================")
			log.Printf("Test stopped. Total reads: %d, Zero readings: %d\n", readCount, zeroCount)
			if zeroCount == readCount && readCount > 0 {
				log.Println("❌ MAGNETOMETER NOT WORKING - all readings were zero!")
			} else if zeroCount > 0 {
				log.Printf("⚠️  PARTIAL FAILURE - %d/%d readings were zero\n", zeroCount, readCount)
			} else if readCount > 0 {
				log.Println("✅ MAGNETOMETER WORKING - all readings had values!")
			}
			return

		case <-ticker.C:
			readCount++

			err := mpu.ReadSensor()
			if err != nil {
				log.Printf("[%04d] ERROR reading sensor: %v\n", readCount, err)
				continue
			}

			m1, m2, m3 := mpu.Magnetometer()
			g1, g2, g3 := mpu.Gyro()
			a1, a2, a3 := mpu.Accel()

			log.Printf("[%04d] Gyro: X=%7.2f Y=%7.2f Z=%7.2f °/s | Accel: X=%6.3f Y=%6.3f Z=%6.3f g\n",
				readCount, g1, g2, g3, a1, a2, a3)
			log.Printf("[%04d] **MAG: X=%7.2f Y=%7.2f Z=%7.2f µT**\n",
				readCount, m1, m2, m3)

			if m1 == 0 && m2 == 0 && m3 == 0 {
				zeroCount++
				log.Printf("[%04d] ⚠️  WARNING: Magnetometer returns all zeros!\n", readCount)
			} else {
				log.Printf("[%04d] ✅ Magnetometer data received!\n", readCount)
			}
		}
	}
}
GOEOF

# Download dependencies
echo -e "${YELLOW}Downloading dependencies (this may take a moment)...${NC}"
${GO_BIN} mod download

echo -e "${YELLOW}Tidying modules...${NC}"
${GO_BIN} mod tidy

# Build
echo -e "${YELLOW}Building test program...${NC}"
${GO_BIN} build -o test-magnetometer main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed!${NC}"
    cd /
    rm -rf "$WORKDIR"
    exit 1
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ Build successful!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}Starting test in 2 seconds...${NC}"
echo -e "${YELLOW}(Press Ctrl+C to stop)${NC}"
echo ""
sleep 2

# Run the test
./test-magnetometer

# Cleanup
echo ""
echo -e "${YELLOW}Cleaning up...${NC}"
cd /
rm -rf "$WORKDIR"

echo -e "${GREEN}Done!${NC}"
