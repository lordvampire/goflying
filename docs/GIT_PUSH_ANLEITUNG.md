# Anleitung: lordvampire/goflying Änderungen pushen

## Schritt 1: Repository klonen (auf deinem lokalen Rechner)

```bash
git clone https://github.com/lordvampire/goflying.git
cd goflying
git checkout stratux_master
```

## Schritt 2: Kritischen Fix anwenden

**Datei:** `icm20948/dmp_constants.go`

Finde Zeile ~74 und ändere:

```go
ICMREG_USER_CTRL          = 0x6A
```

zu:

```go
ICMREG_USER_CTRL          = 0x03 // Bank 0: CRITICAL FIX! Was 0x6A (MPU-9250), should be 0x03 (ICM-20948)
```

## Schritt 3: Änderungen committen und pushen

```bash
git add icm20948/dmp_constants.go
git commit -m "Fix ICM20948 magnetometer: USER_CTRL register address

Critical bug fix: ICMREG_USER_CTRL was set to 0x6A (MPU-9250 address)
instead of 0x03 (ICM-20948 Bank 0 address). This prevented the I2C
Master from being enabled, which blocked magnetometer initialization.

With this fix:
- I2C Master activates correctly
- AK09916 magnetometer initializes successfully
- Continuous mag data is available for AHRS
- MagHeading can be calculated

Tested on: Raspberry Pi with ICM20948 9-DOF IMU"

git push origin stratux_master
```

## Schritt 4: Auf dem Pi testen

Nachdem du gepusht hast, auf dem Raspberry Pi:

```bash
sudo /root/update-stratux-magnetometer.sh
```

Das Script wird:
1. Den neuesten Commit von lordvampire/goflying (stratux_master branch) holen
2. Verifizieren dass USER_CTRL = 0x03 ist
3. sensors.go für MagHeading patchen
4. Stratux neu bauen und installieren
5. Optional Stratux neu starten und testen

## Verifizierung

Nach dem Neustart von Stratux:

```bash
# Magnetometer-Werte in den Logs prüfen
tail -f /var/log/stratux.log | grep Magnetometer

# MagHeading über API prüfen
curl -s http://localhost/getSituation | python3 -m json.tool | grep MagHeading

# Erwartetes Ergebnis: Ein numerischer Wert (nicht 3276.7)
```

## Dateien auf dem Pi

Die folgenden Dateien dokumentieren alle Änderungen:
- `/root/GOFLYING_CHANGES.md` - goflying-b3nn0 Änderungen
- `/root/STRATUX_CHANGES.md` - Stratux Änderungen
- `/root/update-stratux-magnetometer.sh` - Aktualisiertes Update-Script

## Zusammenfassung

**Ein einziger Byte macht den Unterschied:**
- `0x6A` = Magnetometer funktioniert NICHT
- `0x03` = Magnetometer funktioniert ✓

Dieser Bug existierte, weil die ICM-20948 Code auf MPU-9250 Konstanten basierte.
