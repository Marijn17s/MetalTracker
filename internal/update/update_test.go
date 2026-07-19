package update

import (
	"runtime"
	"strings"
	"testing"
)

func TestExpectedAssetName(t *testing.T) {
	name := ExpectedAssetName()
	if !strings.HasPrefix(name, "MetalTracker-") {
		t.Fatalf("unexpected asset name %q", name)
	}
	switch runtime.GOOS {
	case "windows":
		if name != "MetalTracker-Windows-x64-Setup.exe" {
			t.Fatalf("windows asset = %q", name)
		}
	case "darwin":
		want := "MetalTracker-macOS-Intel.dmg"
		if runtime.GOARCH == "arm64" {
			want = "MetalTracker-macOS-AppleSilicon.dmg"
		}
		if name != want {
			t.Fatalf("darwin asset = %q, want %q", name, want)
		}
	default:
		if name != "MetalTracker-Linux-x64.tar.gz" {
			t.Fatalf("linux asset = %q", name)
		}
	}
}

func TestNormalizeSemver(t *testing.T) {
	if got := normalizeSemver("1.2.0"); got != "v1.2.0" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeSemver("v1.2.0"); got != "v1.2.0" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeSemver("dev"); got != "" {
		t.Fatalf("dev should normalize empty, got %q", got)
	}
}
