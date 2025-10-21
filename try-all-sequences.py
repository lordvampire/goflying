#!/usr/bin/env python3
"""
Try DIFFERENT I2C Master initialization sequences
One of them might work!
"""

import time
import sys

try:
    import smbus2 as smbus
except ImportError:
    import smbus

ICM20948_ADDR = 0x68
AK09916_ADDR = 0x0C
I2C_BUS = 1

REG_BANK_SEL = 0x7F
PWR_MGMT_1 = 0x06
PWR_MGMT_2 = 0x07
USER_CTRL = 0x03
INT_PIN_CFG = 0x0F
LP_CONFIG = 0x05

I2C_MST_ODR_CONFIG = 0x00
I2C_MST_CTRL = 0x01
I2C_MST_STATUS = 0x17
I2C_SLV4_ADDR = 0x13
I2C_SLV4_REG = 0x14
I2C_SLV4_CTRL = 0x15
I2C_SLV4_DI = 0x17

bus = smbus.SMBus(I2C_BUS)

def set_bank(bank):
    bus.write_byte_data(ICM20948_ADDR, REG_BANK_SEL, bank << 4)

def read_reg(reg):
    return bus.read_byte_data(ICM20948_ADDR, reg)

def write_reg(reg, value):
    bus.write_byte_data(ICM20948_ADDR, reg, value)

def reset_device():
    """Full device reset"""
    set_bank(0)
    write_reg(PWR_MGMT_1, 0x80)  # Device reset
    time.sleep(0.1)
    write_reg(PWR_MGMT_1, 0x01)  # Wake up, auto clock
    time.sleep(0.01)

def test_slv4_read():
    """Try to read AK09916 WHO_AM_I via Slave 4"""
    set_bank(3)
    write_reg(I2C_SLV4_ADDR, 0x80 | AK09916_ADDR)
    write_reg(I2C_SLV4_REG, 0x00)  # WIA1
    write_reg(I2C_SLV4_CTRL, 0x80)  # Enable

    # Poll for done
    for i in range(50):  # 500ms
        time.sleep(0.01)
        status = read_reg(I2C_MST_STATUS)
        if status & 0x40:  # SLV4_DONE
            data = read_reg(I2C_SLV4_DI)
            return (True, data, (i+1)*10)

    return (False, 0x00, 500)

print("=" * 70)
print("Trying DIFFERENT I2C Master initialization sequences")
print("=" * 70)

sequences = [
    {
        "name": "Sequence 1: Our current approach",
        "steps": [
            ("Reset device", lambda: reset_device()),
            ("Disable bypass", lambda: (set_bank(0), write_reg(INT_PIN_CFG, 0x00))),
            ("Reset I2C Master", lambda: write_reg(USER_CTRL, 0x06)),  # I2C_MST_RST + SRAM_RST
            ("Wait 100ms", lambda: time.sleep(0.1)),
            ("Clear LP_CONFIG", lambda: write_reg(LP_CONFIG, 0x00)),
            ("Set I2C_MST_ODR", lambda: (set_bank(3), write_reg(I2C_MST_ODR_CONFIG, 0x04))),
            ("Set I2C_MST_CTRL", lambda: write_reg(I2C_MST_CTRL, 0x17)),
            ("Enable I2C Master", lambda: (set_bank(0), write_reg(USER_CTRL, 0x20))),
            ("Wait 100ms", lambda: time.sleep(0.1)),
        ]
    },
    {
        "name": "Sequence 2: Enable gyro/accel first (for clock)",
        "steps": [
            ("Reset device", lambda: reset_device()),
            ("Enable gyro/accel", lambda: write_reg(PWR_MGMT_2, 0x00)),  # All on
            ("Wait 50ms", lambda: time.sleep(0.05)),
            ("Disable bypass", lambda: write_reg(INT_PIN_CFG, 0x00)),
            ("Set I2C_MST_ODR", lambda: (set_bank(3), write_reg(I2C_MST_ODR_CONFIG, 0x04))),
            ("Set I2C_MST_CTRL", lambda: write_reg(I2C_MST_CTRL, 0x17)),
            ("Enable I2C Master", lambda: (set_bank(0), write_reg(USER_CTRL, 0x20))),
            ("Wait 100ms", lambda: time.sleep(0.1)),
        ]
    },
    {
        "name": "Sequence 3: PX4-style (with I2C_IF_DIS, for SPI)",
        "steps": [
            ("Reset device", lambda: reset_device()),
            ("Enable gyro/accel", lambda: write_reg(PWR_MGMT_2, 0x00)),
            ("USER_CTRL reset", lambda: write_reg(USER_CTRL, 0x36)),  # All reset bits
            ("Wait 100ms", lambda: time.sleep(0.1)),
            ("Set I2C_MST_ODR", lambda: (set_bank(3), write_reg(I2C_MST_ODR_CONFIG, 0x04))),
            ("Set I2C_MST_CTRL", lambda: write_reg(I2C_MST_CTRL, 0x17)),
            ("Enable I2C Master", lambda: (set_bank(0), write_reg(USER_CTRL, 0x30))),  # I2C_MST_EN + I2C_IF_DIS
            ("Wait 100ms", lambda: time.sleep(0.1)),
        ]
    },
    {
        "name": "Sequence 4: Initialize mag in bypass first, then switch",
        "steps": [
            ("Reset device", lambda: reset_device()),
            ("Enable bypass", lambda: write_reg(INT_PIN_CFG, 0x02)),
            ("Wait 50ms", lambda: time.sleep(0.05)),
            ("Reset AK09916", lambda: bus.write_byte_data(AK09916_ADDR, 0x32, 0x01)),
            ("Wait 100ms", lambda: time.sleep(0.1)),
            ("Disable bypass", lambda: write_reg(INT_PIN_CFG, 0x00)),
            ("Wait 50ms", lambda: time.sleep(0.05)),
            ("Set I2C_MST_ODR", lambda: (set_bank(3), write_reg(I2C_MST_ODR_CONFIG, 0x04))),
            ("Set I2C_MST_CTRL", lambda: write_reg(I2C_MST_CTRL, 0x17)),
            ("Enable I2C Master", lambda: (set_bank(0), write_reg(USER_CTRL, 0x20))),
            ("Wait 100ms", lambda: time.sleep(0.1)),
        ]
    },
    {
        "name": "Sequence 5: Lower I2C clock (345.6 kHz instead of 400)",
        "steps": [
            ("Reset device", lambda: reset_device()),
            ("Disable bypass", lambda: write_reg(INT_PIN_CFG, 0x00)),
            ("Set I2C_MST_ODR", lambda: (set_bank(3), write_reg(I2C_MST_ODR_CONFIG, 0x04))),
            ("Set I2C_MST_CTRL", lambda: write_reg(I2C_MST_CTRL, 0x07)),  # 345.6 kHz, no STOP bit
            ("Enable I2C Master", lambda: (set_bank(0), write_reg(USER_CTRL, 0x20))),
            ("Wait 100ms", lambda: time.sleep(0.1)),
        ]
    },
]

for seq in sequences:
    print(f"\n{'='*70}")
    print(f"Testing: {seq['name']}")
    print(f"{'='*70}")

    # Execute sequence
    for step_name, step_func in seq['steps']:
        print(f"  {step_name}...")
        try:
            step_func()
        except Exception as e:
            print(f"    ERROR: {e}")
            break

    # Test Slave 4 read
    print("\n  Testing Slave 4 read...")
    success, data, ms = test_slv4_read()

    if success and data == 0x48:
        print(f"    ✅ SUCCESS! Read 0x{data:02X} after {ms}ms")
        print(f"\n{'='*70}")
        print(f"FOUND WORKING SEQUENCE: {seq['name']}")
        print(f"{'='*70}")
        break
    elif success:
        print(f"    ⚠️  Transaction completed but got 0x{data:02X} (not 0x48)")
    else:
        print(f"    ❌ FAILED: Timeout after {ms}ms")

else:
    print(f"\n{'='*70}")
    print(f"❌ NONE OF THE SEQUENCES WORKED")
    print(f"{'='*70}")
    print()
    print("This suggests a fundamental issue:")
    print("  1. Hardware problem (aux I2C lines not connected internally)")
    print("  2. ICM20948 variant without functional I2C Master")
    print("  3. Missing hardware requirement (pull-ups, power, etc.)")
    print()
    print("RECOMMENDATION: Check your ICM20948 board documentation")
    print("                Does it support the internal magnetometer via I2C Master?")

bus.close()
