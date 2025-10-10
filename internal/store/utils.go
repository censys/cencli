package store

import "time"

// toZulu formats a time.Time into a RFC3339 time string.
// Useful for SQLite timestamp fields.
func toZulu(t time.Time) string {
	return t.Format(time.RFC3339)
}

// fromZulu parses a Zulu time string into a time.Time.
// Returns a zero time.Time if the string is invalid.
func fromZulu(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
