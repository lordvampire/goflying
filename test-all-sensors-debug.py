#!/usr/bin/env python3
"""
ICM20948 Full Sensor Test with MAXIMUM DEBUG OUTPUT
This version logs EVERY register read/write for comparison with Go code
"""

import time
import sys
import struct

try:
    import smbus2 as smbus
except ImportError:
    import smbus

ICM20948_ADDR = 0x68
AK09916_ADDR = 0x0C
I2C_BUS = 1

# ICM20948 Registers (Bank 0)
REG_BANK_SEL = 0x7F
WHO_AM_I = 0x00
PWR_MGMT_1 = 0x06
PWR_MGMT_2 = 0x07
USER_CTRL = 0x03
INT_PIN_CFG = 0x0F
LP_CONFIG = 0x05

# ICM20948 Data Registers (Bank 0)
ACCEL_XOUT_H = 0x2D
GYRO_XOUT_H = 0x33
EXT_SENS_DATA_00 = 0x3B

# Bank 2 Registers
GYRO_CONFIG_1 = 0x01
ACCEL_CONFIG = 0x14

# Bank 3 Registers
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

# AK09916 Registers
AK09916_WIA1 = 0x00
AK09916_WIA2 = 0x01
AK09916_ST1 = 0x10
AK09916_HXL = 0x11
AK09916_CNTL2 = 0x31
AK09916_CNTL3 = 0x32

bus = smbus.SMBus(I2C_BUS)
current_bank = None

def set_bank(bank):
    global current_bank
    if current_bank != bank:
        print(f"  [BANK] Switching from Bank {current_bank} to Bank {bank}")
        bus.write_byte_data(ICM20948_ADDR, REG_BANK_SEL, bank << 4)
        current_bank = bank
        time.sleep(0.001)  # Small delay after bank switch

def read_reg(reg):
    value = bus.read_byte_data(ICM20948_ADDR, reg)
    return value

def write_reg(reg, value):
    bus.write_byte_data(ICM20948_ADDR, reg, value)

def write_and_verify(reg, value, name):
    """Write register and verify with readback"""
    write_reg(reg, value)
    time.sleep(0.005)
    readback = read_reg(reg)
    if readback == value:
        print(f"  [WRITE] {name} (0x{reg:02X}) = 0x{value:02X} ✓ (readback: 0x{readback:02X})")
    else:
        print(f"  [WRITE] {name} (0x{reg:02X}) = 0x{value:02X} ✗ ERROR! (readback: 0x{readback:02X})")
    return readback

def decode_status(status):
    """Decode I2C_MST_STATUS bits"""
    bits = []
    if status & 0x80:
        bits.append("PASS_THROUGH")
    if status & 0x40:
        bits.append("SLV4_DONE")
    if status & 0x10:
        bits.append("LOST_ARB")
    if status & 0x08:
        bits.append("SLV4_NACK")
    if status & 0x04:
        bits.append("SLV3_NACK")
    if status & 0x02:
        bits.append("SLV2_NACK")
    if status & 0x01:
        bits.append("SLV1_NACK")
    return " | ".join(bits) if bits else "NONE"

def dump_bank0_regs():
    """Dump important Bank 0 registers"""
    set_bank(0)
    print(f"  [DUMP] Bank 0:")
    who = read_reg(WHO_AM_I)
    pwr1 = read_reg(PWR_MGMT_1)
    pwr2 = read_reg(PWR_MGMT_2)
    user = read_reg(USER_CTRL)
    int_cfg = read_reg(INT_PIN_CFG)
    print(f"    WHO_AM_I    (0x00) = 0x{who:02X}")
    print(f"    PWR_MGMT_1  (0x06) = 0x{pwr1:02X}")
    print(f"    PWR_MGMT_2  (0x07) = 0x{pwr2:02X}")
    print(f"    USER_CTRL   (0x03) = 0x{user:02X} (I2C_MST_EN={bool(user & 0x20)})")
    print(f"    INT_PIN_CFG (0x0F) = 0x{int_cfg:02X} (BYPASS={bool(int_cfg & 0x02)})")

def dump_bank2_regs():
    """Dump important Bank 2 registers"""
    set_bank(2)
    print(f"  [DUMP] Bank 2:")
    gyro = read_reg(GYRO_CONFIG_1)
    accel = read_reg(ACCEL_CONFIG)
    print(f"    GYRO_CONFIG_1 (0x01) = 0x{gyro:02X}")
    print(f"    ACCEL_CONFIG  (0x14) = 0x{accel:02X}")

def dump_bank3_regs():
    """Dump important Bank 3 registers"""
    set_bank(3)
    print(f"  [DUMP] Bank 3:")
    odr = read_reg(I2C_MST_ODR_CONFIG)
    ctrl = read_reg(I2C_MST_CTRL)
    status = read_reg(I2C_MST_STATUS)
    slv0_addr = read_reg(I2C_SLV0_ADDR)
    slv0_reg = read_reg(I2C_SLV0_REG)
    slv0_ctrl = read_reg(I2C_SLV0_CTRL)
    slv4_addr = read_reg(I2C_SLV4_ADDR)
    slv4_reg = read_reg(I2C_SLV4_REG)
    slv4_ctrl = read_reg(I2C_SLV4_CTRL)

    print(f"    I2C_MST_ODR_CONFIG (0x00) = 0x{odr:02X}")
    print(f"    I2C_MST_CTRL       (0x01) = 0x{ctrl:02X}")
    print(f"    I2C_MST_STATUS     (0x17) = 0x{status:02X} [{decode_status(status)}]")
    print(f"    I2C_SLV0_ADDR      (0x03) = 0x{slv0_addr:02X}")
    print(f"    I2C_SLV0_REG       (0x04) = 0x{slv0_reg:02X}")
    print(f"    I2C_SLV0_CTRL      (0x05) = 0x{slv0_ctrl:02X}")
    print(f"    I2C_SLV4_ADDR      (0x13) = 0x{slv4_addr:02X}")
    print(f"    I2C_SLV4_REG       (0x14) = 0x{slv4_reg:02X}")
    print(f"    I2C_SLV4_CTRL      (0x15) = 0x{slv4_ctrl:02X}")

print("=" * 80)
print("ICM20948 Full Sensor Test - MAXIMUM DEBUG MODE")
print("Every register read/write will be logged for Go comparison")
print("=" * 80)

# STEP 1: Reset ICM20948
print("\n" + "=" * 80)
print("STEP 1: Reset ICM20948")
print("=" * 80)
set_bank(0)
write_and_verify(PWR_MGMT_1, 0x80, "PWR_MGMT_1 (RESET)")
print("  [SLEEP] 100ms after reset")
time.sleep(0.1)

write_and_verify(PWR_MGMT_1, 0x01, "PWR_MGMT_1 (WAKE + AUTO CLOCK)")
print("  [SLEEP] 10ms after wake")
time.sleep(0.01)

dump_bank0_regs()

# STEP 2: Enable Gyro and Accel EARLY
print("\n" + "=" * 80)
print("STEP 2: Enable Gyro and Accel (PWR_MGMT_2 = 0x00)")
print("=" * 80)
set_bank(0)
write_and_verify(PWR_MGMT_2, 0x00, "PWR_MGMT_2 (ALL SENSORS ON)")
print("  [SLEEP] 50ms for sensors to start")
time.sleep(0.05)

dump_bank0_regs()

# STEP 3: Configure Gyro and Accel (Bank 2) BEFORE I2C Master
print("\n" + "=" * 80)
print("STEP 3: Configure Gyro and Accel sensitivities (Bank 2)")
print("=" * 80)
set_bank(2)
write_and_verify(GYRO_CONFIG_1, 0x06, "GYRO_CONFIG_1 (±1000 dps)")
write_and_verify(ACCEL_CONFIG, 0x04, "ACCEL_CONFIG (±8g)")

dump_bank2_regs()

# STEP 4: Disable I2C bypass
print("\n" + "=" * 80)
print("STEP 4: Disable I2C bypass mode")
print("=" * 80)
set_bank(0)
write_and_verify(INT_PIN_CFG, 0x00, "INT_PIN_CFG (BYPASS OFF)")
print("  [SLEEP] 10ms")
time.sleep(0.01)

dump_bank0_regs()

# STEP 5: Configure I2C Master
print("\n" + "=" * 80)
print("STEP 5: Configure I2C Master (Bank 3)")
print("=" * 80)
set_bank(3)
write_and_verify(I2C_MST_ODR_CONFIG, 0x04, "I2C_MST_ODR_CONFIG (200 Hz)")
write_and_verify(I2C_MST_CTRL, 0x17, "I2C_MST_CTRL (400 kHz + STOP)")

dump_bank3_regs()

# STEP 6: Enable I2C Master
print("\n" + "=" * 80)
print("STEP 6: Enable I2C Master (USER_CTRL bit 5)")
print("=" * 80)
set_bank(0)
write_and_verify(USER_CTRL, 0x20, "USER_CTRL (I2C_MST_EN)")
print("  [SLEEP] 100ms")
time.sleep(0.1)

dump_bank0_regs()
dump_bank3_regs()

# STEP 7: Test Slave 4 with AK09916 WHO_AM_I
print("\n" + "=" * 80)
print("STEP 7: Test Slave 4 communication with AK09916")
print("=" * 80)
set_bank(3)
print("  [WRITE] I2C_SLV4_ADDR (0x13) = 0x8C (READ from 0x0C)")
write_reg(I2C_SLV4_ADDR, 0x80 | AK09916_ADDR)
print("  [WRITE] I2C_SLV4_REG  (0x14) = 0x00 (WIA1)")
write_reg(I2C_SLV4_REG, AK09916_WIA1)
print("  [WRITE] I2C_SLV4_CTRL (0x15) = 0x80 (ENABLE)")
write_reg(I2C_SLV4_CTRL, 0x80)

print("\n  [DUMP] Slave 4 config readback:")
slv4_addr = read_reg(I2C_SLV4_ADDR)
slv4_reg = read_reg(I2C_SLV4_REG)
slv4_ctrl = read_reg(I2C_SLV4_CTRL)
print(f"    I2C_SLV4_ADDR = 0x{slv4_addr:02X} (expect 0x8C)")
print(f"    I2C_SLV4_REG  = 0x{slv4_reg:02X} (expect 0x00)")
print(f"    I2C_SLV4_CTRL = 0x{slv4_ctrl:02X} (expect 0x80)")

# Poll for SLV4_DONE
print("\n  [POLL] Waiting for SLV4_DONE bit (0x40 in I2C_MST_STATUS)...")
wia1 = 0x00
success = False
for i in range(50):
    time.sleep(0.01)
    status = read_reg(I2C_MST_STATUS)

    if i < 5 or i % 10 == 0 or i == 49:
        print(f"    Poll {i+1:2d}: I2C_MST_STATUS=0x{status:02X} [{decode_status(status)}]")

    if status & 0x40:  # SLV4_DONE
        wia1 = read_reg(I2C_SLV4_DI)
        print(f"\n  [SUCCESS] SLV4_DONE after {(i+1)*10}ms")
        print(f"  [READ] I2C_SLV4_DI (0x17) = 0x{wia1:02X}")
        success = True
        break

if success and wia1 == 0x48:
    print(f"  ✓ AK09916 WHO_AM_I = 0x{wia1:02X} (CORRECT!)")
else:
    print(f"  ✗ FAILED! WIA1 = 0x{wia1:02X} (expect 0x48)")
    dump_bank3_regs()
    sys.exit(1)

# STEP 8: Initialize AK09916 via Slave 4
print("\n" + "=" * 80)
print("STEP 8: Initialize AK09916 magnetometer")
print("=" * 80)

# Reset AK09916
print("  [SLAVE4] Sending soft reset to AK09916...")
set_bank(3)
write_reg(I2C_SLV4_ADDR, AK09916_ADDR)  # Write mode
write_reg(I2C_SLV4_REG, AK09916_CNTL3)
write_reg(I2C_SLV4_DO, 0x01)  # SRST bit
write_reg(I2C_SLV4_CTRL, 0x80)
time.sleep(0.01)
for i in range(10):
    status = read_reg(I2C_MST_STATUS)
    if status & 0x40:
        print(f"    Reset done after {(i+1)*10}ms")
        break
    time.sleep(0.01)
time.sleep(0.1)

# Set continuous mode 3 (50 Hz)
print("  [SLAVE4] Setting continuous measurement mode (50 Hz)...")
write_reg(I2C_SLV4_ADDR, AK09916_ADDR)
write_reg(I2C_SLV4_REG, AK09916_CNTL2)
write_reg(I2C_SLV4_DO, 0x06)  # Mode 3
write_reg(I2C_SLV4_CTRL, 0x80)
time.sleep(0.01)
for i in range(10):
    status = read_reg(I2C_MST_STATUS)
    if status & 0x40:
        print(f"    Mode set after {(i+1)*10}ms")
        break
    time.sleep(0.01)
time.sleep(0.1)

# Verify mode
print("  [SLAVE4] Verifying CNTL2 register...")
write_reg(I2C_SLV4_ADDR, 0x80 | AK09916_ADDR)
write_reg(I2C_SLV4_REG, AK09916_CNTL2)
write_reg(I2C_SLV4_CTRL, 0x80)
time.sleep(0.05)
for i in range(10):
    status = read_reg(I2C_MST_STATUS)
    if status & 0x40:
        cntl2 = read_reg(I2C_SLV4_DI)
        print(f"    CNTL2 = 0x{cntl2:02X} (expect 0x06)")
        if cntl2 == 0x06:
            print("    ✓ Continuous mode confirmed")
        break
    time.sleep(0.01)

# STEP 9: Configure Slave 0 for continuous mag reads
print("\n" + "=" * 80)
print("STEP 9: Configure Slave 0 for continuous mag reads")
print("=" * 80)
set_bank(3)
write_and_verify(I2C_SLV0_ADDR, 0x80 | AK09916_ADDR, "I2C_SLV0_ADDR (READ from 0x0C)")
write_and_verify(I2C_SLV0_REG, AK09916_ST1, "I2C_SLV0_REG (ST1=0x10)")
write_and_verify(I2C_SLV0_CTRL, 0x80 | 9, "I2C_SLV0_CTRL (ENABLE, 9 bytes)")
print("  [SLEEP] 100ms")
time.sleep(0.1)

dump_bank3_regs()

# STEP 10: Final register dump before reading data
print("\n" + "=" * 80)
print("STEP 10: Final register dump before reading sensor data")
print("=" * 80)
dump_bank0_regs()
dump_bank2_regs()
dump_bank3_regs()

# STEP 11: Read sensor data
print("\n" + "=" * 80)
print("STEP 11: Reading sensor data (5 samples)")
print("=" * 80)

set_bank(0)

for sample in range(5):
    time.sleep(0.1)

    # Read accel
    accel_data = bus.read_i2c_block_data(ICM20948_ADDR, ACCEL_XOUT_H, 6)
    ax_raw = struct.unpack('>h', bytes(accel_data[0:2]))[0]
    ay_raw = struct.unpack('>h', bytes(accel_data[2:4]))[0]
    az_raw = struct.unpack('>h', bytes(accel_data[4:6]))[0]

    # Read gyro
    gyro_data = bus.read_i2c_block_data(ICM20948_ADDR, GYRO_XOUT_H, 6)
    gx_raw = struct.unpack('>h', bytes(gyro_data[0:2]))[0]
    gy_raw = struct.unpack('>h', bytes(gyro_data[2:4]))[0]
    gz_raw = struct.unpack('>h', bytes(gyro_data[4:6]))[0]

    # Read mag from EXT_SENS_DATA
    mag_data = bus.read_i2c_block_data(ICM20948_ADDR, EXT_SENS_DATA_00, 9)
    st1 = mag_data[0]

    print(f"\n[Sample {sample+1}]")
    print(f"  Accel raw: X={ax_raw:6d}, Y={ay_raw:6d}, Z={az_raw:6d}")
    print(f"  Gyro  raw: X={gx_raw:6d}, Y={gy_raw:6d}, Z={gz_raw:6d}")
    print(f"  Mag   ST1: 0x{st1:02X} (DRDY={bool(st1 & 0x01)})")

    if st1 & 0x01:
        mx_raw = struct.unpack('<h', bytes(mag_data[1:3]))[0]
        my_raw = struct.unpack('<h', bytes(mag_data[3:5]))[0]
        mz_raw = struct.unpack('<h', bytes(mag_data[5:7]))[0]
        st2 = mag_data[7]
        print(f"  Mag   raw: X={mx_raw:6d}, Y={my_raw:6d}, Z={mz_raw:6d}, ST2=0x{st2:02X}")
    else:
        print(f"  Mag: NO DATA READY")

print("\n" + "=" * 80)
print("TEST COMPLETE - Check logs above for comparison with Go code")
print("=" * 80)

bus.close()
