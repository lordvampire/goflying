#!/bin/bash
#
# Build and deploy ICM20948 magnetometer test program to Raspberry Pi
#
# Usage:
#   ./build-test.sh <raspberry-pi-ip> [--run]
#
# Examples:
#   ./build-test.sh 192.168.1.100           # Build and deploy
#   ./build-test.sh 192.168.1.100 --run     # Build, deploy, and run immediately
#

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}ICM20948 Test Build & Deploy${NC}"
echo -e "${GREEN}================================${NC}"

# Check if IP provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Raspberry Pi IP address required!${NC}"
    echo -e "${YELLOW}Usage: ./build-test.sh <pi-ip> [--run]${NC}"
    echo -e "${YELLOW}Example: ./build-test.sh 192.168.1.100 --run${NC}"
    exit 1
fi

PI_IP="$1"
RUN_AFTER="${2:-}"

# Build for ARM (Raspberry Pi)
echo -e "${YELLOW}Building for ARM/Raspberry Pi...${NC}"
GOOS=linux GOARCH=arm GOARM=7 /root/go/bin/go build -o test-magnetometer-arm test-magnetometer.go

if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Build successful!${NC}"
ls -lh test-magnetometer-arm

# Deploy to Pi
echo -e "${YELLOW}Deploying to Raspberry Pi at ${PI_IP}...${NC}"

scp test-magnetometer-arm root@${PI_IP}:/root/test-magnetometer

if [ $? -ne 0 ]; then
    echo -e "${RED}Deployment failed!${NC}"
    exit 1
fi

# Make executable on Pi
ssh root@${PI_IP} "chmod +x /root/test-magnetometer"

echo -e "${GREEN}✓ Deployed successfully!${NC}"

# Run if requested
if [ "$RUN_AFTER" = "--run" ] || [ "$RUN_AFTER" = "-r" ]; then
    echo -e "${GREEN}================================${NC}"
    echo -e "${GREEN}Running test on Raspberry Pi...${NC}"
    echo -e "${GREEN}================================${NC}"
    echo -e "${YELLOW}(Press Ctrl+C to stop)${NC}"
    echo ""

    ssh root@${PI_IP} "sudo /root/test-magnetometer"
else
    echo -e "${YELLOW}Ready to test!${NC}"
    echo -e "${YELLOW}Run on Pi with:${NC}"
    echo -e "  ssh root@${PI_IP} 'sudo /root/test-magnetometer'${NC}"
    echo ""
    echo -e "${YELLOW}Or use --run flag to execute immediately:${NC}"
    echo -e "  ./build-test.sh ${PI_IP} --run${NC}"
fi

echo -e "${GREEN}Done!${NC}"
