package engine

import (
	"errors"
	"strings"
	"testing"
)

func missing(string) (string, error) { return "", errors.New("not found") }
func found(string) (string, error)   { return "/usr/bin/" + "x", nil }

func TestDetect(t *testing.T) {
	if e, err := detect("darwin", missing); err != nil || e.Name != "container" {
		t.Errorf("darwin: got %v, %v", e, err)
	}
	if e, err := detect("windows", missing); err != nil || e.Name != "wslc" {
		t.Errorf("windows: got %v, %v", e, err)
	}
	if e, err := detect("linux", found); err != nil || e.Name != "podman" {
		t.Errorf("linux+podman: got %v, %v", e, err)
	}
	_, err := detect("linux", missing)
	if err == nil || !strings.Contains(err.Error(), "Docker is not supported.") {
		t.Errorf("linux without podman: want the existing refusal, got %v", err)
	}
	if _, err := detect("plan9", found); err == nil {
		t.Error("plan9: want unsupported-OS error")
	}
}
