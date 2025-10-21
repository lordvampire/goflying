#!/usr/bin/env python3
"""
Simple ICM20948 Magnetometer Test
Tests if the AK09916 magnetometer responds via ICM20948 I2C Master

This script directly reads I2C registers to test magnetometer communication.
No complex dependencies - just direct register reads.
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
USER_CTRL = 0x03
INT_PIN_CFG = 0x0F
LP_CONFIG = 0x05
EXT_SENS_DATA_00 = 0x3B

# ICM20948 Registers (Bank 3)
I2C_MST_ODR_CONFIG = 0x00
I2C_MST_CTRL = 0x01
I2C_SLV0_ADDR = 0x03
I2C_SLV0_REG = 0x04
I2C_SLV0_CTRL = 0x05
I2C_SLV4_ADDR = 0x13
I2C_SLV4_REG = 0x14
I2C_SLV4_CTRL = 0x15
I2C_SLV4_DO = 0x16
I2C_SLV4_DI = 0x17

# AK09916 Constants
AK09916_I2C_ADDR = 0x0C
AK09916_WIA1 = 0x00  # Should return 0x48
AK09916_WIA2 = 0x01  # Should return 0x09
AK09916_ST1 = 0x10
AK09916_CNTL2 = 0x31
AK09916_CNTL3 = 0x32

# Bits
BIT_I2C_MST_EN = 0x20
BIT_I2C_READ = 0x80
BIT_SLAVE_EN = 0x80
BIT_I2C_MST_P_NSR = 0x10


class ICM20948:
    def __init__(self, bus_num=1, address=0x68):
        self.bus = smbus.SMBus(bus_num)
        self.addr = address
        print(f"ICM20948: Opened I2C bus {bus_num}, address 0x{address:02X}")

    def set_bank(self, bank):
        """Switch register bank (0-3)"""
        self.bus.write_byte_data(self.addr, REG_BANK_SEL, bank << 4)

    def read_reg(self, reg):
        """Read single register from current bank"""
        return self.bus.read_byte_data(self.addr, reg)

    def write_reg(self, reg, value):
        """Write single register to current bank"""
        self.bus.write_byte_data(self.addr, reg, value)

    def check_whoami(self):
        """Check ICM20948 WHO_AM_I"""
        self.set_bank(0)
        whoami = self.read_reg(0x00)
        print(f"ICM20948: WHO_AM_I = 0x{whoami:02X} (expect 0xEA)")
        return whoami == 0xEA

    def init_basic(self):
        """Basic initialization"""
        print("\n=== ICM20948 Basic Init ===")

        # Bank 0: Wake up device
        self.set_bank(0)
        self.write_reg(PWR_MGMT_1, 0x80)  # Reset
        time.sleep(0.1)
        self.write_reg(PWR_MGMT_1, 0x01)  # Auto-select clock
        time.sleep(0.01)
        print("✓ Device woken up")

    def setup_mag_i2c_master(self):
        """Setup I2C Master to communicate with AK09916"""
        print("\n=== Setting up I2C Master for AK09916 ===")

        # Bank 0: Disable bypass, disable duty-cycle
        self.set_bank(0)

        print("Step 1: Disable I2C bypass mode")
        self.write_reg(INT_PIN_CFG, 0x00)
        time.sleep(0.01)

        print("Step 2: Disable I2C_MST_CYCLE (enable continuous mode)")
        lp_config = self.read_reg(LP_CONFIG)
        print(f"  LP_CONFIG = 0x{lp_config:02X}")
        if lp_config & 0x40:
            self.write_reg(LP_CONFIG, lp_config & ~0x40)
            print("  ✓ Cleared I2C_MST_CYCLE bit")

        # Bank 3: Configure I2C Master
        self.set_bank(3)

        print("Step 3: Set I2C Master ODR to 200 Hz")
        self.write_reg(I2C_MST_ODR_CONFIG, 0x04)

        print("Step 4: Set I2C Master clock to 400 kHz with STOP between reads")
        self.write_reg(I2C_MST_CTRL, 0x07 | BIT_I2C_MST_P_NSR)

        # Bank 0: Enable I2C Master
        self.set_bank(0)
        print("Step 5: Enable I2C Master mode")
        self.write_reg(USER_CTRL, BIT_I2C_MST_EN)
        time.sleep(0.1)

        print("✓ I2C Master configured")

    def read_mag_reg_via_slv4(self, mag_reg):
        """Read single register from AK09916 using Slave 4"""
        self.set_bank(3)

        # Configure Slave 4 for single read
        self.write_reg(I2C_SLV4_ADDR, BIT_I2C_READ | AK09916_I2C_ADDR)
        self.write_reg(I2C_SLV4_REG, mag_reg)
        self.write_reg(I2C_SLV4_CTRL, BIT_SLAVE_EN)

        # Wait for transaction to complete
        time.sleep(0.02)

        # Read result
        value = self.read_reg(I2C_SLV4_DI)
        return value

    def write_mag_reg_via_slv4(self, mag_reg, value):
        """Write single register to AK09916 using Slave 4"""
        self.set_bank(3)

        # Configure Slave 4 for single write
        self.write_reg(I2C_SLV4_ADDR, AK09916_I2C_ADDR)  # Write (no read bit)
        self.write_reg(I2C_SLV4_REG, mag_reg)
        self.write_reg(I2C_SLV4_DO, value)
        self.write_reg(I2C_SLV4_CTRL, BIT_SLAVE_EN)

        # Wait for transaction to complete
        time.sleep(0.02)

    def test_mag_whoami(self):
        """Test AK09916 WHO_AM_I registers"""
        print("\n=== Testing AK09916 WHO_AM_I ===")

        wia1 = self.read_mag_reg_via_slv4(AK09916_WIA1)
        wia2 = self.read_mag_reg_via_slv4(AK09916_WIA2)

        print(f"AK09916 WIA1: 0x{wia1:02X} (expect 0x48)")
        print(f"AK09916 WIA2: 0x{wia2:02X} (expect 0x09)")

        if wia1 == 0x48 and wia2 == 0x09:
            print("✅ SUCCESS! AK09916 is responding correctly!")
            return True
        elif wia1 == 0x00 and wia2 == 0x00:
            print("❌ FAILURE: AK09916 not responding (all zeros)")
            return False
        else:
            print(f"⚠️  UNEXPECTED: Got WIA1=0x{wia1:02X}, WIA2=0x{wia2:02X}")
            return False

    def scan_i2c_addresses(self):
        """Scan I2C addresses 0x0C-0x0F for any response"""
        print("\n=== Scanning I2C Master bus for devices ===")

        self.set_bank(3)
        found_any = False

        for addr in range(0x0C, 0x10):
            # Try to read register 0x00
            self.write_reg(I2C_SLV4_ADDR, BIT_I2C_READ | addr)
            self.write_reg(I2C_SLV4_REG, 0x00)
            self.write_reg(I2C_SLV4_CTRL, BIT_SLAVE_EN)
            time.sleep(0.05)

            val1 = self.read_reg(I2C_SLV4_DI)

            # Try register 0x01
            self.write_reg(I2C_SLV4_REG, 0x01)
            self.write_reg(I2C_SLV4_CTRL, BIT_SLAVE_EN)
            time.sleep(0.05)

            val2 = self.read_reg(I2C_SLV4_DI)

            if val1 != 0x00 or val2 != 0x00:
                print(f"  0x{addr:02X}: Found! Reg[0]=0x{val1:02X}, Reg[1]=0x{val2:02X}")
                found_any = True
            else:
                print(f"  0x{addr:02X}: No response")

        if not found_any:
            print("❌ No devices found on I2C Master bus")

        return found_any

    def test_mag_continuous_read(self):
        """Setup continuous magnetometer reading via Slave 0"""
        print("\n=== Testing Continuous Magnetometer Read ===")

        # Set AK09916 to continuous mode 3 (50 Hz)
        print("Setting AK09916 to continuous mode (50 Hz)...")
        self.write_mag_reg_via_slv4(AK09916_CNTL2, 0x06)
        time.sleep(0.1)

        # Verify it was set
        cntl2 = self.read_mag_reg_via_slv4(AK09916_CNTL2)
        print(f"AK09916 CNTL2 readback: 0x{cntl2:02X} (wrote 0x06)")

        # Configure Slave 0 for continuous reads
        self.set_bank(3)
        self.write_reg(I2C_SLV0_ADDR, BIT_I2C_READ | AK09916_I2C_ADDR)
        self.write_reg(I2C_SLV0_REG, AK09916_ST1)
        self.write_reg(I2C_SLV0_CTRL, BIT_SLAVE_EN | 9)  # Read 9 bytes (ST1 + 6 mag + ST2)

        time.sleep(0.2)

        # Read data from EXT_SENS_DATA
        print("\nReading magnetometer data:")
        for i in range(5):
            self.set_bank(0)

            st1 = self.read_reg(EXT_SENS_DATA_00)
            m1_l = self.read_reg(EXT_SENS_DATA_00 + 1)
            m1_h = self.read_reg(EXT_SENS_DATA_00 + 2)
            m2_l = self.read_reg(EXT_SENS_DATA_00 + 3)
            m2_h = self.read_reg(EXT_SENS_DATA_00 + 4)
            m3_l = self.read_reg(EXT_SENS_DATA_00 + 5)
            m3_h = self.read_reg(EXT_SENS_DATA_00 + 6)
            st2 = self.read_reg(EXT_SENS_DATA_00 + 7)

            # Combine to 16-bit signed values
            m1 = (m1_h << 8) | m1_l
            m2 = (m2_h << 8) | m2_l
            m3 = (m3_h << 8) | m3_l

            if m1 > 32767: m1 -= 65536
            if m2 > 32767: m2 -= 65536
            if m3 > 32767: m3 -= 65536

            print(f"  [{i+1}] ST1=0x{st1:02X} M1={m1:6d} M2={m2:6d} M3={m3:6d} ST2=0x{st2:02X}")

            if st1 & 0x01:
                print(f"      ✅ Data ready bit set!")
            else:
                print(f"      ❌ Data NOT ready")

            time.sleep(0.5)


def main():
    print("=" * 50)
    print("ICM20948 Simple Magnetometer Test")
    print("=" * 50)

    try:
        imu = ICM20948(bus_num=I2C_BUS, address=ICM20948_ADDR)

        # Check ICM20948 is present
        if not imu.check_whoami():
            print("\n❌ ERROR: ICM20948 not found!")
            return 1

        # Initialize
        imu.init_basic()

        # Setup I2C Master
        imu.setup_mag_i2c_master()

        # Scan for devices
        imu.scan_i2c_addresses()

        # Test WHO_AM_I
        if not imu.test_mag_whoami():
            print("\n❌ MAGNETOMETER TEST FAILED")
            print("The AK09916 is not responding via I2C Master")
            return 1

        # Test continuous reading
        imu.test_mag_continuous_read()

        print("\n" + "=" * 50)
        print("✅ MAGNETOMETER TEST COMPLETE")
        print("=" * 50)
        return 0

    except Exception as e:
        print(f"\n❌ ERROR: {e}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == "__main__":
    sys.exit(main())
