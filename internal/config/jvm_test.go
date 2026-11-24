package config

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGCFlagsForJava25(t *testing.T) {
	t.Parallel()

	g1 := G1Flags()
	if !contains(g1, "-XX:+UseG1GC") || !contains(g1, "-XX:+ExitOnOutOfMemoryError") {
		t.Fatalf("G1 flags = %v, want UseG1GC and ExitOnOutOfMemoryError", g1)
	}
	if contains(g1, "-XX:+ZGenerational") {
		t.Fatal("G1 flags include removed ZGenerational")
	}

	zgc := ZGCFlags()
	if !contains(zgc, "-XX:+UseZGC") || !contains(zgc, "-XX:+ExitOnOutOfMemoryError") {
		t.Fatalf("ZGC flags = %v, want UseZGC and ExitOnOutOfMemoryError", zgc)
	}
	if contains(zgc, "-XX:+ZGenerational") || contains(zgc, "-XX:+UseG1GC") {
		t.Fatalf("ZGC flags = %v, still carry a removed or wrong collector flag", zgc)
	}
	if GCFlags("G1GC")[0] != "-XX:+UseG1GC" {
		t.Fatal("unknown collector should fall through to G1")
	}
}

func TestStartupScriptsEmitJava25Flags(t *testing.T) {
	t.Parallel()

	scripts := []string{
		"../../entrypoint.sh",
		"../../embedded/templates/start.sh.tmpl",
		"../../cmd/mc-dad-server/embedded/templates/start.sh.tmpl",
	}
	for _, path := range scripts {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		text := string(body)
		if strings.Contains(text, "-XX:+ZGenerational") {
			t.Errorf("%s still passes -XX:+ZGenerational, which Java 25 removed", path)
		}
		for _, flag := range G1Flags() {
			if !strings.Contains(text, flag) {
				t.Errorf("%s missing G1 flag %s", path, flag)
			}
		}
		for _, flag := range ZGCFlags() {
			if !strings.Contains(text, flag) {
				t.Errorf("%s missing ZGC flag %s", path, flag)
			}
		}
	}
}

func TestSystemJavaAcceptsFlags(t *testing.T) {
	java, err := exec.LookPath("java")
	if err != nil {
		t.Skip("java is not on PATH")
	}

	for _, gc := range []string{"g1gc", "zgc"} {
		t.Run(gc, func(t *testing.T) {
			args := append([]string{"-Xms512M", "-Xmx512M"}, GCFlags(gc)...)
			args = append(args, "-version")
			cmd := exec.CommandContext(context.Background(), java, args...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("java rejected %s flags: %v\n%s", gc, err, out)
			}
		})
	}
}

func contains(flags []string, want string) bool {
	for _, flag := range flags {
		if flag == want {
			return true
		}
	}
	return false
}
