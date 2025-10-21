#!/usr/bin/env python3
"""
Debug I2C Master configuration
Compare what we set vs what's actually in the registers
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
USER_CTRL = 0x03
INT_PIN_CFG = 0x0F
LP_CONFIG = 0x05

# Bank 3 registers
I2C_MST_ODR_CONFIG = 0x00
I2C_MST_CTRL = 0x01
I2C_MST_STATUS = 0x17
I2C_SLV0_ADDR = 0x03
I2C_SLV0_REG = 0x04
I2C_SLV0_CTRL = 0x05
I2C_SLV4_ADDR = 0x13
I2C_SLV4_REG = 0x14
I2C_SLV4_CTRL = 0x15
I2C_SLV4_DO = 0x16
I2C_SLV4_DI = 0x17

# Bank 0 EXT_SENS registers
EXT_SENS_DATA_00 = 0x3B

bus = smbus.SMBus(I2C_BUS)

def set_bank(bank):
    bus.write_byte_data(ICM20948_ADDR, REG_BANK_SEL, bank << 4)

def read_reg(reg):
    return bus.read_byte_data(ICM20948_ADDR, reg)

def write_reg(reg, value):
    bus.write_byte_data(ICM20948_ADDR, reg, value)

print("=" * 70)
print("I2C Master Configuration Debug")
print("=" * 70)

# Initialize
print("\n1. Initializing ICM20948...")
set_bank(0)
write_reg(PWR_MGMT_1, 0x80)  # Reset
time.sleep(0.1)
write_reg(PWR_MGMT_1, 0x01)  # Wake
time.sleep(0.01)
print("✓ ICM20948 ready")

# Setup I2C Master (same as our code)
print("\n2. Configuring I2C Master...")

set_bank(0)
print("  Step A: Disable bypass")
write_reg(INT_PIN_CFG, 0x00)
time.sleep(0.01)

print("  Step B: Reset I2C Master and SRAM")
write_reg(USER_CTRL, 0x02 | 0x04)  # I2C_MST_RST + SRAM_RST
time.sleep(0.1)

print("  Step C: Clear I2C_MST_CYCLE")
lp_config = read_reg(LP_CONFIG)
if lp_config & 0x40:
    write_reg(LP_CONFIG, lp_config & ~0x40)

set_bank(3)
print("  Step D: Set I2C Master ODR = 0x04 (200 Hz)")
write_reg(I2C_MST_ODR_CONFIG, 0x04)

print("  Step E: Set I2C Master CTRL = 0x17 (400kHz + STOP)")
write_reg(I2C_MST_CTRL, 0x17)

set_bank(0)
print("  Step F: Enable I2C Master")
write_reg(USER_CTRL, 0x20)  # BIT_I2C_MST_EN
time.sleep(0.1)

print("✓ I2C Master configured")

# NOW READ BACK ALL REGISTERS
print("\n3. Reading back ALL configuration registers...")
print("=" * 70)

set_bank(0)
print("\nBANK 0:")
print(f"  PWR_MGMT_1     (0x06): 0x{read_reg(PWR_MGMT_1):02X}")
print(f"  USER_CTRL      (0x03): 0x{read_reg(USER_CTRL):02X} (expect 0x20 = I2C_MST_EN)")
print(f"  INT_PIN_CFG    (0x0F): 0x{read_reg(INT_PIN_CFG):02X} (expect 0x00 = bypass OFF)")
print(f"  LP_CONFIG      (0x05): 0x{read_reg(LP_CONFIG):02X} (bit 6 should be 0)")

set_bank(3)
print("\nBANK 3:")
i2c_mst_odr = read_reg(I2C_MST_ODR_CONFIG)
i2c_mst_ctrl = read_reg(I2C_MST_CTRL)
i2c_mst_status = read_reg(I2C_MST_STATUS)

print(f"  I2C_MST_ODR_CONFIG (0x00): 0x{i2c_mst_odr:02X} (expect 0x04)")
print(f"  I2C_MST_CTRL       (0x01): 0x{i2c_mst_ctrl:02X} (expect 0x17)")
print(f"  I2C_MST_STATUS     (0x17): 0x{i2c_mst_status:02X}")
print(f"    Bit 6 (I2C_SLV4_DONE):  {(i2c_mst_status >> 6) & 1}")
print(f"    Bit 4 (I2C_LOST_ARB):   {(i2c_mst_status >> 4) & 1}")
print(f"    Bit 0 (I2C_SLV0_NACK):  {(i2c_mst_status >> 0) & 1}")

# Try to read AK09916 via Slave 4
print("\n4. Testing Slave 4 read from AK09916...")

set_bank(3)
print("  Configuring Slave 4 to read AK09916 WIA1...")
write_reg(I2C_SLV4_ADDR, 0x80 | AK09916_ADDR)  # Read from 0x0C
write_reg(I2C_SLV4_REG, 0x00)  # WIA1 register
write_reg(I2C_SLV4_CTRL, 0x80)  # Enable

# Poll for SLV4_DONE (like JeVois does)
print("  Waiting for SLV4_DONE bit (polling up to 300ms)...")
done = False
for i in range(30):  # 30 * 10ms = 300ms
    time.sleep(0.01)
    i2c_mst_status = read_reg(I2C_MST_STATUS)
    if i2c_mst_status & 0x40:  # SLV4_DONE
        print(f"    ✓ SLV4_DONE set after {(i+1)*10}ms")
        done = True
        break
    if i2c_mst_status & 0x10:  # LOST_ARB
        print(f"    ❌ Lost arbitration after {(i+1)*10}ms")
        break

if not done:
    print(f"    ❌ Timeout: SLV4_DONE never set (waited 300ms)")

slv4_di = read_reg(I2C_SLV4_DI)

print(f"  I2C_SLV4_DI (data):   0x{slv4_di:02X} (expect 0x48)")
print(f"  I2C_MST_STATUS:       0x{i2c_mst_status:02X}")
print(f"    Bit 6 (SLV4_DONE):  {(i2c_mst_status >> 6) & 1} (should be 1)")
print(f"    Bit 4 (LOST_ARB):   {(i2c_mst_status >> 4) & 1} (should be 0)")
print(f"    Bit 0 (SLV0_NACK):  {(i2c_mst_status >> 0) & 1}")

if slv4_di == 0x48 and (i2c_mst_status & 0x40):
    print("\n  ✅ SUCCESS! I2C Master CAN read AK09916!")
    print("     Problem might be in continuous read setup (Slave 0)")
elif slv4_di == 0x00:
    print("\n  ❌ FAILURE: Read returned 0x00")
    print("     I2C Master is not reaching the magnetometer")
else:
    print(f"\n  ⚠️  UNEXPECTED: Read returned 0x{slv4_di:02X}")

# Check Slave 0 configuration
print("\n5. Checking Slave 0 configuration...")
set_bank(3)
slv0_addr = read_reg(I2C_SLV0_ADDR)
slv0_reg = read_reg(I2C_SLV0_REG)
slv0_ctrl = read_reg(I2C_SLV0_CTRL)

print(f"  I2C_SLV0_ADDR: 0x{slv0_addr:02X} (expect 0x8C = read from 0x0C)")
print(f"  I2C_SLV0_REG:  0x{slv0_reg:02X} (expect 0x10 = ST1)")
print(f"  I2C_SLV0_CTRL: 0x{slv0_ctrl:02X} (expect 0x89 = enabled, 9 bytes)")

# Read EXT_SENS_DATA
print("\n6. Reading EXT_SENS_DATA registers (Slave 0 data)...")
set_bank(0)
ext_data = []
for i in range(10):
    ext_data.append(read_reg(EXT_SENS_DATA_00 + i))

print(f"  EXT_SENS_DATA_00-09: {' '.join([f'{b:02X}' for b in ext_data])}")
print(f"  Byte 0 (ST1):  0x{ext_data[0]:02X} (should have bit 0 set if data ready)")
print(f"  Byte 1-2 (MX): 0x{ext_data[2]:02X}{ext_data[1]:02X}")
print(f"  Byte 3-4 (MY): 0x{ext_data[4]:02X}{ext_data[3]:02X}")
print(f"  Byte 5-6 (MZ): 0x{ext_data[6]:02X}{ext_data[5]:02X}")
print(f"  Byte 7 (ST2):  0x{ext_data[7]:02X}")

if all(b == 0 for b in ext_data):
    print("\n  ❌ All EXT_SENS_DATA is 0x00")
    print("     Slave 0 continuous read is NOT working")
else:
    print("\n  ✅ EXT_SENS_DATA has non-zero values")
    print("     Slave 0 continuous read might be working")

# Summary
print("\n" + "=" * 70)
print("SUMMARY:")
print("=" * 70)

if slv4_di == 0x48:
    print("✅ I2C Master CAN communicate with AK09916 (Slave 4 works)")
    print()
    if all(b == 0 for b in ext_data):
        print("❌ BUT: Continuous read (Slave 0) is NOT working")
        print()
        print("PROBLEM: Slave 0 is not transferring data to EXT_SENS_DATA")
        print()
        print("Possible causes:")
        print("  1. Slave 0 not configured correctly")
        print("  2. AK09916 not in continuous mode")
        print("  3. I2C Master ODR timing issue")
        print("  4. Need to trigger reads manually")
    else:
        print("✅ Continuous read (Slave 0) IS working!")
        print("   Check if data is being updated")
else:
    print("❌ I2C Master CANNOT communicate with AK09916")
    print()
    print("This is strange because:")
    print("  - Bypass mode works fine")
    print("  - Chip exists at 0x0C")
    print("  - I2C Master is enabled")
    print()
    print("Check:")
    print("  - Is there a hardware switch/jumper for bypass mode?")
    print("  - Are there pull-up resistors on aux I2C lines?")

bus.close()
