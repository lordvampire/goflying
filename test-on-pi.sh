#!/bin/bash
#
# Quick test script for Raspberry Pi
# Fetches latest code and runs magnetometer test
#
# Usage on Raspberry Pi:
#   curl -sSL https://raw.githubusercontent.com/lordvampire/goflying/stratux_master/test-on-pi.sh | sudo bash
#
# Or manually:
#   sudo ./test-on-pi.sh
#

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}ICM20948 Magnetometer Quick Test${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: Please run as root (sudo)${NC}"
    exit 1
fi

# Create temp directory
TMPDIR=$(mktemp -d)
cd "$TMPDIR"

echo -e "${YELLOW}Fetching latest commit hash from GitHub...${NC}"
LATEST_HASH=$(curl -s https://api.github.com/repos/lordvampire/goflying/commits/stratux_master | jq -r '.sha')

if [ -z "$LATEST_HASH" ] || [ "$LATEST_HASH" = "null" ]; then
    echo -e "${RED}Error: Could not fetch latest commit hash${NC}"
    exit 1
fi

echo -e "${GREEN}Latest hash: ${LATEST_HASH}${NC}"

# Create a minimal go.mod
echo -e "${YELLOW}Creating test module...${NC}"
cat > go.mod <<EOF
module test-icm20948

go 1.16

require github.com/b3nn0/goflying v0.0.0

replace github.com/b3nn0/goflying => github.com/lordvampire/goflying ${LATEST_HASH}
EOF

# Create the test program
echo -e "${YELLOW}Creating test program...${NC}"
cat > main.go <<'EOF'
package main

import (
	"log"
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

	mpu, err := icm20948.NewICM20948(i2cbus, gyroRange, accelRange, updateFreq, enableMag, enableDMP)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize ICM20948: %v\n", err)
	}

	log.Println("ICM20948 initialized successfully!")
	log.Println("===========================================")
	log.Println("Reading magnetometer data for 30 seconds...")
	log.Println("===========================================")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)
	readCount := 0

	for {
		select {
		case <-timeout:
			log.Println("===========================================")
			log.Println("Test completed!")
			return
		case <-ticker.C:
			readCount++

			err := mpu.ReadSensor()
			if err != nil {
				log.Printf("[%04d] ERROR: %v\n", readCount, err)
				continue
			}

			m1, m2, m3 := mpu.Magnetometer()
			g1, g2, g3 := mpu.Gyro()
			a1, a2, a3 := mpu.Accel()

			log.Printf("[%04d] Gyro: X=%7.2f Y=%7.2f Z=%7.2f | Accel: X=%6.3f Y=%6.3f Z=%6.3f\n",
				readCount, g1, g2, g3, a1, a2, a3)
			log.Printf("[%04d] **MAG: X=%7.2f Y=%7.2f Z=%7.2f µT**\n",
				readCount, m1, m2, m3)

			if m1 == 0 && m2 == 0 && m3 == 0 {
				log.Printf("[%04d] WARNING: Magnetometer all zeros!\n", readCount)
			}
		}
	}
}
EOF

# Download dependencies
echo -e "${YELLOW}Downloading dependencies...${NC}"
go mod download
go mod tidy

# Build
echo -e "${YELLOW}Building test program...${NC}"
go build -o /root/test-magnetometer main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi

# Cleanup
cd /
rm -rf "$TMPDIR"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Build successful!${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "${YELLOW}Running test program...${NC}"
echo -e "${GREEN}========================================${NC}"

# Run the test
/root/test-magnetometer
