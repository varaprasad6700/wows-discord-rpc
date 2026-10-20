package detector_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/detector"
)

func TestDeployEmbeddedMod(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wows_deploy_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	buildPath := filepath.Join(tempDir, "bin", "1234567")
	if err := os.MkdirAll(buildPath, 0755); err != nil {
		t.Fatalf("failed to create build path: %v", err)
	}

	// This should fail before DeployEmbeddedMod is implemented
	err = detector.DeployEmbeddedMod(buildPath)
	if err != nil {
		t.Fatalf("DeployEmbeddedMod failed: %v", err)
	}

	loaderPath := filepath.Join(buildPath, "res_mods", "PnFModsLoader.py")
	if _, err := os.Stat(loaderPath); os.IsNotExist(err) {
		t.Errorf("expected PnFModsLoader.py to exist at %s", loaderPath)
	}

	mainPath := filepath.Join(buildPath, "res_mods", "PnFMods", "TelemetryExporter", "Main.py")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Errorf("expected Main.py to exist at %s", mainPath)
	}
}
