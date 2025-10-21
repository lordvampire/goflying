# Stratux Magnetometer Update Script

Automatisches Update-Script für Stratux mit ICM20948 AK09916 Magnetometer-Unterstützung.

## Verwendung auf dem Raspberry Pi

```bash
# Script auf Raspberry Pi kopieren
scp update-stratux-magnetometer.sh root@<raspberry-ip>:/root/

# Auf dem Raspberry Pi ausführbar machen und ausführen
chmod +x /root/update-stratux-magnetometer.sh
sudo /root/update-stratux-magnetometer.sh
```

## Was macht das Script?

1. **Holt automatisch den neuesten Commit-Hash** von GitHub (lordvampire/goflying, Branch: stratux_master)
2. **Löscht den Go Module Cache** (go clean -modcache)
3. **Aktualisiert die replace directive** in go.mod mit dem neuesten Hash
4. **Lädt alle Module neu** (go mod download, go mod tidy)
5. **Verifiziert**, dass der neue Code geladen wurde (prüft auf Debug-Logging und I2C_MST_ODR_CONFIG)
6. **Baut Stratux neu** (make clean, make, make install)
7. **Fragt**, ob Stratux neugestartet werden soll

## Voraussetzungen

- Root-Zugriff auf dem Raspberry Pi
- Internet-Verbindung (für GitHub API)
- `jq` installiert (für JSON parsing):
  ```bash
  apt install jq -y
  ```

## Manuelle Hash-Eingabe

Falls die GitHub API nicht erreichbar ist, fragt das Script nach manueller Hash-Eingabe.

Den aktuellen Hash finden Sie hier:
https://github.com/lordvampire/goflying/commits/stratux_master

## Fehlerbehandlung

Das Script stoppt bei Fehlern (`set -e`). Bei Problemen:

1. Prüfen Sie die Ausgabe auf Fehler
2. Verifizieren Sie Internet-Verbindung
3. Prüfen Sie, ob Go korrekt installiert ist: `/root/go/bin/go version`

## Nach dem Update

Logs überwachen:
```bash
journalctl -u stratux -f
```

Erwartete Log-Meldungen:
- `ICM20948: Initializing AK09916 magnetometer...`
- `ICM20948: I2C master ODR set to 200 Hz`
- `ICM20948: AK09916 WHO_AM_I: WIA1=0x48 (expect 0x48), WIA2=0x09 (expect 0x09)`
- `ICM20948: Magnetometer read #1: M1=..., M2=..., M3=...`

## Troubleshooting

**Problem:** "Could not fetch latest commit hash from GitHub"
- Lösung: Internet-Verbindung prüfen oder Hash manuell eingeben

**Problem:** "Debug logging not found"
- Lösung: Cache nochmal löschen: `go clean -modcache` und Script erneut ausführen

**Problem:** Build schlägt fehl
- Lösung: Prüfen Sie verfügbaren Speicher: `df -h`
