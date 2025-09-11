package server

import "testing"

func TestPortManager(t *testing.T) {
	pm := newPortManager(1234)
	port := pm.ReservePort()
	if port != 1234 {
		t.Errorf("Expected lowest available port to be reserved, got %d", port)
	}

	pm.ReleasePort(port)
	port = pm.ReservePort()
	if port != 1234 {
		t.Errorf("Expected port to be reused, got %d", port)
	}

	port2 := pm.ReservePort()
	if port2 != 1235 {
		t.Errorf("Expected next available port to be reserved, got %d", port)
	}
}
