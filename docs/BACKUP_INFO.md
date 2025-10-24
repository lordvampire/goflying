# Backup des funktionierenden Magnetometer-Status

**Backup-Zeitpunkt:** 2025-10-24 14:09 UTC
**Backup-Verzeichnis:** `/root/backup-20251024-140931/`

## Was wurde gesichert?

1. **goflying-patched/** (780K)
   - Vollständige funktionierende goflying-Version
   - **Kritischer Fix:** `ICMREG_USER_CTRL = 0x03` in `icm20948/dmp_constants.go` (Zeile 74)
   - Magnetometer-Initialisierung funktioniert

2. **sensors.go.working** (17K)
   - Stratux sensors.go mit MagHeading-Integration
   - **Zeilen 371-376:** `s.MagHeading()` statt hardcoded `ahrs.Invalid`
   - AHRS MagHeading wird korrekt berechnet

3. **go.mod.working** (3K)
   - Stratux go.mod mit korrekter goflying replace-Direktive
   - Zeigt auf funktionierende lordvampire/goflying-b3nn0 Version

## Status vor dem Backup

✅ **Magnetometer:** Funktioniert (Werte: M1~5000, M2~-4000, M3~-22000)
✅ **AHRS Integration:** MagHeading wird berechnet (~17889°, unkalibriert)
✅ **API Sichtbarkeit:** `/getSituation` zeigt AHRSMagHeading
✅ **I2C Master:** Korrekt aktiviert mit USER_CTRL = 0x03

## Wiederherstellen des funktionierenden Status

Falls nach dem Neu-Kompilieren etwas nicht funktioniert:

```bash
sudo /root/restore-working-magnetometer.sh
```

Das Script wird:
1. Stratux stoppen
2. sensors.go mit MagHeading-Fix wiederherstellen
3. go.mod wiederherstellen
4. USER_CTRL Fix (0x03) auf heruntergeladene goflying-Module anwenden
5. Stratux neu bauen und installieren
6. Optional Stratux neu starten

## Manuelle Wiederherstellung

Falls das Script nicht funktioniert:

### 1. sensors.go wiederherstellen:
```bash
cp /root/backup-20251024-140931/sensors.go.working /root/stratux/main/sensors.go
```

### 2. go.mod wiederherstellen:
```bash
cp /root/backup-20251024-140931/go.mod.working /root/stratux/go.mod
```

### 3. USER_CTRL Fix manuell anwenden:

In heruntergeladener goflying-Version, Datei `icm20948/dmp_constants.go`, Zeile 74:

**Ändern von:**
```go
ICMREG_USER_CTRL = 0x6A
```

**Zu:**
```go
ICMREG_USER_CTRL = 0x03 // Bank 0: CRITICAL FIX! Was 0x6A (MPU-9250), should be 0x03 (ICM-20948)
```

### 4. Stratux neu bauen:
```bash
cd /root/stratux
make clean
make
sudo make install
sudo systemctl restart stratux
```

## Verifizierung nach Wiederherstellung

```bash
# Magnetometer-Werte prüfen
curl -s http://localhost/getSituation | python3 -c "import sys, json; print('MagHeading:', json.load(sys.stdin)['AHRSMagHeading'])"

# Erwartetes Ergebnis: Numerischer Wert (nicht 3276.7)
```

## Wichtige Dateien auf dem System

- `/root/GOFLYING_CHANGES.md` - Dokumentation der goflying Änderungen
- `/root/STRATUX_CHANGES.md` - Dokumentation der Stratux Änderungen
- `/root/GIT_PUSH_ANLEITUNG.md` - Anleitung zum Pushen auf GitHub
- `/root/update-stratux-magnetometer.sh` - Automatisches Update-Script
- `/root/restore-working-magnetometer.sh` - Dieses Wiederherstellungs-Script

## Der kritische Unterschied

**Ein einziges Byte entscheidet:**
- `ICMREG_USER_CTRL = 0x6A` → Magnetometer funktioniert NICHT ❌
- `ICMREG_USER_CTRL = 0x03` → Magnetometer funktioniert ✅

Dieser Bug existierte, weil der ICM-20948 Code ursprünglich auf MPU-9250 Konstanten basierte.
