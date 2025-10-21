#!/bin/bash
# Komplettes Stratux EU32 Setup mit ICM20948 Magnetometer Support
# Führe dies auf einem blanken Debian/Raspberry Pi OS aus

set -e  # Bei Fehler abbrechen

echo "=========================================================="
echo "Stratux EU32 Installation mit ICM20948 Magnetometer"
echo "by lordvampire - Komplette Installation in einem Durchgang"
echo "=========================================================="

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

# System Update
echo -e "${YELLOW}Aktualisiere System...${NC}"
apt-get update
apt-get upgrade -y
echo -e "${GREEN}✓ System aktualisiert${NC}"

# Installiere Build-Abhängigkeiten
echo -e "${YELLOW}Installiere Build-Abhängigkeiten...${NC}"
apt-get install -y \
    git \
    golang \
    libusb-1.0-0-dev \
    pkg-config \
    libjpeg-dev \
    i2c-tools \
    cmake \
    build-essential \
    mercurial \
    autoconf \
    fftw3-dev \
    libtool
echo -e "${GREEN}✓ Abhängigkeiten installiert${NC}"

# Aktiviere I2C
echo -e "${YELLOW}Aktiviere I2C...${NC}"
if ! grep -q "^dtparam=i2c_arm=on" /boot/config.txt; then
    echo "dtparam=i2c_arm=on" >> /boot/config.txt
fi
if ! grep -q "^i2c-dev" /etc/modules; then
    echo "i2c-dev" >> /etc/modules
fi
modprobe i2c-dev
echo -e "${GREEN}✓ I2C aktiviert${NC}"

# Erstelle Arbeitsverzeichnis
echo -e "${YELLOW}Erstelle Arbeitsverzeichnis...${NC}"
cd /root
if [ -d "stratux" ]; then
    echo -e "${YELLOW}Altes Stratux Verzeichnis gefunden, erstelle Backup...${NC}"
    mv stratux stratux.backup.$(date +%Y%m%d_%H%M%S)
fi
echo -e "${GREEN}✓ Verzeichnis vorbereitet${NC}"

# Clone Stratux Repository
echo -e "${YELLOW}Lade Stratux EU32 herunter...${NC}"
git clone --branch v1.6r1-eu032 --depth 1 https://github.com/b3nn0/stratux.git
cd stratux
echo -e "${GREEN}✓ Stratux heruntergeladen${NC}"

# Zeige aktuelle goflying Version
echo -e "${YELLOW}Aktuelle goflying Version:${NC}"
grep "github.com/b3nn0/goflying" go.mod || echo "Nicht gefunden"

# Ersetze goflying mit lordvampire Fork (MIT MAGNETOMETER SUPPORT!)
echo -e "${YELLOW}Ersetze goflying mit lordvampire/goflying (Magnetometer-Support)...${NC}"
go mod edit -replace github.com/b3nn0/goflying=github.com/lordvampire/goflying@master

# Hole die neue Version
echo -e "${YELLOW}Lade lordvampire/goflying...${NC}"
go get github.com/lordvampire/goflying@master

# Zeige die neue Version
echo -e "${GREEN}Neue goflying Version:${NC}"
grep "lordvampire/goflying" go.mod go.sum || echo "Prüfe go.sum..."

# Hole alle Dependencies
echo -e "${YELLOW}Hole alle Dependencies...${NC}"
go mod download
go mod tidy
echo -e "${GREEN}✓ Dependencies geladen${NC}"

# Baue Stratux MIT MAGNETOMETER SUPPORT
echo -e "${YELLOW}Baue Stratux mit Magnetometer-Support...${NC}"
echo -e "${YELLOW}Das kann 10-20 Minuten dauern...${NC}"
make clean
make
echo -e "${GREEN}✓ Build erfolgreich!${NC}"

# Installiere
echo -e "${YELLOW}Installiere Stratux...${NC}"
make install
echo -e "${GREEN}✓ Stratux installiert${NC}"

# Erstelle Systemd Service (falls nicht vorhanden)
if [ ! -f "/lib/systemd/system/stratux.service" ]; then
    echo -e "${YELLOW}Erstelle Stratux Service...${NC}"
    cat > /lib/systemd/system/stratux.service <<EOF
[Unit]
Description=Stratux ADS-B/AHRS receiver
DefaultDependencies=no
After=local-fs.target network.target

[Service]
Type=simple
ExecStart=/usr/bin/stratux
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable stratux
    echo -e "${GREEN}✓ Service erstellt${NC}"
fi

# Starte Service
echo -e "${YELLOW}Starte Stratux Service...${NC}"
systemctl start stratux

# Warte kurz
sleep 5

# Prüfe Status
echo -e "${YELLOW}Prüfe Service Status...${NC}"
if systemctl is-active --quiet stratux; then
  echo -e "${GREEN}✓ Stratux läuft!${NC}"
else
  echo -e "${RED}✗ Stratux läuft nicht - prüfe Logs:${NC}"
  echo "  journalctl -u stratux -f"
fi

echo ""
echo "=========================================================="
echo -e "${GREEN}✓ Installation abgeschlossen!${NC}"
echo "=========================================================="
echo ""
echo "Nächste Schritte:"
echo ""
echo "1. Prüfe ob ICM20948 erkannt wird:"
echo "   sudo i2cdetect -y 1"
echo "   (Sollte 0x68 oder 0x69 zeigen)"
echo ""
echo "2. Prüfe Stratux Logs:"
echo "   journalctl -u stratux -f"
echo "   (Suche nach 'ICM20948' und 'magnetometer')"
echo ""
echo "3. Öffne Web Interface:"
echo "   http://$(hostname -I | awk '{print $1}')"
echo ""
echo "4. Im Web Interface unter AHRS solltest du nun Magnetometer-Werte sehen:"
echo "   - M1, M2, M3 sollten nicht-null sein"
echo "   - Magnetic Heading sollte sich ändern wenn du den Pi drehst"
echo ""
echo "Bei Problemen: Logs prüfen mit 'journalctl -u stratux -n 100'"
echo ""
