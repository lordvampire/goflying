# ICM20948 Simple Magnetometer Test

**Einfachstes Test-Script** - direkt I2C Register lesen, keine komplizierten Dependencies!

## Was macht dieses Script?

Ein Python-Script, das **direkt die I2C-Register** des ICM20948 liest:

1. ✓ Initialisiert ICM20948
2. ✓ Konfiguriert I2C Master (genau wie Go-Code)
3. ✓ Scannt nach AK09916 auf I2C Master Bus
4. ✓ Liest WHO_AM_I (0x48, 0x09)
5. ✓ Testet continuous magnetometer reads

**Zeigt sofort ob der AK09916 antwortet!**

## Schnellstart

### Kopiere zum Pi:

```bash
scp simple-mag-test.py root@<pi-ip>:/root/
```

### Ausführen auf dem Pi:

```bash
ssh root@<pi-ip>
sudo python3 /root/simple-mag-test.py
```

**Das war's!** Keine Kompilierung, keine Go-Dependencies.

## Voraussetzungen

Nur Python3 mit smbus:

```bash
# Falls nicht installiert:
sudo apt install python3-smbus
```

## Erwartete Ausgabe

### ✅ Wenn es funktioniert:

```
==================================================
ICM20948 Simple Magnetometer Test
==================================================
ICM20948: Opened I2C bus 1, address 0x68
ICM20948: WHO_AM_I = 0xEA (expect 0xEA)

=== ICM20948 Basic Init ===
✓ Device woken up

=== Setting up I2C Master for AK09916 ===
Step 1: Disable I2C bypass mode
Step 2: Disable I2C_MST_CYCLE (enable continuous mode)
  LP_CONFIG = 0x40
  ✓ Cleared I2C_MST_CYCLE bit
Step 3: Set I2C Master ODR to 200 Hz
Step 4: Set I2C Master clock to 400 kHz with STOP between reads
Step 5: Enable I2C Master mode
✓ I2C Master configured

=== Scanning I2C Master bus for devices ===
  0x0C: Found! Reg[0]=0x48, Reg[1]=0x09
  0x0D: No response
  0x0E: No response
  0x0F: No response

=== Testing AK09916 WHO_AM_I ===
AK09916 WIA1: 0x48 (expect 0x48)
AK09916 WIA2: 0x09 (expect 0x09)
✅ SUCCESS! AK09916 is responding correctly!

=== Testing Continuous Magnetometer Read ===
Setting AK09916 to continuous mode (50 Hz)...
AK09916 CNTL2 readback: 0x06 (wrote 0x06)

Reading magnetometer data:
  [1] ST1=0x01 M1=   234 M2=  -123 M3=   456 ST2=0x00
      ✅ Data ready bit set!
  [2] ST1=0x01 M1=   235 M2=  -124 M3=   457 ST2=0x00
      ✅ Data ready bit set!
...

==================================================
✅ MAGNETOMETER TEST COMPLETE
==================================================
```

### ❌ Wenn es NICHT funktioniert:

```
=== Scanning I2C Master bus for devices ===
  0x0C: No response
  0x0D: No response
  0x0E: No response
  0x0F: No response
❌ No devices found on I2C Master bus

=== Testing AK09916 WHO_AM_I ===
AK09916 WIA1: 0x00 (expect 0x48)
AK09916 WIA2: 0x00 (expect 0x09)
❌ FAILURE: AK09916 not responding (all zeros)

❌ MAGNETOMETER TEST FAILED
```

## Was wird getestet?

Das Script setzt **exakt die gleichen Register** wie der Go-Code:

| Schritt | Register | Wert | Bedeutung |
|---------|----------|------|-----------|
| 1 | INT_PIN_CFG (0x0F) | 0x00 | Bypass mode AUS |
| 2 | LP_CONFIG (0x05) | Clear bit 6 | Duty-cycle AUS |
| 3 | I2C_MST_ODR_CONFIG (0x00) | 0x04 | 200 Hz |
| 4 | I2C_MST_CTRL (0x01) | 0x17 | 400kHz + STOP |
| 5 | USER_CTRL (0x03) | 0x20 | I2C Master AN |

Dann versucht es über Slave 4:
- AK09916 Register 0x00 lesen (WIA1, sollte 0x48 sein)
- AK09916 Register 0x01 lesen (WIA2, sollte 0x09 sein)

## Vorteile

| Feature | Dieses Script | Go-Programm |
|---------|---------------|-------------|
| **Dependencies** | Nur Python3 | Go + Libraries |
| **Kompilierung** | Keine | Nötig |
| **Zeit** | 2 Sekunden | 10+ Sekunden |
| **Debug** | Jeder Schritt sichtbar | Abstrahiert |
| **Änderungen** | Script editieren | Neu kompilieren |

## Debugging

Das Script zeigt **jeden Schritt** mit Debug-Output:

```python
Step 1: Disable I2C bypass mode
Step 2: Disable I2C_MST_CYCLE (enable continuous mode)
  LP_CONFIG = 0x40
  ✓ Cleared I2C_MST_CYCLE bit
...
```

Du siehst sofort, wo es hängt!

## Nächste Schritte

1. **Führe das Script aus**
2. **Schick mir den Output**
3. Wenn WHO_AM_I = 0x00:
   - Problem ist in der I2C Master Konfiguration
   - Müssen Register prüfen
4. Wenn WHO_AM_I = 0x48/0x09:
   - **ERFOLG!** Magnetometer antwortet
   - Dann Go-Code entsprechend anpassen

## Troubleshooting

**"No module named 'smbus'":**
```bash
sudo apt install python3-smbus
```

**"Permission denied":**
```bash
sudo python3 /root/simple-mag-test.py
```

**"I2C device not found":**
```bash
# Check if I2C is enabled
sudo i2cdetect -y 1

# Should show 0x68
```
