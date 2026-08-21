package utils

import (
	"fmt"
	"strings"
	"time"
)

var bogotaLocation *time.Location

func init() {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		// UTC-5 fallback
		loc = time.FixedZone("America/Bogota", -5*60*60)
	}
	bogotaLocation = loc
}

// GetBogotaTime retorna la hora actual en zona horaria de Colombia
func GetBogotaTime() time.Time {
	return time.Now().In(bogotaLocation)
}

// FormatSpanishTime12h formatea un time.Time al estilo colombiano 12 horas: "02/01/2006, 03:04 p. m."
func FormatSpanishDateTime12h(t time.Time) string {
	tBogota := t.In(bogotaLocation)
	dayMonthYear := tBogota.Format("02/01/2006")
	hour12 := tBogota.Format("03:04")
	ampm := "a. m."
	if tBogota.Hour() >= 12 {
		ampm = "p. m."
	}
	return fmt.Sprintf("%s, %s %s", dayMonthYear, hour12, ampm)
}

// FormatSpanishTimeOnly12h formatea solo la hora al estilo: "03:04:05 p. m."
func FormatSpanishTimeOnly12h(t time.Time) string {
	tBogota := t.In(bogotaLocation)
	timeStr := tBogota.Format("03:04:05")
	ampm := "a. m."
	if tBogota.Hour() >= 12 {
		ampm = "p. m."
	}
	return fmt.Sprintf("%s %s", timeStr, ampm)
}

// ParseAndFormatDate toma una fecha en cualquier formato (ISO 8601, RFC3339, YYYY-MM-DD, etc.)
// y la convierte a formato normal estándar de BusinessAdmin: "DD/MM/YYYY, hh:mm a" (ej. "20/08/2026, 07:15 p. m.")
func ParseAndFormatDate(dateStr string) string {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return FormatSpanishDateTime12h(GetBogotaTime())
	}

	// Si ya está formateada como DD/MM/YYYY hh:mm... retornarla directamente o sanitizarla
	if strings.Contains(dateStr, "a. m.") || strings.Contains(dateStr, "p. m.") ||
		strings.Contains(dateStr, "a.m.") || strings.Contains(dateStr, "p.m.") ||
		strings.Contains(dateStr, "AM") || strings.Contains(dateStr, "PM") {
		return dateStr
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"02/01/2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return FormatSpanishDateTime12h(t)
		}
	}

	// Si no se pudo parsear, devolver el string original o la hora actual
	return dateStr
}
