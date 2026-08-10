package config

import (
	"testing"
	"time"
)

func TestOperatorModeDefaultsOffWithOneHourLease(t *testing.T) {
	cfg, err := LoadFrom(env(nil))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.OperatorMode {
		t.Fatal("operator mode must default off")
	}
	if cfg.OperatorLeaseDuration != time.Hour {
		t.Fatalf("operator lease = %s, want 1h", cfg.OperatorLeaseDuration)
	}
}

func TestOperatorModeParsesExplicitLease(t *testing.T) {
	cfg, err := LoadFrom(env(map[string]string{
		"OPERATOR_MODE":           "true",
		"OPERATOR_LEASE_DURATION": "45m",
	}))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if !cfg.OperatorMode {
		t.Fatal("expected operator mode enabled")
	}
	if cfg.OperatorLeaseDuration != 45*time.Minute {
		t.Fatalf("operator lease = %s, want 45m", cfg.OperatorLeaseDuration)
	}
}

func TestOperatorLeaseIsBounded(t *testing.T) {
	for _, value := range []string{"1m", "25h"} {
		if _, err := LoadFrom(env(map[string]string{"OPERATOR_LEASE_DURATION": value})); err == nil {
			t.Fatalf("expected lease %q to be rejected", value)
		}
	}
}
