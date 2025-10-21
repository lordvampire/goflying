package main

import (
	"log"
	"time"

	"github.com/b3nn0/goflying/icm20948"
)

func main() {
	log.Println("===========================================")
	log.Println("ICM20948 Magnetometer Standalone Test")
	log.Println("===========================================")

	// I2C bus 1, same settings as Stratux uses
	i2cbus := 1
	gyroRange := 2000   // ±2000°/s
	accelRange := 16    // ±16g
	updateFreq := 50    // 50 Hz
	enableMag := true   // Enable magnetometer
	enableDMP := false  // Disable DMP for simpler testing

	log.Printf("Initializing ICM20948 on I2C bus %d...\n", i2cbus)
	log.Printf("Settings: Gyro=±%d°/s, Accel=±%dg, Freq=%dHz, Mag=%v, DMP=%v\n",
		gyroRange, accelRange, updateFreq, enableMag, enableDMP)

	// Initialize the sensor
	mpu, err := icm20948.NewICM20948(i2cbus, gyroRange, accelRange, updateFreq, enableMag, enableDMP)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize ICM20948: %v\n", err)
	}

	log.Println("ICM20948 initialized successfully!")
	log.Println("===========================================")
	log.Println("Starting magnetometer data readout...")
	log.Println("Press Ctrl+C to exit")
	log.Println("===========================================")

	// Read data continuously
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	readCount := 0
	for range ticker.C {
		readCount++

		// Read sensor data
		err := mpu.ReadSensor()
		if err != nil {
			log.Printf("[%04d] ERROR reading sensor: %v\n", readCount, err)
			continue
		}

		// Get magnetometer data
		m1, m2, m3 := mpu.Magnetometer()

		// Get gyro and accel for comparison (to verify sensor is working)
		g1, g2, g3 := mpu.Gyro()
		a1, a2, a3 := mpu.Accel()

		// Print data
		log.Printf("[%04d] Gyro: X=%7.2f Y=%7.2f Z=%7.2f °/s | Accel: X=%6.3f Y=%6.3f Z=%6.3f g\n",
			readCount, g1, g2, g3, a1, a2, a3)
		log.Printf("[%04d] **MAG: X=%7.2f Y=%7.2f Z=%7.2f µT**\n",
			readCount, m1, m2, m3)

		if m1 == 0 && m2 == 0 && m3 == 0 {
			log.Printf("[%04d] WARNING: Magnetometer returns all zeros!\n", readCount)
		}
	}
}
