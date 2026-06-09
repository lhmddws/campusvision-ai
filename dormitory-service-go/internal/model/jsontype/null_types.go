package jsontype

import (
	"database/sql"
	"encoding/json"
	"time"
)

// NullFloat64 wraps sql.NullFloat64 with proper JSON serialization.
// Serializes as null when Valid=false, or as a plain number when Valid=true.
type NullFloat64 struct {
	sql.NullFloat64
}

// MarshalJSON implements json.Marshaler.
func (nf NullFloat64) MarshalJSON() ([]byte, error) {
	if !nf.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nf.Float64)
}

// UnmarshalJSON implements json.Unmarshaler.
func (nf *NullFloat64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nf.Valid = false
		return nil
	}
	nf.Valid = true
	return json.Unmarshal(data, &nf.Float64)
}

// NullInt64 wraps sql.NullInt64 with proper JSON serialization.
type NullInt64 struct {
	sql.NullInt64
}

// MarshalJSON implements json.Marshaler.
func (ni NullInt64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ni.Int64)
}

// UnmarshalJSON implements json.Unmarshaler.
func (ni *NullInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ni.Valid = false
		return nil
	}
	ni.Valid = true
	return json.Unmarshal(data, &ni.Int64)
}

// NullString wraps sql.NullString with proper JSON serialization.
type NullString struct {
	sql.NullString
}

// MarshalJSON implements json.Marshaler.
func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

// UnmarshalJSON implements json.Unmarshaler.
func (ns *NullString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ns.Valid = false
		return nil
	}
	ns.Valid = true
	return json.Unmarshal(data, &ns.String)
}

// NullTime wraps sql.NullTime with proper JSON serialization.
type NullTime struct {
	sql.NullTime
}

// MarshalJSON implements json.Marshaler.
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nt.Time)
}

// UnmarshalJSON implements json.Unmarshaler.
func (nt *NullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}
	nt.Valid = true
	return json.Unmarshal(data, &nt.Time)
}

// compile-time interface checks
var (
	_ json.Marshaler   = NullFloat64{}
	_ json.Unmarshaler = (*NullFloat64)(nil)
	_ json.Marshaler   = NullInt64{}
	_ json.Unmarshaler = (*NullInt64)(nil)
	_ json.Marshaler   = NullString{}
	_ json.Unmarshaler = (*NullString)(nil)
	_ json.Marshaler   = NullTime{}
	_ json.Unmarshaler = (*NullTime)(nil)
)

// Helper constructors for convenience.
func NewNullFloat64(v float64) NullFloat64 {
	return NullFloat64{sql.NullFloat64{Float64: v, Valid: true}}
}

func NewNullInt64(v int64) NullInt64 {
	return NullInt64{sql.NullInt64{Int64: v, Valid: true}}
}

func NewNullString(v string) NullString {
	return NullString{sql.NullString{String: v, Valid: true}}
}

func NewNullTime(v time.Time) NullTime {
	return NullTime{sql.NullTime{Time: v, Valid: true}}
}
