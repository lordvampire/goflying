package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
)

const (
	ICM20948_ADDR = 0x68
	AK09916_ADDR  = 0x0C

	// Bank 0 Registers
	REG_BANK_SEL  = 0x7F
	WHO_AM_I      = 0x00
	PWR_MGMT_1    = 0x06
	PWR_MGMT_2    = 0x07
	USER_CTRL     = 0x03
	INT_PIN_CFG   = 0x0F
	LP_CONFIG     = 0x05

	// Bank 2 Registers
	GYRO_CONFIG_1 = 0x01
	ACCEL_CONFIG  = 0x14

	// Bank 3 Registers
	I2C_MST_ODR_CONFIG = 0x00
	I2C_MST_CTRL       = 0x01
	I2C_MST_STATUS     = 0x17
	I2C_SLV0_ADDR      = 0x03
	I2C_SLV0_REG       = 0x04
	I2C_SLV0_CTRL      = 0x05
	I2C_SLV4_ADDR      = 0x13
	I2C_SLV4_REG       = 0x14
	I2C_SLV4_CTRL      = 0x15
	I2C_SLV4_DO        = 0x16
	I2C_SLV4_DI        = 0x17
)

var (
	bus         embd.I2CBus
	currentBank *int
)

func setBank(bank int) error {
	if currentBank == nil || *currentBank != bank {
		oldBank := "None"
		if currentBank != nil {
			oldBank = fmt.Sprintf("%d", *currentBank)
		}
		fmt.Printf("  [BANK] Switching from Bank %s to Bank %d\n", oldBank, bank)

		if err := bus.WriteByteToReg(ICM20948_ADDR, REG_BANK_SEL, byte(bank<<4)); err != nil {
			return err
		}
		currentBank = &bank
		time.Sleep(1 * time.Millisecond)
	}
	return nil
}

func readReg(reg byte) (byte, error) {
	val, err := bus.ReadByteFromReg(ICM20948_ADDR, reg)
	return val, err
}

func writeReg(reg, value byte) error {
	return bus.WriteByteToReg(ICM20948_ADDR, reg, value)
}

func writeAndVerify(reg, value byte, name string) error {
	if err := writeReg(reg, value); err != nil {
		return err
	}
	time.Sleep(5 * time.Millisecond)

	readback, err := readReg(reg)
	if err != nil {
		return err
	}

	if readback == value {
		fmt.Printf("  [WRITE] %s (0x%02X) = 0x%02X ✓ (readback: 0x%02X)\n", name, reg, value, readback)
	} else {
		fmt.Printf("  [WRITE] %s (0x%02X) = 0x%02X ✗ ERROR! (readback: 0x%02X)\n", name, reg, value, readback)
	}
	return nil
}

func decodeStatus(status byte) string {
	bits := ""
	if status&0x80 != 0 {
		bits += "PASS_THROUGH | "
	}
	if status&0x40 != 0 {
		bits += "SLV4_DONE | "
	}
	if status&0x10 != 0 {
		bits += "LOST_ARB | "
	}
	if status&0x08 != 0 {
		bits += "SLV4_NACK | "
	}
	if status&0x04 != 0 {
		bits += "SLV3_NACK | "
	}
	if status&0x02 != 0 {
		bits += "SLV2_NACK | "
	}
	if status&0x01 != 0 {
		bits += "SLV1_NACK | "
	}

	if bits == "" {
		return "NONE"
	}
	return bits[:len(bits)-3] // Remove trailing " | "
}

func dumpBank0Regs() error {
	if err := setBank(0); err != nil {
		return err
	}

	fmt.Println("  [DUMP] Bank 0:")
	who, _ := readReg(WHO_AM_I)
	pwr1, _ := readReg(PWR_MGMT_1)
	pwr2, _ := readReg(PWR_MGMT_2)
	user, _ := readReg(USER_CTRL)
	intCfg, _ := readReg(INT_PIN_CFG)

	fmt.Printf("    WHO_AM_I    (0x00) = 0x%02X\n", who)
	fmt.Printf("    PWR_MGMT_1  (0x06) = 0x%02X\n", pwr1)
	fmt.Printf("    PWR_MGMT_2  (0x07) = 0x%02X\n", pwr2)
	fmt.Printf("    USER_CTRL   (0x03) = 0x%02X (I2C_MST_EN=%v)\n", user, (user&0x20) != 0)
	fmt.Printf("    INT_PIN_CFG (0x0F) = 0x%02X (BYPASS=%v)\n", intCfg, (intCfg&0x02) != 0)

	return nil
}

func dumpBank2Regs() error {
	if err := setBank(2); err != nil {
		return err
	}

	fmt.Println("  [DUMP] Bank 2:")
	gyro, _ := readReg(GYRO_CONFIG_1)
	accel, _ := readReg(ACCEL_CONFIG)

	fmt.Printf("    GYRO_CONFIG_1 (0x01) = 0x%02X\n", gyro)
	fmt.Printf("    ACCEL_CONFIG  (0x14) = 0x%02X\n", accel)

	return nil
}

func dumpBank3Regs() error {
	if err := setBank(3); err != nil {
		return err
	}

	fmt.Println("  [DUMP] Bank 3:")
	odr, _ := readReg(I2C_MST_ODR_CONFIG)
	ctrl, _ := readReg(I2C_MST_CTRL)
	status, _ := readReg(I2C_MST_STATUS)
	slv0Addr, _ := readReg(I2C_SLV0_ADDR)
	slv0Reg, _ := readReg(I2C_SLV0_REG)
	slv0Ctrl, _ := readReg(I2C_SLV0_CTRL)
	slv4Addr, _ := readReg(I2C_SLV4_ADDR)
	slv4Reg, _ := readReg(I2C_SLV4_REG)
	slv4Ctrl, _ := readReg(I2C_SLV4_CTRL)

	fmt.Printf("    I2C_MST_ODR_CONFIG (0x00) = 0x%02X\n", odr)
	fmt.Printf("    I2C_MST_CTRL       (0x01) = 0x%02X\n", ctrl)
	fmt.Printf("    I2C_MST_STATUS     (0x17) = 0x%02X [%s]\n", status, decodeStatus(status))
	fmt.Printf("    I2C_SLV0_ADDR      (0x03) = 0x%02X\n", slv0Addr)
	fmt.Printf("    I2C_SLV0_REG       (0x04) = 0x%02X\n", slv0Reg)
	fmt.Printf("    I2C_SLV0_CTRL      (0x05) = 0x%02X\n", slv0Ctrl)
	fmt.Printf("    I2C_SLV4_ADDR      (0x13) = 0x%02X\n", slv4Addr)
	fmt.Printf("    I2C_SLV4_REG       (0x14) = 0x%02X\n", slv4Reg)
	fmt.Printf("    I2C_SLV4_CTRL      (0x15) = 0x%02X\n", slv4Ctrl)

	return nil
}

func main() {
	fmt.Println("================================================================================")
	fmt.Println("ICM20948 Full Sensor Test - Go Implementation")
	fmt.Println("Direct translation of test-all-sensors-debug.py")
	fmt.Println("================================================================================")

	// Initialize I2C
	if err := embd.InitI2C(); err != nil {
		log.Fatal("Failed to initialize I2C:", err)
	}
	defer embd.CloseI2C()

	var err error
	bus = embd.NewI2CBus(1)

	// STEP 1: Reset ICM20948
	fmt.Println("\n" + "================================================================================")
	fmt.Println("STEP 1: Reset ICM20948")
	fmt.Println("================================================================================")

	setBank(0)
	writeAndVerify(PWR_MGMT_1, 0x80, "PWR_MGMT_1 (RESET)")
	fmt.Println("  [SLEEP] 100ms after reset")
	time.Sleep(100 * time.Millisecond)

	writeAndVerify(PWR_MGMT_1, 0x01, "PWR_MGMT_1 (WAKE + AUTO CLOCK)")
	fmt.Println("  [SLEEP] 10ms after wake")
	time.Sleep(10 * time.Millisecond)

	dumpBank0Regs()

	// STEP 2: Enable Gyro and Accel
	fmt.Println("\n" + "================================================================================")
	fmt.Println("STEP 2: Enable Gyro and Accel (PWR_MGMT_2 = 0x00)")
	fmt.Println("================================================================================")

	setBank(0)
	writeAndVerify(PWR_MGMT_2, 0x00, "PWR_MGMT_2 (ALL SENSORS ON)")
	fmt.Println("  [SLEEP] 50ms for sensors to start")
	time.Sleep(50 * time.Millisecond)

	dumpBank0Regs()

	// STEP 3: Configure Gyro and Accel
	fmt.Println("\n" + "================================================================================")
	fmt.Println("STEP 3: Configure Gyro and Accel sensitivities (Bank 2)")
	fmt.Println("================================================================================")

	setBank(2)
	writeAndVerify(GYRO_CONFIG_1, 0x06, "GYRO_CONFIG_1 (±1000 dps)")
	writeAndVerify(ACCEL_CONFIG, 0x04, "ACCEL_CONFIG (±8g)")

	dumpBank2Regs()

	// STEP 4: Disable I2C bypass
	fmt.Println("\n" + "================================================================================")
	fmt.Println("STEP 4: Disable I2C bypass mode")
	fmt.Println("================================================================================")

	setBank(0)
	writeAndVerify(INT_PIN_CFG, 0x00, "INT_PIN_CFG (BYPASS OFF)")
	fmt.Println("  [SLEEP] 10ms")
	time.Sleep(10 * time.Millisecond)

	dumpBank0Regs()

	// STEP 5: Configure I2C Master
	fmt.Println("\n" + "================================================================================")
	fmt.Println("STEP 5: Configure I2C Master (Bank 3)")
	fmt.Println("================================================================================")

	setBank(3)
	writeAndVerify(I2C_MST_ODR_CONFIG, 0x04, "I2C_MST_ODR_CONFIG (200 Hz)")
	writeAndVerify(I2C_MST_CTRL, 0x17, "I2C_MST_CTRL (400 kHz + STOP)")

	dumpBank3Regs()

	// STEP 6: Enable I2C Master
	fmt.Println("\n" + "================================================================================")
	fmt.Println("STEP 6: Enable I2C Master (USER_CTRL bit 5)")
	fmt.Println("================================================================================")

	setBank(0)
	writeAndVerify(USER_CTRL, 0x20, "USER_CTRL (I2C_MST_EN)")
	fmt.Println("  [SLEEP] 100ms")
	time.Sleep(100 * time.Millisecond)

	dumpBank0Regs()
	dumpBank3Regs()

	// STEP 7: Test Slave 4 communication
	fmt.Println("\n" + "================================================================================")
	fmt.Println("STEP 7: Test Slave 4 communication with AK09916")
	fmt.Println("================================================================================")

	setBank(3)
	fmt.Println("  [WRITE] I2C_SLV4_ADDR (0x13) = 0x8C (READ from 0x0C)")
	writeReg(I2C_SLV4_ADDR, 0x80|AK09916_ADDR)

	fmt.Println("  [WRITE] I2C_SLV4_REG  (0x14) = 0x00 (WIA1)")
	writeReg(I2C_SLV4_REG, 0x00)

	fmt.Println("  [WRITE] I2C_SLV4_CTRL (0x15) = 0x80 (ENABLE)")
	writeReg(I2C_SLV4_CTRL, 0x80)

	fmt.Println("\n  [DUMP] Slave 4 config readback:")
	slv4Addr, _ := readReg(I2C_SLV4_ADDR)
	slv4Reg, _ := readReg(I2C_SLV4_REG)
	slv4Ctrl, _ := readReg(I2C_SLV4_CTRL)
	fmt.Printf("    I2C_SLV4_ADDR = 0x%02X (expect 0x8C)\n", slv4Addr)
	fmt.Printf("    I2C_SLV4_REG  = 0x%02X (expect 0x00)\n", slv4Reg)
	fmt.Printf("    I2C_SLV4_CTRL = 0x%02X (expect 0x80)\n", slv4Ctrl)

	// Poll for SLV4_DONE
	fmt.Println("\n  [POLL] Waiting for SLV4_DONE bit (0x40 in I2C_MST_STATUS)...")
	wia1 := byte(0x00)
	success := false

	for i := 0; i < 50; i++ {
		time.Sleep(10 * time.Millisecond)
		status, _ := readReg(I2C_MST_STATUS)

		if i < 5 || i%10 == 0 || i == 49 {
			fmt.Printf("    Poll %2d: I2C_MST_STATUS=0x%02X [%s]\n", i+1, status, decodeStatus(status))
		}

		if (status & 0x40) != 0 { // SLV4_DONE
			wia1, _ = readReg(I2C_SLV4_DI)
			fmt.Printf("\n  [SUCCESS] SLV4_DONE after %dms\n", (i+1)*10)
			fmt.Printf("  [READ] I2C_SLV4_DI (0x17) = 0x%02X\n", wia1)
			success = true
			break
		}
	}

	if success && wia1 == 0x48 {
		fmt.Printf("  ✓ AK09916 WHO_AM_I = 0x%02X (CORRECT!)\n", wia1)
	} else {
		fmt.Printf("  ✗ FAILED! WIA1 = 0x%02X (expect 0x48)\n", wia1)
		dumpBank3Regs()
		return
	}

	fmt.Println("\n" + "================================================================================")
	fmt.Println("TEST COMPLETE - I2C Master is working!")
	fmt.Println("================================================================================")
}
