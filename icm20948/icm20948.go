package icm20948

// Approach adapted from the InvenSense DMP 6.1 drivers
// Also referenced https://github.com/brianc118/ICM20948/blob/master/ICM20948.cpp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all" // Empty import needed to initialize embd library.
	_ "github.com/kidoman/embd/host/rpi" // Empty import needed to initialize embd library.
)

const (
	bufSize          = 250                // Size of buffer storing instantaneous sensor values
	scaleMagAK8963   = 9830.0 / 65536
	scaleMagAK09916  = 4912.0 / 32752     // AK09916: ±4912 µT range, 16-bit
	calDataLocation = "/etc/icm20948cal.json"
)

// MPUData contains all the values measured by an ICM20948.
type MPUData struct {
	G1, G2, G3        float64
	A1, A2, A3        float64
	M1, M2, M3        float64
	Temp              float64
	GAError, MagError error
	N, NM             int
	T, TM             time.Time
	DT, DTM           time.Duration
}

type mpuCalData struct {
	A01, A02, A03    float64 // Accelerometer hardware bias
	G01, G02, G03    float64 // Gyro hardware bias
	M01, M02, M03    float64 // Magnetometer hardware bias
	Ms11, Ms12, Ms13 float64 // Magnetometer rescaling matrix
	Ms21, Ms22, Ms23 float64 // (Only diagonal is used currently)
	Ms31, Ms32, Ms33 float64
}

func (d *mpuCalData) reset() {
	d.Ms11 = 1
	d.Ms22 = 1
	d.Ms33 = 1
}

func (d *mpuCalData) save() {
	fd, err := os.OpenFile(calDataLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		log.Printf("ICM20948: Error saving calibration data to %s: %s", calDataLocation, err.Error())
		return
	}
	defer fd.Close()
	calData, err := json.Marshal(d)
	if err != nil {
		log.Printf("ICM20948: Error marshaling calibration data: %s", err)
		return
	}
	fd.Write(calData)
}

func (d *mpuCalData) load() (err error) {
	//d.M01 = 1638.0
	//d.M02 = -589.0
	//d.M03 = -2153.0
	//d.Ms11 = 0.00031969309462915601
	//d.Ms22 = 0.00035149384885764499
	//d.Ms33 = 0.00028752156411730879
	//d.save()
	//return
	errstr := "ICM20948: Error reading calibration data from %s: %s"
	fd, rerr := os.Open(calDataLocation)
	if rerr != nil {
		err = fmt.Errorf(errstr, calDataLocation, rerr.Error())
		return
	}
	defer fd.Close()
	buf := make([]byte, 1024)
	count, rerr := fd.Read(buf)
	if rerr != nil {
		err = fmt.Errorf(errstr, calDataLocation, rerr.Error())
		return
	}
	rerr = json.Unmarshal(buf[0:count], d)
	if rerr != nil {
		err = fmt.Errorf(errstr, calDataLocation, rerr.Error())
		return
	}
	return
}

/*
ICM20948 represents an InvenSense ICM20948 9DoF chip.
All communication is via channels.
*/
type ICM20948 struct {
	i2cbus                embd.I2CBus
	scaleGyro, scaleAccel float64 // Max sensor reading for value 2**15-1
	sampleRate            int
	enableMag             bool
	mpuCalData
	mcal1, mcal2, mcal3 float64         // Hardware magnetometer calibration values, uT
	C                   <-chan *MPUData // Current instantaneous sensor values
	CAvg                <-chan *MPUData // Average sensor values (since CAvg last read)
	CBuf                <-chan *MPUData // Buffer of instantaneous sensor values
	cClose              chan bool       // Turn off MPU polling
}

/*
NewICM20948 creates a new ICM20948 object according to the supplied parameters.  If there is no ICM20948 available or there
is an error creating the object, an error is returned.
*/
func NewICM20948(i2cbus *embd.I2CBus, sensitivityGyro, sensitivityAccel, sampleRate int, enableMag bool, applyHWOffsets bool) (*ICM20948, error) {
	var mpu = new(ICM20948)
	if err := mpu.mpuCalData.load(); err != nil {
		mpu.mpuCalData.reset()
	}

	mpu.sampleRate = sampleRate
	mpu.enableMag = enableMag // Enable magnetometer based on parameter

	mpu.i2cbus = *i2cbus

	mpu.setRegBank(0)

	// Initialization of MPU
	// Reset device.
	if err := mpu.i2cWrite(ICMREG_PWR_MGMT_1, BIT_H_RESET); err != nil {
		return nil, errors.New("Error resetting ICM20948")
	}

	// Wake up chip.
	time.Sleep(100 * time.Millisecond)
	// CLKSEL = 1.
	// From ICM-20948 register map (PWR_MGMT_1):
	//  "NOTE: CLKSEL[2:0] should be set to 1~5 to achieve full gyroscope performance."
	if err := mpu.i2cWrite(ICMREG_PWR_MGMT_1, 0x01); err != nil {
		return nil, errors.New("Error waking ICM20948")
	}

	// CRITICAL: Enable gyro and accel immediately (PWR_MGMT_2 = 0x00)
	// This MUST be done early for I2C Master to work!
	// Python Sequence 2 (which works) does this right after PWR_MGMT_1
	if err := mpu.i2cWrite(ICMREG_PWR_MGMT_2, 0x00); err != nil {
		return nil, errors.New("Error enabling gyro/accel in PWR_MGMT_2")
	}
	time.Sleep(50 * time.Millisecond) // Give sensors time to start
	log.Println("ICM20948: Gyro and Accel powered on early (PWR_MGMT_2=0x00)")

	// Note: inv_mpu.c sets some registers here to allocate 1kB to the FIFO buffer and 3kB to the DMP.
	// It doesn't seem to be supported in the 1.6 version of the register map and we're not using FIFO anyway,
	// so we skip this.
	// Don't let FIFO overwrite DMP data
	//if err := mpu.i2cWrite(ICMREG_ACCEL_CONFIG_2, BIT_FIFO_SIZE_1024|0x8); err != nil {
	//	return nil, errors.New("Error setting up ICM20948")
	//}

	// Set Gyro and Accel sensitivities
	if err := mpu.SetGyroSensitivity(sensitivityGyro); err != nil {
		log.Println(err)
	}

	if err := mpu.SetAccelSensitivity(sensitivityAccel); err != nil {
		log.Println(err)
	}

	sampRate := byte(1125/mpu.sampleRate - 1)
	// Default: Set Gyro LPF to half of sample rate
	if err := mpu.SetGyroLPF(sampRate >> 1); err != nil {
		return nil, err
	}

	// Default: Set Accel LPF to half of sample rate
	if err := mpu.SetAccelLPF(sampRate >> 1); err != nil {
		return nil, err
	}

	// Set sample rate to chosen
	if err := mpu.SetGyroSampleRate(sampRate); err != nil {
		return nil, err
	}

	if err := mpu.SetAccelSampleRate(sampRate); err != nil {
		return nil, err
	}

	// Turn off FIFO buffer. Not necessary - default off.

	// Turn off interrupts. Not necessary - default off.

	// Set up magnetometer (AK09916)
	if mpu.enableMag {
		log.Println("ICM20948: Initializing AK09916 magnetometer...")
		log.Println("ICM20948: PWR_MGMT_2 already set to 0x00 during initialization")

		// CRITICAL: Disable I2C bypass mode FIRST (before any I2C master config)
		// This is essential - bypass mode routes aux I2C to external pins, preventing I2C master from working
		if err := mpu.setRegBank(0); err != nil {
			return nil, errors.New("Error setting register bank 0 for bypass disable")
		}

		// Write 0x00 to INT_PIN_CFG to ensure bypass mode is OFF
		if err := mpu.i2cWrite(ICMREG_INT_PIN_CFG, 0x00); err != nil {
			return nil, errors.New("Error disabling I2C bypass mode")
		}
		log.Println("ICM20948: I2C bypass mode explicitly disabled (INT_PIN_CFG=0x00)")
		time.Sleep(10 * time.Millisecond)

		// SECOND: Ensure I2C Master is NOT in duty-cycle mode
		if err := mpu.setRegBank(0); err != nil {
			return nil, errors.New("Error setting register bank 0")
		}

		// Read LP_CONFIG and ensure I2C_MST_CYCLE is 0 (continuous, not duty-cycle)
		lpConfig, err := mpu.i2cRead(ICMREG_LP_CONFIG)
		if err != nil {
			log.Printf("ICM20948: Warning: Could not read LP_CONFIG: %v\n", err)
		} else {
			log.Printf("ICM20948: LP_CONFIG = 0x%02X\n", lpConfig)
			// Clear I2C_MST_CYCLE bit (bit 6) to ensure continuous operation
			if (lpConfig & 0x40) != 0 {
				lpConfig &= ^byte(0x40)
				mpu.i2cWrite(ICMREG_LP_CONFIG, lpConfig)
				log.Println("ICM20948: Disabled I2C_MST_CYCLE (enabled continuous mode)")
			}
		}

		// Switch to register bank 3 for I2C master configuration
		// IMPORTANT: Configure BEFORE enabling!
		if err := mpu.setRegBank(3); err != nil {
			return nil, errors.New("Error setting register bank 3")
		}

		// Set I2C Master ODR (Output Data Rate) to 100 Hz (0x09)
		// This determines how often the I2C master reads from slaves
		// Values: 0=1.1kHz, 1=500Hz, 2=333Hz, ..., 9=100Hz, 10=90.9Hz, 11=83.3Hz
		if err := mpu.i2cWrite(ICMREG_I2C_MST_ODR_CONFIG, 0x04); err != nil {
			return nil, errors.New("Error setting I2C master ODR")
		}
		log.Println("ICM20948: I2C master ODR set to 200 Hz")

		// Set I2C master clock to 400 kHz with STOP between reads
		// BIT_I2C_MST_P_NSR: Issues STOP condition between reads (critical for AK09916)
		if err := mpu.i2cWrite(ICMREG_I2C_MST_CTRL, 0x07|BIT_I2C_MST_P_NSR); err != nil {
			return nil, errors.New("Error setting up I2C master clock")
		}
		log.Println("ICM20948: I2C master clock set to 400 kHz with STOP between reads")

		// Configure I2C Slave 0 to read from AK09916
		// Set slave 0 address to AK09916 with read bit
		if err := mpu.i2cWrite(ICMREG_I2C_SLV0_ADDR, BIT_I2C_READ|AK09916_I2C_ADDR); err != nil {
			return nil, errors.New("Error setting up AK09916 slave address")
		}

		// Start reading from ST1 register
		if err := mpu.i2cWrite(ICMREG_I2C_SLV0_REG, AK09916_ST1); err != nil {
			return nil, errors.New("Error setting up AK09916 read register")
		}

		// Enable 9-byte reads on slave 0 (ST1 + 6 bytes mag data + ST2 + 1 reserved)
		if err := mpu.i2cWrite(ICMREG_I2C_SLV0_CTRL, BIT_SLAVE_EN|9); err != nil {
			return nil, errors.New("Error setting up AK09916 read control")
		}

		// IMPORTANT: We DON'T use Slave 1 for continuous writes
		// Instead, we'll use direct register writes after enabling I2C master

		// Set continuous measurement mode based on sample rate
		var magMode byte
		if mpu.sampleRate >= 100 {
			magMode = AK09916_MODE_CONT4 // 100 Hz
		} else if mpu.sampleRate >= 50 {
			magMode = AK09916_MODE_CONT3 // 50 Hz
		} else if mpu.sampleRate >= 20 {
			magMode = AK09916_MODE_CONT2 // 20 Hz
		} else {
			magMode = AK09916_MODE_CONT1 // 10 Hz
		}

		log.Printf("ICM20948: Will set AK09916 to continuous mode 0x%02X (sample rate: %d Hz)\n", magMode, mpu.sampleRate)

		// Set magnetometer hardware calibration values (AK09916 doesn't have sensitivity adjustment like AK8963)
		// Using default scale factor
		mpu.mpuCalData.M01 = scaleMagAK09916
		mpu.mpuCalData.M02 = scaleMagAK09916
		mpu.mpuCalData.M03 = scaleMagAK09916

		// Switch back to register bank 0
		if err := mpu.setRegBank(0); err != nil {
			return nil, errors.New("Error setting register bank 0")
		}

		// NOW enable I2C master mode (after all configuration is done!)
		// Enable I2C Master but keep I2C Slave enabled (Stratux needs to talk to us!)
		if err := mpu.i2cWrite(ICMREG_USER_CTRL, BIT_I2C_MST_EN); err != nil {
			return nil, errors.New("Error enabling I2C master mode")
		}
		log.Println("ICM20948: I2C master enabled (USER_CTRL=0x20)")

		time.Sleep(100 * time.Millisecond) // Give magnetometer time to initialize

		// Wait longer for I2C master to stabilize and start polling
		// I2C Master runs at 200Hz = 5ms per cycle, wait for at least 20 cycles
		time.Sleep(500 * time.Millisecond)

		// Verify AK09916 is responding by reading WHO_AM_I registers
		log.Println("ICM20948: Verifying AK09916 communication...")

		// Try scanning I2C addresses 0x0C, 0x0D, 0x0E, 0x0F (common mag addresses)
		log.Println("ICM20948: Scanning for AK09916 on I2C master bus...")
		if err := mpu.setRegBank(3); err != nil {
			log.Printf("ICM20948: Warning: Could not switch to bank 3: %v\n", err)
		} else {
			for addr := byte(0x0C); addr <= 0x0F; addr++ {
				// Try to read WHO_AM_I from this address
				mpu.i2cWrite(ICMREG_I2C_SLV0_ADDR, BIT_I2C_READ|addr)
				mpu.i2cWrite(ICMREG_I2C_SLV0_REG, 0x00) // WIA1 register
				mpu.i2cWrite(ICMREG_I2C_SLV0_CTRL, BIT_SLAVE_EN|2)
				// Wait longer - multiple I2C master cycles (200Hz = 5ms/cycle)
				time.Sleep(50 * time.Millisecond)

				mpu.setRegBank(0)
				val1, _ := mpu.i2cRead(ICMREG_EXT_SENS_DATA_00)
				val2, _ := mpu.i2cRead(ICMREG_EXT_SENS_DATA_01)

				if val1 != 0x00 || val2 != 0x00 {
					log.Printf("ICM20948: Found device at 0x%02X: 0x%02X 0x%02X\n", addr, val1, val2)
				}
				mpu.setRegBank(3)
			}
		}

		// Now try the expected address 0x0C
		// Switch to bank 3 to configure slave 0 for WHO_AM_I read
		if err := mpu.setRegBank(3); err != nil {
			log.Printf("ICM20948 Warning: Could not switch to bank 3 for verification: %v\n", err)
		} else {
			// Configure slave 0 to read WIA1 and WIA2 (2 bytes)
			mpu.i2cWrite(ICMREG_I2C_SLV0_ADDR, BIT_I2C_READ|AK09916_I2C_ADDR)
			mpu.i2cWrite(ICMREG_I2C_SLV0_REG, AK09916_WIA1)
			mpu.i2cWrite(ICMREG_I2C_SLV0_CTRL, BIT_SLAVE_EN|2) // Read 2 bytes

			time.Sleep(10 * time.Millisecond)

			// Switch to bank 0 to read EXT_SENS_DATA
			mpu.setRegBank(0)
			wia1, _ := mpu.i2cRead(ICMREG_EXT_SENS_DATA_00)
			wia2, _ := mpu.i2cRead(ICMREG_EXT_SENS_DATA_01)

			log.Printf("ICM20948: AK09916 WHO_AM_I: WIA1=0x%02X (expect 0x48), WIA2=0x%02X (expect 0x09)\n", wia1, wia2)

			if wia1 != 0x48 || wia2 != 0x09 {
				log.Printf("ICM20948 ERROR: AK09916 not responding correctly!\n")
			}

			// FIRST: Try soft reset of AK09916 via CNTL3
			log.Println("ICM20948: Sending soft reset to AK09916 via CNTL3...")
			mpu.setRegBank(3)
			mpu.i2cWrite(ICMREG_I2C_SLV4_ADDR, AK09916_I2C_ADDR)
			mpu.i2cWrite(ICMREG_I2C_SLV4_REG, AK09916_CNTL3) // CNTL3 = 0x32
			mpu.i2cWrite(ICMREG_I2C_SLV4_DO, 0x01) // SRST bit
			mpu.i2cWrite(ICMREG_I2C_SLV4_CTRL, BIT_SLAVE_EN)
			time.Sleep(100 * time.Millisecond) // Wait for reset to complete

			// NOW write to AK09916 CNTL2 using Slave 4 (single transaction)
			log.Printf("ICM20948: Writing 0x%02X to AK09916 CNTL2 via Slave 4...\n", magMode)
			mpu.i2cWrite(ICMREG_I2C_SLV4_ADDR, AK09916_I2C_ADDR) // Write mode (no BIT_I2C_READ)
			mpu.i2cWrite(ICMREG_I2C_SLV4_REG, AK09916_CNTL2)
			mpu.i2cWrite(ICMREG_I2C_SLV4_DO, magMode)
			mpu.i2cWrite(ICMREG_I2C_SLV4_CTRL, BIT_SLAVE_EN) // Start single transaction

			// Wait for transaction to complete (check I2C_MST_STATUS or just wait)
			time.Sleep(20 * time.Millisecond)

			// Read back CNTL2 to verify mode was set
			mpu.i2cWrite(ICMREG_I2C_SLV0_ADDR, BIT_I2C_READ|AK09916_I2C_ADDR)
			mpu.i2cWrite(ICMREG_I2C_SLV0_REG, AK09916_CNTL2)
			mpu.i2cWrite(ICMREG_I2C_SLV0_CTRL, BIT_SLAVE_EN|1) // Read 1 byte

			time.Sleep(10 * time.Millisecond)

			mpu.setRegBank(0)
			cntl2, _ := mpu.i2cRead(ICMREG_EXT_SENS_DATA_00)
			log.Printf("ICM20948: AK09916 CNTL2 readback=0x%02X (expected 0x%02X)\n", cntl2, magMode)

			// Reconfigure slave 0 back to reading ST1+mag data
			mpu.setRegBank(3)
			mpu.i2cWrite(ICMREG_I2C_SLV0_ADDR, BIT_I2C_READ|AK09916_I2C_ADDR)
			mpu.i2cWrite(ICMREG_I2C_SLV0_REG, AK09916_ST1)
			mpu.i2cWrite(ICMREG_I2C_SLV0_CTRL, BIT_SLAVE_EN|9)
			mpu.setRegBank(0)

			log.Println("ICM20948: Slave 0 reconfigured for continuous ST1+mag data reading")
		}

		log.Println("ICM20948: AK09916 magnetometer initialization complete")
	}
	// Set clock source to PLL. Not necessary - default "auto select" (PLL when ready).

	if applyHWOffsets {
		if err := mpu.ReadAccelBias(sensitivityAccel); err != nil {
			return nil, err
		}
		if err := mpu.ReadGyroBias(sensitivityGyro); err != nil {
			return nil, err
		}
	}

	// Usually we don't want the automatic gyro bias compensation - it pollutes the gyro in a non-inertial frame.
	/*	if err := mpu.EnableGyroBiasCal(false); err != nil {
			return nil, err
		}
	*/
	go mpu.readSensors()

	// Give the IMU time to fully initialize and then clear out any bad values from the averages.
	time.Sleep(500 * time.Millisecond) // Make sure it's ready
	<-mpu.CAvg                         // Discard the first readings.

	return mpu, nil
}

// readSensors polls the gyro, accelerometer and magnetometer sensors as well as the die temperature.
// Communication is via channels.
func (mpu *ICM20948) readSensors() {
	var (
		g1, g2, g3, a1, a2, a3, m1, m2, m3, tmp int16   // Current values
		avg1, avg2, avg3, ava1, ava2, ava3, avtmp   float64 // Accumulators for averages
		avm1, avm2, avm3                            int32
		n, nm                                       float64
		notReadyCount                               int // Counter for magnetometer not ready
		gaError, magError                           error
		t0, t, t0m, tm                              time.Time
		magSampleRate                               int
		curdata                                     *MPUData
	)

	//FIXME: Temporary (testing).
	//	mpu.setRegBank(2)
	//	mpu.i2cWrite(ICMREG_TEMP_CONFIG, 0x04)
	//	mpu.setRegBank(0)

	acRegMap := map[*int16]byte{
		&g1: ICMREG_GYRO_XOUT_H, &g2: ICMREG_GYRO_YOUT_H, &g3: ICMREG_GYRO_ZOUT_H,
		&a1: ICMREG_ACCEL_XOUT_H, &a2: ICMREG_ACCEL_YOUT_H, &a3: ICMREG_ACCEL_ZOUT_H,
		&tmp: ICMREG_TEMP_OUT_H,
	}
	magRegMap := map[*int16]byte{
		// AK09916 data starts at EXT_SENS_DATA_01 (after ST1 at _00)
		// HXL at _01, HXH at _02, HYL at _03, HYH at _04, HZL at _05, HZH at _06
		&m1: ICMREG_EXT_SENS_DATA_01, &m2: ICMREG_EXT_SENS_DATA_03, &m3: ICMREG_EXT_SENS_DATA_05,
	}

	if mpu.sampleRate > 100 {
		magSampleRate = 100
	} else {
		magSampleRate = mpu.sampleRate
	}

	cC := make(chan *MPUData)
	defer close(cC)
	mpu.C = cC
	cAvg := make(chan *MPUData)
	defer close(cAvg)
	mpu.CAvg = cAvg
	cBuf := make(chan *MPUData, bufSize)
	defer close(cBuf)
	mpu.CBuf = cBuf
	mpu.cClose = make(chan bool)
	defer close(mpu.cClose)

	clock := time.NewTicker(time.Duration(int(1125.0/float32(mpu.sampleRate)+0.5)) * time.Millisecond)
	//TODO westphae: use the clock to record actual time instead of a timer
	defer clock.Stop()

	clockMag := time.NewTicker(time.Duration(int(1125.0/float32(magSampleRate)+0.5)) * time.Millisecond)
	t0 = time.Now()
	t0m = time.Now()

	makeMPUData := func() *MPUData {
		mm1 := float64(m1)*mpu.mcal1 - mpu.M01
		mm2 := float64(m2)*mpu.mcal2 - mpu.M02
		mm3 := float64(m3)*mpu.mcal3 - mpu.M03
		//		fmt.Printf("a1=%d,a2=%d,a3=%d\n", a1, a2, a3)
		d := MPUData{
			G1:      (float64(g1) - mpu.G01) * mpu.scaleGyro,
			G2:      (float64(g2) - mpu.G02) * mpu.scaleGyro,
			G3:      (float64(g3) - mpu.G03) * mpu.scaleGyro,
			A1:      (float64(a1) - mpu.A01) * mpu.scaleAccel,
			A2:      (float64(a2) - mpu.A02) * mpu.scaleAccel,
			A3:      (float64(a3) - mpu.A03) * mpu.scaleAccel,
			M1:      mpu.Ms11*mm1 + mpu.Ms12*mm2 + mpu.Ms13*mm3,
			M2:      mpu.Ms21*mm1 + mpu.Ms22*mm2 + mpu.Ms23*mm3,
			M3:      mpu.Ms31*mm1 + mpu.Ms32*mm2 + mpu.Ms33*mm3,
			Temp:    float64(tmp)/333.87 + 21.0,
			GAError: gaError, MagError: magError,
			N: 1, NM: 1,
			T: t, TM: tm,
			DT: time.Duration(0), DTM: time.Duration(0),
		}
		if gaError != nil {
			d.N = 0
		}
		if magError != nil {
			d.NM = 0
		}
		return &d
	}

	makeAvgMPUData := func() *MPUData {
		mm1 := float64(avm1)*mpu.mcal1/nm - mpu.M01
		mm2 := float64(avm2)*mpu.mcal2/nm - mpu.M02
		mm3 := float64(avm3)*mpu.mcal3/nm - mpu.M03
		d := MPUData{}
		if n > 0.5 {
			d.G1 = (avg1/n - mpu.G01) * mpu.scaleGyro
			d.G2 = (avg2/n - mpu.G02) * mpu.scaleGyro
			d.G3 = (avg3/n - mpu.G03) * mpu.scaleGyro
			d.A1 = (ava1/n - mpu.A01) * mpu.scaleAccel
			d.A2 = (ava2/n - mpu.A02) * mpu.scaleAccel
			d.A3 = (ava3/n - mpu.A03) * mpu.scaleAccel
			d.Temp = (float64(avtmp)/n)/333.87 + 21.0
			d.N = int(n + 0.5)
			d.T = t
			d.DT = t.Sub(t0)
		} else {
			d.GAError = errors.New("ICM20948 Error: No new accel/gyro values")
		}
		if nm > 0 {
			d.M1 = mpu.Ms11*mm1 + mpu.Ms12*mm2 + mpu.Ms13*mm3
			d.M2 = mpu.Ms21*mm1 + mpu.Ms22*mm2 + mpu.Ms23*mm3
			d.M3 = mpu.Ms31*mm1 + mpu.Ms32*mm2 + mpu.Ms33*mm3
			d.NM = int(nm + 0.5)
			d.TM = tm
			d.DTM = t.Sub(t0m)
		} else {
			d.MagError = errors.New("ICM20948 Error: No new magnetometer values")
		}
		return &d
	}

	for {
		select {
		case t = <-clock.C: // Read accel/gyro data:
			for p, reg := range acRegMap {
				*p, gaError = mpu.i2cRead2(reg)
				if gaError != nil {
					log.Println("ICM20948 Warning: error reading gyro/accel")
				}
			}
			curdata = makeMPUData()
			// Update accumulated values and increment count of gyro/accel readings
			avg1 += float64(g1)
			avg2 += float64(g2)
			avg3 += float64(g3)
			ava1 += float64(a1)
			ava2 += float64(a2)
			ava3 += float64(a3)
			avtmp += float64(tmp)
			avm1 += int32(m1)
			avm2 += int32(m2)
			avm3 += int32(m3)
			n++
			select {
			case cBuf <- curdata: // We update the buffer every time we read a new value.
			default: // If buffer is full, remove oldest value and put in newest.
				<-cBuf
				cBuf <- curdata
			}
		case tm = <-clockMag.C: // Read magnetometer data:
			if mpu.enableMag {
				// Read magnetometer data from external sensor data registers
				var st1, st2 byte

				// Read ST1 status register
				st1, magError = mpu.i2cRead(ICMREG_EXT_SENS_DATA_00)
				if magError != nil {
					log.Println("ICM20948 Warning: error reading magnetometer ST1")
					continue
				}

				// Check if data is ready
				if (st1 & AK09916_ST1_DRDY) == 0 {
					notReadyCount++
					// Log first few times and then every 100th time
					if notReadyCount <= 5 || notReadyCount%100 == 0 {
						log.Printf("ICM20948: Magnetometer data not ready (count=%d, ST1=0x%02X)\n", notReadyCount, st1)
					}
					continue // Data not ready yet
				}

				// Read magnetometer data
				for p, reg := range magRegMap {
					*p, magError = mpu.i2cRead2(reg)
					if magError != nil {
						log.Println("ICM20948 Warning: error reading magnetometer data")
						continue
					}
				}

				// Read ST2 status register (at offset +8 from ST1)
				st2, magError = mpu.i2cRead(ICMREG_EXT_SENS_DATA_00 + 8)
				if magError != nil {
					log.Println("ICM20948 Warning: error reading magnetometer ST2")
					continue
				}

				// Check for data overflow
				if (st2 & AK09916_ST2_HOFL) != 0 {
					log.Println("ICM20948 mag data overflow")
					continue
				}

				// Update values and increment count of magnetometer readings
				avm1 += int32(m1)
				avm2 += int32(m2)
				avm3 += int32(m3)
				nm++

				// Log first successful read and every 100th read
				if nm == 1 || int(nm)%100 == 0 {
					log.Printf("ICM20948: Magnetometer read #%d: M1=%d, M2=%d, M3=%d (ST1=0x%02X, ST2=0x%02X)\n", int(nm), m1, m2, m3, st1, st2)
				}
			}
		case cC <- curdata: // Send the latest values
		case cAvg <- makeAvgMPUData(): // Send the averages
			avg1, avg2, avg3 = 0, 0, 0
			ava1, ava2, ava3 = 0, 0, 0
			avm1, avm2, avm3 = 0, 0, 0
			avtmp = 0
			n, nm = 0, 0
			t0, t0m = t, tm
		case <-mpu.cClose: // Stop the goroutine, ease up on the CPU
			break
		}
	}
}

// CloseMPU stops the driver from reading the MPU.
//TODO westphae: need a way to start it going again!
func (mpu *ICM20948) CloseMPU() {
	// Nothing to do bitwise for the 9250?
	mpu.cClose <- true
}

// SetGyroSampleRate changes the sampling rate of the gyro on the MPU.
func (mpu *ICM20948) SetGyroSampleRate(rate byte) (err error) {
	// Gyro config registers on Bank 2.
	if errWrite := mpu.setRegBank(2); errWrite != nil {
		return errors.New("ICM20948 Error: change register bank.")
	}

	defer mpu.setRegBank(0)

	errWrite := mpu.i2cWrite(ICMREG_GYRO_SMPLRT_DIV, byte(rate)) // Set sample rate to chosen
	if errWrite != nil {
		err = fmt.Errorf("ICM20948 Error: Couldn't set sample rate: %s", errWrite.Error())
	}
	return
}

// SetAccelSampleRate changes the sampling rate of the accelerometer on the MPU.
func (mpu *ICM20948) SetAccelSampleRate(rate byte) (err error) {
	// Gyro config registers on Bank 2.
	if errWrite := mpu.setRegBank(2); errWrite != nil {
		return errors.New("ICM20948 Error: change register bank.")
	}

	defer mpu.setRegBank(0)

	errWrite := mpu.i2cWrite(ICMREG_ACCEL_SMPLRT_DIV_2, byte(rate)) // Set sample rate to chosen
	if errWrite != nil {
		err = fmt.Errorf("ICM20948 Error: Couldn't set sample rate: %s", errWrite.Error())
	}
	return
}

// SetGyroLPF sets the low pass filter for the gyro.
func (mpu *ICM20948) SetGyroLPF(rate byte) (err error) {
	var r byte

	// Gyro config registers on Bank 2.
	if errWrite := mpu.setRegBank(2); errWrite != nil {
		return errors.New("ICM20948 Error: change register bank.")
	}

	defer mpu.setRegBank(0)

	cfg, err := mpu.i2cRead(ICMREG_GYRO_CONFIG)
	if err != nil {
		return errors.New("ICM20948 Error: SetGyroLPF error reading chip")
	}

	switch {
	case rate >= 197:
		r = BITS_DLPF_GYRO_CFG_197HZ
	case rate >= 152:
		r = BITS_DLPF_GYRO_CFG_152HZ
	case rate >= 120:
		r = BITS_DLPF_GYRO_CFG_120HZ
	case rate >= 51:
		r = BITS_DLPF_GYRO_CFG_51HZ
	case rate >= 24:
		r = BITS_DLPF_GYRO_CFG_24HZ
	case rate >= 12:
		r = BITS_DLPF_GYRO_CFG_12HZ
	default:
		r = BITS_DLPF_GYRO_CFG_6HZ
	}

	cfg |= 0x01
	cfg |= r

	errWrite := mpu.i2cWrite(ICMREG_GYRO_CONFIG, cfg)
	if errWrite != nil {
		err = fmt.Errorf("ICM20948 Error: couldn't set Gyro LPF: %s", errWrite.Error())
	}
	return
}

// SetAccelLPF sets the low pass filter for the accelerometer.
func (mpu *ICM20948) SetAccelLPF(rate byte) (err error) {
	var r byte

	// Accel config registers on Bank 2.
	if errWrite := mpu.setRegBank(2); errWrite != nil {
		return errors.New("ICM20948 Error: change register bank.")
	}

	defer mpu.setRegBank(0)

	cfg, err := mpu.i2cRead(ICMREG_ACCEL_CONFIG)
	if err != nil {
		return errors.New("ICM20948 Error: SetGyroLPF error reading chip")
	}

	switch {
	case rate >= 246:
		r = BITS_DLPF_ACCEL_CFG_246HZ
	case rate >= 111:
		r = BITS_DLPF_ACCEL_CFG_111HZ
	case rate >= 50:
		r = BITS_DLPF_ACCEL_CFG_50HZ
	case rate >= 24:
		r = BITS_DLPF_ACCEL_CFG_24HZ
	case rate >= 12:
		r = BITS_DLPF_ACCEL_CFG_12HZ
	default:
		r = BITS_DLPF_ACCEL_CFG_5HZ
	}

	cfg |= 0x01
	cfg |= r

	errWrite := mpu.i2cWrite(ICMREG_ACCEL_CONFIG, cfg)
	if errWrite != nil {
		err = fmt.Errorf("ICM20948 Error: couldn't set Accel LPF: %s", errWrite.Error())
	}
	return
}

// EnableGyroBiasCal enables or disables motion bias compensation for the gyro.
// For flying we generally do not want this!
func (mpu *ICM20948) EnableGyroBiasCal(enable bool) error {
	enableRegs := []byte{0xb8, 0xaa, 0xb3, 0x8d, 0xb4, 0x98, 0x0d, 0x35, 0x5d}
	disableRegs := []byte{0xb8, 0xaa, 0xaa, 0xaa, 0xb0, 0x88, 0xc3, 0xc5, 0xc7}

	if enable {
		if err := mpu.memWrite(CFG_MOTION_BIAS, &enableRegs); err != nil {
			return errors.New("Unable to enable motion bias compensation")
		}
	} else {
		if err := mpu.memWrite(CFG_MOTION_BIAS, &disableRegs); err != nil {
			return errors.New("Unable to disable motion bias compensation")
		}
	}

	return nil
}

// SampleRate returns the current sample rate of the ICM20948, in Hz.
func (mpu *ICM20948) SampleRate() int {
	return mpu.sampleRate
}

// MagEnabled returns whether or not the magnetometer is being read.
func (mpu *ICM20948) MagEnabled() bool {
	return mpu.enableMag
}

// SetGyroSensitivity sets the gyro sensitivity of the ICM20948; it must be one of the following values:
// 250, 500, 1000, 2000 (all in deg/s).
func (mpu *ICM20948) SetGyroSensitivity(sensitivityGyro int) (err error) {
	var sensGyro byte

	// Gyro config registers on Bank 2.
	if errWrite := mpu.setRegBank(2); errWrite != nil {
		return errors.New("ICM20948 Error: change register bank.")
	}

	defer mpu.setRegBank(0)

	switch sensitivityGyro {
	case 2000:
		sensGyro = BITS_FS_2000DPS
		mpu.scaleGyro = 2000.0 / float64(math.MaxInt16)
	case 1000:
		sensGyro = BITS_FS_1000DPS
		mpu.scaleGyro = 1000.0 / float64(math.MaxInt16)
	case 500:
		sensGyro = BITS_FS_500DPS
		mpu.scaleGyro = 500.0 / float64(math.MaxInt16)
	case 250:
		sensGyro = BITS_FS_250DPS
		mpu.scaleGyro = 250.0 / float64(math.MaxInt16)
	default:
		err = fmt.Errorf("ICM20948 Error: %d is not a valid gyro sensitivity", sensitivityGyro)
	}

	if errWrite := mpu.i2cWrite(ICMREG_GYRO_CONFIG, sensGyro); errWrite != nil {
		err = errors.New("ICM20948 Error: couldn't set gyro sensitivity")
	}

	return
}

func (mpu *ICM20948) setRegBank(bank byte) error {
	return mpu.i2cWrite(ICMREG_BANK_SEL, bank<<4)
}

// SetAccelSensitivity sets the accelerometer sensitivity of the ICM20948; it must be one of the following values:
// 2, 4, 8, 16, all in G (gravity).
func (mpu *ICM20948) SetAccelSensitivity(sensitivityAccel int) error {
	var sensAccel byte

	// Accel config registers on Bank 2.
	if errWrite := mpu.setRegBank(2); errWrite != nil {
		return errors.New("ICM20948 Error: change register bank.")
	}

	defer mpu.setRegBank(0)

	switch sensitivityAccel {
	case 16:
		sensAccel = BITS_FS_16G
		mpu.scaleAccel = 16.0 / float64(math.MaxInt16)
	case 8:
		sensAccel = BITS_FS_8G
		mpu.scaleAccel = 8.0 / float64(math.MaxInt16)
	case 4:
		sensAccel = BITS_FS_4G
		mpu.scaleAccel = 4.0 / float64(math.MaxInt16)
	case 2:
		sensAccel = BITS_FS_2G
		mpu.scaleAccel = 2.0 / float64(math.MaxInt16)
	default:
		return fmt.Errorf("ICM20948 Error: %d is not a valid accel sensitivity", sensitivityAccel)
	}

	if errWrite := mpu.i2cWrite(ICMREG_ACCEL_CONFIG, sensAccel); errWrite != nil {
		return errors.New("ICM20948 Error: couldn't set accel sensitivity")
	}

	return nil
}

// ReadAccelBias reads the bias accelerometer value stored on the chip.
// These values are set at the factory.
func (mpu *ICM20948) ReadAccelBias(sensitivityAccel int) error {
	if errWrite := mpu.setRegBank(1); errWrite != nil {
		return errors.New("ICM20948 Error: change register bank.")
	}
	defer mpu.setRegBank(0)

	a0x, err := mpu.i2cRead2(ICMREG_XA_OFFSET_H)
	if err != nil {
		return errors.New("ICM20948 Error: ReadAccelBias error reading chip")
	}
	a0y, err := mpu.i2cRead2(ICMREG_YA_OFFSET_H)
	if err != nil {
		return errors.New("ICM20948 Error: ReadAccelBias error reading chip")
	}
	a0z, err := mpu.i2cRead2(ICMREG_ZA_OFFSET_H)
	if err != nil {
		return errors.New("ICM20948 Error: ReadAccelBias error reading chip")
	}

	switch sensitivityAccel {
	case 16:
		mpu.A01 = float64(a0x >> 1)
		mpu.A02 = float64(a0y >> 1)
		mpu.A03 = float64(a0z >> 1)
	case 8:
		mpu.A01 = float64(a0x)
		mpu.A02 = float64(a0y)
		mpu.A03 = float64(a0z)
	case 4:
		mpu.A01 = float64(a0x << 1)
		mpu.A02 = float64(a0y << 1)
		mpu.A03 = float64(a0z << 1)
	case 2:
		mpu.A01 = float64(a0x << 2)
		mpu.A02 = float64(a0y << 2)
		mpu.A03 = float64(a0z << 2)
	default:
		return fmt.Errorf("ICM20948 Error: %d is not a valid acceleration sensitivity", sensitivityAccel)
	}

	return nil
}

// ReadGyroBias reads the bias gyro value stored on the chip.
// These values are set at the factory.
func (mpu *ICM20948) ReadGyroBias(sensitivityGyro int) error {
	if errWrite := mpu.setRegBank(2); errWrite != nil {
		return errors.New("ICM20948 Error: change register bank.")
	}
	defer mpu.setRegBank(0)

	g0x, err := mpu.i2cRead2(ICMREG_XG_OFFS_USRH)
	if err != nil {
		return errors.New("ICM20948 Error: ReadGyroBias error reading chip")
	}
	g0y, err := mpu.i2cRead2(ICMREG_YG_OFFS_USRH)
	if err != nil {
		return errors.New("ICM20948 Error: ReadGyroBias error reading chip")
	}
	g0z, err := mpu.i2cRead2(ICMREG_ZG_OFFS_USRH)
	if err != nil {
		return errors.New("ICM20948 Error: ReadGyroBias error reading chip")
	}

	switch sensitivityGyro {
	case 2000:
		mpu.G01 = float64(g0x >> 1)
		mpu.G02 = float64(g0y >> 1)
		mpu.G03 = float64(g0z >> 1)
	case 1000:
		mpu.G01 = float64(g0x)
		mpu.G02 = float64(g0y)
		mpu.G03 = float64(g0z)
	case 500:
		mpu.G01 = float64(g0x << 1)
		mpu.G02 = float64(g0y << 1)
		mpu.G03 = float64(g0z << 1)
	case 250:
		mpu.G01 = float64(g0x << 2)
		mpu.G02 = float64(g0y << 2)
		mpu.G03 = float64(g0z << 2)
	default:
		return fmt.Errorf("ICM20948 Error: %d is not a valid gyro sensitivity", sensitivityGyro)
	}

	return nil
}

// ReadMagCalibration reads the magnetometer bias values stored on the chpi.
// These values are set at the factory.
func (mpu *ICM20948) ReadMagCalibration() error {
	// Enable bypass mode
	var tmp uint8
	var err error
	tmp, err = mpu.i2cRead(ICMREG_USER_CTRL)
	if err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	if err = mpu.i2cWrite(ICMREG_USER_CTRL, tmp & ^BIT_AUX_IF_EN); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	time.Sleep(3 * time.Millisecond)
	if err = mpu.i2cWrite(ICMREG_INT_PIN_CFG, BIT_BYPASS_EN); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}

	// Prepare for getting sensitivity data from AK8963
	//Set the I2C slave address of AK8963
	if err = mpu.i2cWrite(ICMREG_I2C_SLV0_ADDR, AK8963_I2C_ADDR); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	// Power down the AK8963
	if err = mpu.i2cWrite(ICMREG_I2C_SLV0_CTRL, AK8963_CNTL1); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	// Power down the AK8963
	if err = mpu.i2cWrite(ICMREG_I2C_SLV0_DO, AKM_POWER_DOWN); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	time.Sleep(time.Millisecond)
	// Fuse AK8963 ROM access
	if mpu.i2cWrite(ICMREG_I2C_SLV0_DO, AK8963_I2CDIS); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	time.Sleep(time.Millisecond)

	// Get sensitivity data from AK8963 fuse ROM
	mcal1, err := mpu.i2cRead(AK8963_ASAX)
	if err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	mcal2, err := mpu.i2cRead(AK8963_ASAY)
	if err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	mcal3, err := mpu.i2cRead(AK8963_ASAZ)
	if err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}

	mpu.mcal1 = float64(int16(mcal1)+128) / 256 * scaleMagAK8963
	mpu.mcal2 = float64(int16(mcal2)+128) / 256 * scaleMagAK8963
	mpu.mcal3 = float64(int16(mcal3)+128) / 256 * scaleMagAK8963

	// Clean up from getting sensitivity data from AK8963
	// Fuse AK8963 ROM access
	if err = mpu.i2cWrite(ICMREG_I2C_SLV0_DO, AK8963_I2CDIS); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	time.Sleep(time.Millisecond)

	// Disable bypass mode now that we're done getting sensitivity data
	tmp, err = mpu.i2cRead(ICMREG_USER_CTRL)
	if err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	if err = mpu.i2cWrite(ICMREG_USER_CTRL, tmp|BIT_AUX_IF_EN); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	time.Sleep(3 * time.Millisecond)
	if err = mpu.i2cWrite(ICMREG_INT_PIN_CFG, 0x00); err != nil {
		return errors.New("ReadMagCalibration error reading chip")
	}
	time.Sleep(3 * time.Millisecond)

	return nil
}

func (mpu *ICM20948) i2cWrite(register, value byte) (err error) {

	if errWrite := mpu.i2cbus.WriteByteToReg(MPU_ADDRESS, register, value); errWrite != nil {
		err = fmt.Errorf("ICM20948 Error writing %X to %X: %s\n",
			value, register, errWrite.Error())
	} else {
		time.Sleep(time.Millisecond)
	}
	return
}

func (mpu *ICM20948) i2cRead(register byte) (value uint8, err error) {
	value, errWrite := mpu.i2cbus.ReadByteFromReg(MPU_ADDRESS, register)
	if errWrite != nil {
		err = fmt.Errorf("i2cRead error: %s", errWrite.Error())
	}
	return
}

func (mpu *ICM20948) i2cRead2(register byte) (value int16, err error) {

	v, errWrite := mpu.i2cbus.ReadWordFromReg(MPU_ADDRESS, register)
	if errWrite != nil {
		err = fmt.Errorf("ICM20948 Error reading %x: %s\n", register, errWrite.Error())
	} else {
		value = int16(v)
	}
	return
}

func (mpu *ICM20948) memWrite(addr uint16, data *[]byte) error {
	var err error
	var tmp = make([]byte, 2)

	tmp[0] = byte(addr >> 8)
	tmp[1] = byte(addr & 0xFF)

	// Check memory bank boundaries
	if tmp[1]+byte(len(*data)) > MPU_BANK_SIZE {
		return errors.New("Bad address: writing outside of memory bank boundaries")
	}

	err = mpu.i2cbus.WriteToReg(MPU_ADDRESS, ICMREG_BANK_SEL, tmp)
	if err != nil {
		return fmt.Errorf("ICM20948 Error selecting memory bank: %s\n", err.Error())
	}

	err = mpu.i2cbus.WriteToReg(MPU_ADDRESS, ICMREG_MEM_R_W, *data)
	if err != nil {
		return fmt.Errorf("ICM20948 Error writing to the memory bank: %s\n", err.Error())
	}

	return nil
}
