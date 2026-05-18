package config

import (
	"testing"
)

func TestParse_FPSOverride(t *testing.T) {
	def := FrameConfig{FPSDay: 5, FPSNight: 1}
	got := ParseCameraConfig(`{"fps_override":10}`, def)
	if got.FPSDay != 10 {
		t.Errorf("FPSDay = %d, want 10", got.FPSDay)
	}
	if got.FPSNight != 10 {
		t.Errorf("FPSNight = %d, want 10", got.FPSNight)
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	def := FrameConfig{FPSDay: 5, FPSNight: 1}
	got := ParseCameraConfig("not valid", def)
	if got != def {
		t.Errorf("got %+v, want default %+v", got, def)
	}
}

func TestParse_EmptyString(t *testing.T) {
	def := FrameConfig{FPSDay: 5, FPSNight: 1}
	got := ParseCameraConfig("", def)
	if got != def {
		t.Errorf("got %+v, want default %+v", got, def)
	}
}

func TestParse_NullFPSOverride(t *testing.T) {
	def := FrameConfig{FPSDay: 5, FPSNight: 1}
	got := ParseCameraConfig(`{}`, def)
	if got != def {
		t.Errorf("got %+v, want default %+v", got, def)
	}
}

func TestParse_TypeParamsIgnored(t *testing.T) {
	def := FrameConfig{FPSDay: 5, FPSNight: 1}
	got := ParseCameraConfig(`{"type_params":{"detection":"fast"}}`, def)
	if got != def {
		t.Errorf("got %+v, want default %+v", got, def)
	}
}
