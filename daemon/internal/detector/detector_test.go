package detector_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/detector"
)

func TestFindLatestBuild(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wows-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	builds := []string{"812345", "950123", "1024567", "not_a_build_folder"}
	for _, b := range builds {
		if err := os.Mkdir(filepath.Join(binDir, b), 0755); err != nil {
			t.Fatal(err)
		}
	}

	latest, err := detector.FindLatestBuild(tmpDir)
	if err != nil {
		t.Fatalf("expected to find latest build, got error: %v", err)
	}

	expected := "1024567"
	if filepath.Base(latest) != expected {
		t.Errorf("expected latest build '%s', got '%s'", expected, filepath.Base(latest))
	}
}

func TestFindLatestBuild_NoBin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wows-empty-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = detector.FindLatestBuild(tmpDir)
	if err == nil {
		t.Error("expected error when bin directory does not exist, got nil")
	}
}

func TestEnsureModDeployed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wows-deploy-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	buildPath := filepath.Join(tmpDir, "bin", "1234567")
	if err := os.MkdirAll(buildPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock mod source directory
	modSourceDir := filepath.Join(tmpDir, "mod_source")
	exporterSourceDir := filepath.Join(modSourceDir, "PnFMods", "TelemetryExporter")
	if err := os.MkdirAll(exporterSourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	loaderSrc := filepath.Join(modSourceDir, "PnFModsLoader.py")
	mainSrc := filepath.Join(exporterSourceDir, "Main.py")
	if err := os.WriteFile(loaderSrc, []byte("# loader v1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainSrc, []byte("# main v1"), 0644); err != nil {
		t.Fatal(err)
	}

	// Deploy
	err = detector.EnsureModDeployed(buildPath, modSourceDir)
	if err != nil {
		t.Fatalf("EnsureModDeployed failed: %v", err)
	}

	// Verify target files exist
	targetLoader := filepath.Join(buildPath, "res_mods", "PnFModsLoader.py")
	targetMain := filepath.Join(buildPath, "res_mods", "PnFMods", "TelemetryExporter", "Main.py")

	if _, err := os.Stat(targetLoader); err != nil {
		t.Errorf("expected target loader at %s, got: %v", targetLoader, err)
	}
	if _, err := os.Stat(targetMain); err != nil {
		t.Errorf("expected target main at %s, got: %v", targetMain, err)
	}

	// Verify content
	content, _ := os.ReadFile(targetLoader)
	if string(content) != "# loader v1" {
		t.Errorf("unexpected content: %s", string(content))
	}

	// Update source file and redeploy
	_ = os.WriteFile(loaderSrc, []byte("# loader v2"), 0644)
	if err := detector.EnsureModDeployed(buildPath, modSourceDir); err != nil {
		t.Fatalf("re-deploy failed: %v", err)
	}
	contentUpdated, _ := os.ReadFile(targetLoader)
	if string(contentUpdated) != "# loader v2" {
		t.Errorf("expected updated content '# loader v2', got '%s'", string(contentUpdated))
	}
}
