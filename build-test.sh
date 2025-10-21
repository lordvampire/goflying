#!/bin/bash
#
# Build and deploy ICM20948 magnetometer test program to Raspberry Pi
#
# Usage:
#   ./build-test.sh [raspberry-pi-ip]
#

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}ICM20948 Test Build Script${NC}"
echo -e "${GREEN}================================${NC}"

# Build for ARM (Raspberry Pi)
echo -e "${YELLOW}Building for ARM/Raspberry Pi...${NC}"
GOOS=linux GOARCH=arm GOARM=7 go build -o test-magnetometer-arm test-magnetometer.go

if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi

echo -e "${GREEN}Build successful!${NC}"
ls -lh test-magnetometer-arm

# If IP provided, deploy to Pi
if [ -n "$1" ]; then
    PI_IP="$1"
    echo -e "${YELLOW}Deploying to Raspberry Pi at ${PI_IP}...${NC}"

    scp test-magnetometer-arm root@${PI_IP}:/root/test-magnetometer

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Deployed successfully!${NC}"
        echo -e "${YELLOW}Run on Pi with:${NC}"
        echo -e "  ssh root@${PI_IP}"
        echo -e "  sudo /root/test-magnetometer"
    else
        echo -e "${RED}Deployment failed!${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}No IP provided. Binary ready: test-magnetometer-arm${NC}"
    echo -e "${YELLOW}Deploy manually with:${NC}"
    echo -e "  scp test-magnetometer-arm root@<pi-ip>:/root/test-magnetometer${NC}"
fi

echo -e "${GREEN}Done!${NC}"
