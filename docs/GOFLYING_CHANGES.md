# lordvampire/goflying Änderungen für ICM20948 Magnetometer-Fix

**Repository:** https://github.com/lordvampire/goflying
**Branch:** stratux_master

## Kritischer Bug-Fix

**Datei:** `icm20948/dmp_constants.go`

**Problem:** Falsche Register-Adresse für USER_CTRL (MPU-9250 statt ICM-20948)

### Änderung (Zeile 74):

**VORHER:**
```go
ICMREG_USER_CTRL          = 0x6A
```

**NACHHER:**
```go
ICMREG_USER_CTRL          = 0x03 // Bank 0: CRITICAL FIX! Was 0x6A (MPU-9250), should be 0x03 (ICM-20948)
```

## Optionaler Patch (bereits im lordvampire/goflying vorhanden)

**Datei:** `icm20948/icm20948.go`

**Zeile ~1149** - Nach dem SLV4_DONE Check, vor dem Lesen von SLV4_DI:

```go
if (status & 0x40) != 0 { // SLV4_DONE
    // CRITICAL FIX: Wait before reading SLV4_DI (register 0x17 shared with I2C_MST_STATUS)
    time.Sleep(5 * time.Millisecond)
    wia1, _ = mpu.i2cRead(ICMREG_I2C_SLV4_DI)
    log.Println("")
    log.Printf("  [SUCCESS] SLV4_DONE after %dms", (i+1)*10)
```

## Ergebnis nach dem Fix

- ✅ I2C Master wird korrekt aktiviert (USER_CTRL Register 0x03)
- ✅ Magnetometer (AK09916) wird erfolgreich initialisiert
- ✅ Kontinuierliche Mag-Daten werden geliefert
- ✅ AHRS kann MagHeading berechnen

## Test-Kommando

Nach dem Update von goflying:
```bash
cd /home/pi/test-embd
go mod edit -replace=github.com/b3nn0/goflying=github.com/lordvampire/goflying@latest
go mod tidy
go build test-goflying-only.go
./test-goflying-only
```

Erwartet: "✓ MAGNETOMETER WORKS!"

## Aktueller Status (2025-10-24)

✅ **Der Fix ist bereits im GitHub Repository!**
- Repository: https://github.com/lordvampire/goflying
- Branch: stratux_master
- Commit: e7543f71af1e29d95d062be02310559aa6488156
- USER_CTRL = 0x03 ✓
