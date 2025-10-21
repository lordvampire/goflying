# ICM20948 Magnetometer Schnelltest

Standalone Test-Programm für den ICM20948 Magnetometer - **viel schneller** als Stratux neu zu kompilieren!

## Warum dieses Script?

- ⚡ **Schnell:** Nur ~10 Sekunden statt 5+ Minuten für Stratux-Build
- 🎯 **Direkt:** Testet nur den ICM20948-Treiber, keine Stratux-Dependencies
- 🔍 **Detailliert:** Zeigt alle Debug-Logs vom Treiber
- 🔄 **Live:** Zeigt Sensor-Daten jede Sekunde

## Methode 1: Direkt auf dem Raspberry Pi (EMPFOHLEN)

**Ein einziger Befehl - alles automatisch:**

```bash
curl -sSL https://raw.githubusercontent.com/lordvampire/goflying/stratux_master/test-on-pi.sh | sudo bash
```

Was passiert:
1. ✓ Holt neuesten Code von GitHub
2. ✓ Kompiliert Test-Programm
3. ✓ Führt es sofort aus
4. ✓ Zeigt 30 Sekunden lang Magnetometer-Daten

**Oder manuell:**

```bash
# Script herunterladen
cd /root
wget https://raw.githubusercontent.com/lordvampire/goflying/stratux_master/test-on-pi.sh
chmod +x test-on-pi.sh

# Ausführen
sudo ./test-on-pi.sh
```

## Methode 2: Lokal kompilieren, dann deployen

**Auf deinem PC:**

```bash
cd /home/faruktuefekli/GitHub/icm20948/goflying-b3nn0

# Kompilieren für ARM (Raspberry Pi)
chmod +x build-test.sh
./build-test.sh <raspberry-pi-ip>

# Oder nur kompilieren ohne Deploy:
./build-test.sh
```

**Auf dem Raspberry Pi:**

```bash
# Wenn automatisch deployed:
sudo /root/test-magnetometer

# Oder manuell kopiert:
scp test-magnetometer-arm root@<pi-ip>:/root/test-magnetometer
ssh root@<pi-ip>
sudo /root/test-magnetometer
```

## Erwartete Ausgabe

### ✅ Wenn es funktioniert:

```
ICM20948: I2C bypass mode explicitly disabled (INT_PIN_CFG=0x00)
ICM20948: I2C master clock set to 400 kHz with STOP between reads
ICM20948: AK09916 WHO_AM_I: WIA1=0x48 (expect 0x48), WIA2=0x09 (expect 0x09)
ICM20948: AK09916 magnetometer initialization complete

[0001] Gyro: X=   0.12 Y=  -0.34 Z=   0.56 | Accel: X= 0.001 Y= 0.002 Z= 0.981
[0001] **MAG: X=  23.45 Y= -12.34 Z=  45.67 µT**
```

### ❌ Wenn es NICHT funktioniert:

```
ICM20948: AK09916 WHO_AM_I: WIA1=0x00 (expect 0x48), WIA2=0x00 (expect 0x09)
ICM20948 ERROR: AK09916 not responding correctly!

[0001] Gyro: X=   0.12 Y=  -0.34 Z=   0.56 | Accel: X= 0.001 Y= 0.002 Z= 0.981
[0001] **MAG: X=   0.00 Y=   0.00 Z=   0.00 µT**
[0001] WARNING: Magnetometer all zeros!
```

## Vorteile gegenüber Stratux-Build

| Aspekt | Stratux kompilieren | Test-Script |
|--------|---------------------|-------------|
| **Zeit** | ~5-10 Minuten | ~10 Sekunden |
| **Größe** | Ganzes Stratux | Nur ICM20948 |
| **Logs** | Vermischt mit Stratux | Nur Sensor |
| **Neustart** | Stratux-Service | Sofort sichtbar |
| **Iteration** | Langsam | **Schnell!** |

## Nach dem Test

Wenn der Magnetometer endlich funktioniert, kannst du die Änderungen in Stratux übernehmen:

```bash
# Normales Stratux-Update
sudo /root/update-stratux-magnetometer.sh
```

## Troubleshooting

**"Permission denied":**
```bash
sudo chmod +x test-on-pi.sh
sudo ./test-on-pi.sh
```

**"Command not found: jq":**
```bash
sudo apt install jq -y
```

**"Cannot find module":**
```bash
# Manuelle Dependencies installation
go clean -modcache
go mod download
go mod tidy
```

## Technische Details

Das Test-Programm:
- Importiert: `github.com/b3nn0/goflying/icm20948`
- Nutzt: `NewICM20948()` mit gleichen Parametern wie Stratux
- Liest: Gyro, Accel, Magnetometer jede Sekunde
- Läuft: 30 Sekunden (bei test-on-pi.sh) oder endlos (bei test-magnetometer.go)

## Workflow für schnelle Iteration

1. Code ändern in `/home/faruktuefekli/GitHub/icm20948/goflying-b3nn0/icm20948/`
2. Commit & push zu `lordvampire/goflying@stratux_master`
3. Auf Pi: `curl -sSL https://raw.githubusercontent.com/.../test-on-pi.sh | sudo bash`
4. Ergebnis in 10-20 Sekunden statt 5-10 Minuten! 🚀
