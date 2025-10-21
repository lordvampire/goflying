#!/bin/bash
# Patch Script für ICM20948 Magnetometer Support in Stratux
# Führe dies NACH dem VirusPilot Setup-Script aus

set -e  # Bei Fehler abbrechen

echo "=================================================="
echo "Stratux ICM20948 Magnetometer Patch"
echo "by lordvampire - AK09916 Support"
echo "=================================================="

# Farben für Ausgabe
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Prüfe ob wir root sind
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Bitte als root ausführen (sudo)${NC}"
  exit 1
fi

# Backup erstellen
echo -e "${YELLOW}Erstelle Backup des aktuellen Stratux...${NC}"
if [ -d "/root/stratux" ]; then
  cp -r /root/stratux /root/stratux.backup.$(date +%Y%m%d_%H%M%S)
  echo -e "${GREEN}✓ Backup erstellt${NC}"
fi

# Stoppe Stratux falls es läuft
echo -e "${YELLOW}Stoppe Stratux Service...${NC}"
systemctl stop stratux 2>/dev/null || true
echo -e "${GREEN}✓ Service gestoppt${NC}"

# Gehe ins Stratux Verzeichnis
cd /root/stratux || exit 1

# Aktuellen goflying Dependency merken
echo -e "${YELLOW}Aktuelle goflying Version:${NC}"
grep "github.com/b3nn0/goflying" go.mod

# Ersetze goflying mit lordvampire Fork
echo -e "${YELLOW}Ersetze goflying mit lordvampire/goflying (Magnetometer-Support)...${NC}"

# Ändere go.mod um lordvampire/goflying zu nutzen
go mod edit -replace github.com/b3nn0/goflying=github.com/lordvampire/goflying@master

# Hole die neue Version
echo -e "${YELLOW}Lade lordvampire/goflying...${NC}"
go get github.com/lordvampire/goflying@master

# Zeige die neue Version
echo -e "${GREEN}Neue goflying Version:${NC}"
grep "lordvampire/goflying" go.mod go.sum

# Räume auf und hole alle Dependencies
echo -e "${YELLOW}Hole alle Dependencies...${NC}"
go mod download
go mod tidy

# Baue Stratux neu
echo -e "${YELLOW}Baue Stratux mit Magnetometer-Support...${NC}"
cd /root/stratux
make clean
make

# Installiere
echo -e "${YELLOW}Installiere neue Stratux Version...${NC}"
make install

# Starte Service neu
echo -e "${YELLOW}Starte Stratux Service...${NC}"
systemctl start stratux

# Warte kurz
sleep 3

# Prüfe Status
echo -e "${YELLOW}Prüfe Service Status...${NC}"
if systemctl is-active --quiet stratux; then
  echo -e "${GREEN}✓ Stratux läuft!${NC}"
else
  echo -e "${RED}✗ Stratux läuft nicht - prüfe Logs:${NC}"
  echo "  journalctl -u stratux -f"
fi

echo ""
echo "=================================================="
echo -e "${GREEN}✓ ICM20948 Magnetometer Patch abgeschlossen!${NC}"
echo "=================================================="
echo ""
echo "Nächste Schritte:"
echo "1. Prüfe ob ICM20948 erkannt wird:"
echo "   sudo i2cdetect -y 1"
echo ""
echo "2. Prüfe Stratux Logs:"
echo "   journalctl -u stratux -f"
echo ""
echo "3. Öffne Web Interface:"
echo "   http://$(hostname -I | awk '{print $1}')"
echo ""
echo "Die Magnetometer-Werte sollten jetzt in AHRS verfügbar sein!"
echo ""
echo "Bei Problemen: Backup wiederherstellen mit:"
echo "  sudo systemctl stop stratux"
echo "  sudo rm -rf /root/stratux"
echo "  sudo mv /root/stratux.backup.* /root/stratux"
echo "  sudo systemctl start stratux"
echo ""
