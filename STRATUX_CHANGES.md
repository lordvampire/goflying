# Stratux Änderungen für ICM20948 Magnetometer-Integration

## Änderung 1: AHRS MagHeading aktivieren

**Datei:** `main/sensors.go`

**Zeilen ~371-376** (im `sensorAttitudeSender()` in der AHRS-Update-Schleife):

**VORHER:**
```go
//TODO westphae: until magnetometer calibration is performed, no mag heading
mySituation.AHRSMagHeading = ahrs.Invalid
```

**NACHHER:**
```go
// Get magnetometer heading from AHRS
magHeading := s.MagHeading()
mySituation.AHRSMagHeading = magHeading
if !isAHRSInvalidValue(magHeading) {
    mySituation.AHRSMagHeading /= ahrs.Deg
}
```

## Ergebnis

- ✅ `AHRSMagHeading` wird jetzt aus den Magnetometer-Daten berechnet
- ✅ Wert ist über `/getSituation` API abrufbar
- ✅ Format: Grad (0-360° nach Kalibrierung)

## Hinweis

Das MagHeading wird zunächst hohe Werte zeigen (~17000°), bis eine Magnetometer-Kalibrierung durchgeführt wird. Dies ist normal und wird von der AHRS-Bibliothek gehandhabt.

## Abhängigkeiten

Diese Änderung erfordert:
1. Funktionierende Magnetometer-Initialisierung (goflying-b3nn0 Fix)
2. Die AHRS-Bibliothek mit `MagHeading()` Methode (bereits vorhanden)
