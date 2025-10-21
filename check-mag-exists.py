#!/usr/bin/env python3
"""
Check if AK09916 magnetometer exists in ICM20948
Tests via I2C BYPASS mode - direct access without I2C Master
"""

import time
import sys

try:
    import smbus2 as smbus
except ImportError:
    try:
        import smbus
    except ImportError:
        print("ERROR: smbus not found!")
        print("Install with: sudo apt install python3-smbus")
        sys.exit(1)

# Constants
ICM20948_ADDR = 0x68
I2C_BUS = 1

# ICM20948 Registers (Bank 0)
REG_BANK_SEL = 0x7F
PWR_MGMT_1 = 0x06
INT_PIN_CFG = 0x0F

# AK09916 possible addresses
AK09916_ADDR_LIST = [0x0C, 0x0D, 0x0E, 0x0F]

# AK09916 Registers
AK09916_WIA1 = 0x00  # Should return 0x48
AK09916_WIA2 = 0x01  # Should return 0x09

print("=" * 60)
print("ICM20948: Check if magnetometer chip exists")
print("=" * 60)

bus = smbus.SMBus(I2C_BUS)

# Wake up ICM20948
print("\n1. Waking up ICM20948...")
bus.write_byte_data(ICM20948_ADDR, REG_BANK_SEL, 0x00)
bus.write_byte_data(ICM20948_ADDR, PWR_MGMT_1, 0x80)  # Reset
time.sleep(0.1)
bus.write_byte_data(ICM20948_ADDR, PWR_MGMT_1, 0x01)  # Wake
time.sleep(0.01)
print("✓ ICM20948 is awake")

# Check ICM20948 WHO_AM_I
whoami = bus.read_byte_data(ICM20948_ADDR, 0x00)
print(f"  ICM20948 WHO_AM_I: 0x{whoami:02X} (expect 0xEA)")

# Enable I2C bypass mode
print("\n2. Enabling I2C BYPASS mode...")
print("   This routes the auxiliary I2C bus to external pins")
print("   Magnetometer should appear as separate I2C device")

bus.write_byte_data(ICM20948_ADDR, INT_PIN_CFG, 0x02)  # Enable bypass
time.sleep(0.1)

int_pin_cfg = bus.read_byte_data(ICM20948_ADDR, INT_PIN_CFG)
print(f"  INT_PIN_CFG = 0x{int_pin_cfg:02X} (bit 1 should be set)")

# Scan entire I2C bus for all devices
print("\n3. Scanning ENTIRE I2C bus for all devices...")
print("   (This shows what's actually connected)")
found_devices = []

for addr in range(0x03, 0x78):
    try:
        bus.read_byte(addr)
        found_devices.append(addr)
        print(f"  0x{addr:02X}: Device found!")
    except:
        pass

if not found_devices:
    print("  ❌ No I2C devices found at all!")
else:
    print(f"\n  ✓ Found {len(found_devices)} device(s): {[hex(a) for a in found_devices]}")

# Try to read AK09916 directly on known addresses
print("\n4. Trying to read AK09916 WHO_AM_I directly...")
print("   Testing addresses 0x0C, 0x0D, 0x0E, 0x0F")

found_mag = False
for addr in AK09916_ADDR_LIST:
    try:
        wia1 = bus.read_byte_data(addr, AK09916_WIA1)
        wia2 = bus.read_byte_data(addr, AK09916_WIA2)

        print(f"  0x{addr:02X}: WIA1=0x{wia1:02X}, WIA2=0x{wia2:02X}")

        if wia1 == 0x48 and wia2 == 0x09:
            print(f"       ✅ FOUND AK09916 at address 0x{addr:02X}!")
            found_mag = True
        elif wia1 != 0xFF or wia2 != 0xFF:
            print(f"       ⚠️  Device responds but not AK09916")
    except Exception as e:
        print(f"  0x{addr:02X}: No response ({e})")

# Disable bypass mode
print("\n5. Disabling I2C bypass mode...")
bus.write_byte_data(ICM20948_ADDR, INT_PIN_CFG, 0x00)
print("  ✓ Bypass disabled")

# Results
print("\n" + "=" * 60)
print("RESULTS:")
print("=" * 60)

if found_mag:
    print("✅ SUCCESS: AK09916 magnetometer IS present!")
    print("   The chip exists and responds correctly.")
    print("   Problem is with I2C Master configuration.")
elif found_devices:
    print("⚠️  PARTIAL: I2C bus works, but no AK09916 found")
    print("   Your ICM20948 might:")
    print("   - Have a different magnetometer chip")
    print("   - Have no magnetometer at all")
    print("   - Have a broken/disconnected magnetometer")
    print("\n   Devices found on I2C bus:")
    for addr in found_devices:
        print(f"   - 0x{addr:02X}")
else:
    print("❌ FAILURE: No I2C devices found at all!")
    print("   This could mean:")
    print("   - I2C bus not working")
    print("   - Wrong I2C bus number")
    print("   - Hardware issue")

print("\n" + "=" * 60)
print("NEXT STEPS:")
print("=" * 60)

if found_mag:
    print("Since AK09916 is present, we need to debug I2C Master setup.")
    print("The chip is there - just need to configure ICM20948 correctly.")
else:
    print("1. Check your ICM20948 board/module documentation")
    print("2. Verify it actually includes a magnetometer")
    print("3. Run: sudo i2cdetect -y 1")
    print("4. Share the model/part number of your ICM20948 module")

print("=" * 60)

bus.close()
