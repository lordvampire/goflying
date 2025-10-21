#!/usr/bin/env python3
"""
Test AK09916 magnetometer in BYPASS mode
Since we confirmed the chip exists, let's read actual magnetometer data
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
        sys.exit(1)

# Constants
ICM20948_ADDR = 0x68
AK09916_ADDR = 0x0C
I2C_BUS = 1

# ICM20948 Registers
REG_BANK_SEL = 0x7F
PWR_MGMT_1 = 0x06
INT_PIN_CFG = 0x0F

# AK09916 Registers
AK09916_WIA1 = 0x00
AK09916_WIA2 = 0x01
AK09916_ST1 = 0x10    # Status 1
AK09916_HXL = 0x11    # X-axis data low
AK09916_HXH = 0x12    # X-axis data high
AK09916_HYL = 0x13    # Y-axis data low
AK09916_HYH = 0x14    # Y-axis data high
AK09916_HZL = 0x15    # Z-axis data low
AK09916_HZH = 0x16    # Z-axis data high
AK09916_ST2 = 0x18    # Status 2
AK09916_CNTL2 = 0x31  # Control 2
AK09916_CNTL3 = 0x32  # Control 3

print("=" * 60)
print("AK09916 Magnetometer Test (Bypass Mode)")
print("=" * 60)

bus = smbus.SMBus(I2C_BUS)

# Wake up ICM20948
print("\n1. Initializing ICM20948...")
bus.write_byte_data(ICM20948_ADDR, REG_BANK_SEL, 0x00)
bus.write_byte_data(ICM20948_ADDR, PWR_MGMT_1, 0x80)  # Reset
time.sleep(0.1)
bus.write_byte_data(ICM20948_ADDR, PWR_MGMT_1, 0x01)  # Wake
time.sleep(0.01)
print("✓ ICM20948 ready")

# Enable bypass mode
print("\n2. Enabling I2C bypass mode...")
bus.write_byte_data(ICM20948_ADDR, INT_PIN_CFG, 0x02)
time.sleep(0.05)
print("✓ Bypass enabled - AK09916 is now directly accessible")

# Verify AK09916
print("\n3. Verifying AK09916...")
wia1 = bus.read_byte_data(AK09916_ADDR, AK09916_WIA1)
wia2 = bus.read_byte_data(AK09916_ADDR, AK09916_WIA2)
print(f"  WIA1: 0x{wia1:02X} (expect 0x48)")
print(f"  WIA2: 0x{wia2:02X} (expect 0x09)")

if wia1 != 0x48 or wia2 != 0x09:
    print("❌ ERROR: AK09916 not responding correctly!")
    sys.exit(1)

print("✓ AK09916 confirmed")

# Soft reset AK09916
print("\n4. Resetting AK09916...")
bus.write_byte_data(AK09916_ADDR, AK09916_CNTL3, 0x01)  # SRST
time.sleep(0.1)
print("✓ AK09916 reset complete")

# Set to continuous measurement mode 3 (50 Hz)
print("\n5. Setting continuous measurement mode (50 Hz)...")
bus.write_byte_data(AK09916_ADDR, AK09916_CNTL2, 0x06)  # Continuous mode 3
time.sleep(0.1)

cntl2 = bus.read_byte_data(AK09916_ADDR, AK09916_CNTL2)
print(f"  CNTL2: 0x{cntl2:02X} (expect 0x06)")

if cntl2 != 0x06:
    print(f"⚠️  Warning: CNTL2 readback is 0x{cntl2:02X}, expected 0x06")
else:
    print("✓ Continuous mode enabled")

# Read magnetometer data
print("\n6. Reading magnetometer data...")
print("  (Press Ctrl+C to stop)")
print()

try:
    count = 0
    valid_readings = 0

    while count < 20:  # Read 20 samples
        count += 1

        # Read status 1
        st1 = bus.read_byte_data(AK09916_ADDR, AK09916_ST1)

        # Check if data is ready (bit 0)
        if st1 & 0x01:
            # Read magnetometer data (6 bytes)
            data = bus.read_i2c_block_data(AK09916_ADDR, AK09916_HXL, 7)

            # Extract raw values (16-bit signed)
            hx = (data[1] << 8) | data[0]
            hy = (data[3] << 8) | data[2]
            hz = (data[5] << 8) | data[4]
            st2 = data[6]

            # Convert to signed
            if hx > 32767: hx -= 65536
            if hy > 32767: hy -= 65536
            if hz > 32767: hz -= 65536

            # Convert to µT (AK09916: 4912 µT range, 16-bit)
            scale = 4912.0 / 32752.0
            mx = hx * scale
            my = hy * scale
            mz = hz * scale

            print(f"  [{count:3d}] ST1=0x{st1:02X} | ", end="")
            print(f"Raw: X={hx:6d} Y={hy:6d} Z={hz:6d} | ", end="")
            print(f"µT: X={mx:7.2f} Y={my:7.2f} Z={mz:7.2f} | ST2=0x{st2:02X}")

            valid_readings += 1

            # Check for overflow (bit 3 in ST2)
            if st2 & 0x08:
                print(f"       ⚠️  Magnetic sensor overflow!")
        else:
            print(f"  [{count:3d}] Data not ready (ST1=0x{st1:02X})")

        time.sleep(0.2)  # 5 Hz read rate

    print()
    print("=" * 60)
    print("RESULTS:")
    print("=" * 60)
    print(f"Total readings: {count}")
    print(f"Valid readings: {valid_readings}")

    if valid_readings > 0:
        print()
        print("✅ SUCCESS! Magnetometer is working in bypass mode!")
        print()
        print("This confirms:")
        print("  ✓ AK09916 chip is functional")
        print("  ✓ I2C connection is good")
        print("  ✓ Magnetometer can measure magnetic fields")
        print()
        print("❌ BUT: I2C Master mode is NOT working")
        print()
        print("NEXT STEP: Debug I2C Master configuration")
        print("The chip works fine - we just need to fix the I2C Master setup")
    else:
        print()
        print("❌ No valid readings received")
        print("Magnetometer is present but not providing data")

except KeyboardInterrupt:
    print("\n\nStopped by user")

# Disable bypass
print("\n7. Disabling bypass mode...")
bus.write_byte_data(ICM20948_ADDR, INT_PIN_CFG, 0x00)
print("✓ Bypass disabled")

bus.close()
print("\nDone!")
