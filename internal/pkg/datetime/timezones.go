// Package datetime provides timezone and timestamp parsing utilities for cencli.
//
// This file defines the supported timezone identifiers that can be used
// for the default-tz configuration option. These timezones are used when
// parsing timestamp inputs that don't include explicit timezone information.
package datetime

import (
	"encoding"
	"fmt"
	"time"

	_ "time/tzdata"
)

var locations = map[TimeZone]*time.Location{}

func init() {
	for _, tz := range allTimeZones {
		loc, err := time.LoadLocation(string(tz))
		if err != nil {
			panic(fmt.Sprintf("failed to load timezone %s: %v", tz, err))
		}
		locations[tz] = loc
	}
}

type TimeZone string

// location returns the *time.Location for the given TimeZone
func (tz TimeZone) location() *time.Location {
	return locations[tz]
}

var _ encoding.TextUnmarshaler = (*TimeZone)(nil)

func (tz *TimeZone) UnmarshalText(text []byte) error {
	*tz = TimeZone(text)
	if _, ok := locations[*tz]; !ok {
		return fmt.Errorf("invalid timezone: %s", *tz)
	}
	return nil
}

const (
	// Core
	TimeZoneUTC TimeZone = "UTC"

	// America
	TimeZoneAmericaNewYork    TimeZone = "America/New_York"
	TimeZoneAmericaChicago    TimeZone = "America/Chicago"
	TimeZoneAmericaDenver     TimeZone = "America/Denver"
	TimeZoneAmericaLosAngeles TimeZone = "America/Los_Angeles"
	TimeZoneAmericaPhoenix    TimeZone = "America/Phoenix"
	TimeZoneAmericaToronto    TimeZone = "America/Toronto"
	TimeZoneAmericaSaoPaulo   TimeZone = "America/Sao_Paulo"
	TimeZoneAmericaMexicoCity TimeZone = "America/Mexico_City"
	TimeZoneAmericaAnchorage  TimeZone = "America/Anchorage"

	// Europe
	TimeZoneEuropeLondon   TimeZone = "Europe/London"
	TimeZoneEuropeParis    TimeZone = "Europe/Paris"
	TimeZoneEuropeBerlin   TimeZone = "Europe/Berlin"
	TimeZoneEuropeMadrid   TimeZone = "Europe/Madrid"
	TimeZoneEuropeRome     TimeZone = "Europe/Rome"
	TimeZoneEuropeMoscow   TimeZone = "Europe/Moscow"
	TimeZoneEuropeIstanbul TimeZone = "Europe/Istanbul"
	TimeZoneEuropeWarsaw   TimeZone = "Europe/Warsaw"

	// Asia
	TimeZoneAsiaTokyo     TimeZone = "Asia/Tokyo"
	TimeZoneAsiaShanghai  TimeZone = "Asia/Shanghai"
	TimeZoneAsiaSingapore TimeZone = "Asia/Singapore"
	TimeZoneAsiaHongKong  TimeZone = "Asia/Hong_Kong"
	TimeZoneAsiaBangkok   TimeZone = "Asia/Bangkok"
	TimeZoneAsiaKolkata   TimeZone = "Asia/Kolkata"
	TimeZoneAsiaSeoul     TimeZone = "Asia/Seoul"
	TimeZoneAsiaDubai     TimeZone = "Asia/Dubai"
	TimeZoneAsiaJerusalem TimeZone = "Asia/Jerusalem"

	// Oceania
	TimeZoneAustraliaSydney    TimeZone = "Australia/Sydney"
	TimeZoneAustraliaMelbourne TimeZone = "Australia/Melbourne"
	TimeZonePacificAuckland    TimeZone = "Pacific/Auckland"
	TimeZonePacificHonolulu    TimeZone = "Pacific/Honolulu"

	// Africa
	TimeZoneAfricaCairo        TimeZone = "Africa/Cairo"
	TimeZoneAfricaJohannesburg TimeZone = "Africa/Johannesburg"
	TimeZoneAfricaNairobi      TimeZone = "Africa/Nairobi"
	TimeZoneAfricaLagos        TimeZone = "Africa/Lagos"
)

var allTimeZones = []TimeZone{
	TimeZoneUTC,

	TimeZoneAmericaNewYork,
	TimeZoneAmericaChicago,
	TimeZoneAmericaDenver,
	TimeZoneAmericaLosAngeles,
	TimeZoneAmericaPhoenix,
	TimeZoneAmericaToronto,
	TimeZoneAmericaSaoPaulo,
	TimeZoneAmericaMexicoCity,
	TimeZoneAmericaAnchorage,

	TimeZoneEuropeLondon,
	TimeZoneEuropeParis,
	TimeZoneEuropeBerlin,
	TimeZoneEuropeMadrid,
	TimeZoneEuropeRome,
	TimeZoneEuropeMoscow,
	TimeZoneEuropeIstanbul,
	TimeZoneEuropeWarsaw,

	TimeZoneAsiaTokyo,
	TimeZoneAsiaShanghai,
	TimeZoneAsiaSingapore,
	TimeZoneAsiaHongKong,
	TimeZoneAsiaBangkok,
	TimeZoneAsiaKolkata,
	TimeZoneAsiaSeoul,
	TimeZoneAsiaDubai,
	TimeZoneAsiaJerusalem,

	TimeZoneAustraliaSydney,
	TimeZoneAustraliaMelbourne,
	TimeZonePacificAuckland,
	TimeZonePacificHonolulu,

	TimeZoneAfricaCairo,
	TimeZoneAfricaJohannesburg,
	TimeZoneAfricaNairobi,
	TimeZoneAfricaLagos,
}
