package platform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

func TestParseJavaVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		out     string
		want    int
		wantErr bool
	}{
		{
			name: "modern --version",
			out:  "openjdk 21.0.2 2024-01-16\nOpenJDK Runtime Environment Temurin-21.0.2+13\n",
			want: 21,
		},
		{
			name: "major only",
			out:  "openjdk 25 2025-09-16\n",
			want: 25,
		},
		{
			name: "legacy quoted -version",
			out:  "openjdk version \"21.0.2\" 2024-01-16\n",
			want: 21,
		},
		{
			name: "legacy 1.x",
			out:  "java version \"1.8.0_392\"\n",
			want: 8,
		},
		{
			name:    "no version",
			out:     "command not found\n",
			wantErr: true,
		},
		{
			name:    "empty",
			out:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseJavaVersion(tt.out)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got version %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestJavaVersionUsesStdoutFlag(t *testing.T) {
	t.Parallel()

	runner := NewMockRunner()
	runner.OutputMap["java [--version]"] = []byte("openjdk 21.0.2 2024-01-16\n")

	ver, err := javaVersion(t.Context(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != 21 {
		t.Fatalf("got %d, want 21", ver)
	}
}

func TestPaperJavaRequirement(t *testing.T) {
	t.Parallel()

	old := paperJavaRequirement(21, true)
	for _, want := range []string{
		"Paper 26.3 requires Java 25",
		"Eclipse Temurin",
		"This machine has Java 21.",
		"Temurin 25",
		temurin25URL,
		"Java 21 and Minecraft 1.21",
		"v0.2.0",
		previousJavaReleaseURL,
	} {
		if !strings.Contains(old, want) {
			t.Errorf("Java 21 message missing %q:\n%s", want, old)
		}
	}

	missing := paperJavaRequirement(0, false)
	if !strings.Contains(missing, "Java was not found on PATH.") {
		t.Fatalf("missing-java message = %q", missing)
	}
}

func TestInstallJavaKeepsTemurin25(t *testing.T) {
	t.Parallel()

	runner := &versionedRunner{
		MockRunner: NewMockRunner(),
		outputs:    []string{"openjdk 25.0.1 2025-10-21\n"},
	}
	runner.ExistsMap["java"] = true
	var buf bytes.Buffer
	err := InstallJava(t.Context(), runner, &Platform{PkgMgr: "apt"}, ui.NewWriter(&buf, false))
	if err != nil {
		t.Fatalf("InstallJava: %v", err)
	}
	if strings.Contains(buf.String(), "will install Temurin") {
		t.Fatalf("Java 25 should not trigger an install:\n%s", buf.String())
	}
	for _, cmd := range runner.Commands {
		if cmd.Name == "apt-get" {
			t.Fatalf("did not expect a package install, ran %v", runner.Commands)
		}
	}
}

func TestInstallJavaReplacesJava21(t *testing.T) {
	t.Parallel()

	runner := &versionedRunner{
		MockRunner: NewMockRunner(),
		outputs: []string{
			"openjdk 21.0.2 2024-01-16\n",
			"openjdk 25.0.1 2025-10-21\n",
		},
	}
	runner.ExistsMap["java"] = true
	runner.OutputMap[runner.key("cat", "/etc/os-release")] = []byte("VERSION_CODENAME=bookworm\n")
	var buf bytes.Buffer
	err := InstallJava(t.Context(), runner, &Platform{PkgMgr: "apt"}, ui.NewWriter(&buf, false))
	if err != nil {
		t.Fatalf("InstallJava: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Java 21", "Temurin 25", "v0.2.0", temurin25URL} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	script := bashScript(runner.Commands)
	for _, want := range []string{
		"https://packages.adoptium.net/artifactory/api/gpg/key/public",
		"deb https://packages.adoptium.net/artifactory/deb bookworm main",
		"temurin-25-jdk",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("Adoptium script missing %q:\n%s", want, script)
		}
	}
}

func TestAdoptiumAptScriptUsesCodename(t *testing.T) {
	t.Parallel()

	script := adoptiumAptScript("bookworm")
	if !strings.Contains(script, "deb https://packages.adoptium.net/artifactory/deb bookworm main") {
		t.Fatalf("script = %s", script)
	}
	if strings.Contains(adoptiumAptScript("noble"), "bookworm") {
		t.Fatal("codename leaked into another script")
	}
}

func TestInstallJavaMissingJavaInstallsTemurin(t *testing.T) {
	t.Parallel()

	runner := &versionedRunner{
		MockRunner: NewMockRunner(),
		outputs:    []string{"openjdk 25.0.1 2025-10-21\n"},
	}
	var buf bytes.Buffer
	err := InstallJava(t.Context(), runner, &Platform{PkgMgr: "brew"}, ui.NewWriter(&buf, false))
	if err != nil {
		t.Fatalf("InstallJava: %v", err)
	}
	if !strings.Contains(buf.String(), "Java was not found on PATH.") {
		t.Fatalf("output = %s", buf.String())
	}
	if !sawCommand(runner.Commands, "brew", "temurin@25") {
		t.Fatalf("expected brew temurin@25, commands: %+v", runner.Commands)
	}
}

func TestInstallJavaFailureIsActionable(t *testing.T) {
	t.Parallel()

	runner := &versionedRunner{
		MockRunner: NewMockRunner(),
		outputs:    []string{"openjdk 21.0.2 2024-01-16\n"},
	}
	runner.ExistsMap["java"] = true
	runner.ErrorMap[runner.key("apt-get", "update", "-qq")] = fmt.Errorf("apt failed")
	var buf bytes.Buffer
	err := InstallJava(t.Context(), runner, &Platform{PkgMgr: "apt"}, ui.NewWriter(&buf, false))
	if err == nil {
		t.Fatal("expected install failure")
	}
	msg := err.Error()
	for _, want := range []string{"Paper 26.3 requires Java 25", "Java 21", temurin25URL, "v0.2.0", previousJavaReleaseURL, "apt failed"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q: %v", want, err)
		}
	}
}

func TestInstallJavaVerificationRejectsLeftoverJava21(t *testing.T) {
	t.Parallel()

	runner := &versionedRunner{
		MockRunner: NewMockRunner(),
		outputs: []string{
			"openjdk 21.0.2 2024-01-16\n",
			"openjdk 21.0.2 2024-01-16\n",
		},
	}
	runner.ExistsMap["java"] = true
	err := InstallJava(t.Context(), runner, &Platform{PkgMgr: "apt"}, ui.NewWriter(&bytes.Buffer{}, false))
	if err == nil {
		t.Fatal("expected verification failure")
	}
	if !strings.Contains(err.Error(), "found Java 21") || !strings.Contains(err.Error(), temurin25URL) {
		t.Fatalf("error = %v", err)
	}
}

func TestSystemJavaMeetsPaperRequirement(t *testing.T) {
	runner := NewOSCommandRunner()
	require := os.Getenv("REQUIRE_JAVA25") == "1"
	if !runner.CommandExists("java") {
		if require {
			t.Fatal("REQUIRE_JAVA25=1 but java is not on PATH; install Eclipse Temurin 25")
		}
		t.Skip("java is not installed")
	}
	ver, err := javaVersion(t.Context(), runner)
	if err != nil {
		if require {
			t.Fatalf("reading java --version: %v", err)
		}
		t.Skipf("java --version: %v", err)
	}
	if require && ver < minJavaVersion {
		t.Fatalf("Paper 26.3 requires Java %d (Eclipse Temurin); java --version reports %d", minJavaVersion, ver)
	}
	if ver < 1 {
		t.Fatalf("java version = %d", ver)
	}
}

type versionedRunner struct {
	*MockRunner
	outputs []string
	n       int
}

func (v *versionedRunner) RunWithOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "java" {
		if v.n >= len(v.outputs) {
			return nil, fmt.Errorf("unexpected java call %v", args)
		}
		out := v.outputs[v.n]
		v.n++
		v.Commands = append(v.Commands, MockCommand{Name: name, Args: args})
		return []byte(out), nil
	}
	return v.MockRunner.RunWithOutput(ctx, name, args...)
}

func bashScript(cmds []MockCommand) string {
	for _, cmd := range cmds {
		if cmd.Name == "bash" {
			for _, arg := range cmd.Args {
				if strings.Contains(arg, "adoptium") {
					return arg
				}
			}
		}
	}
	return ""
}

func sawCommand(cmds []MockCommand, name, arg string) bool {
	for _, cmd := range cmds {
		if cmd.Name != name {
			continue
		}
		for _, a := range cmd.Args {
			if a == arg {
				return true
			}
		}
	}
	return false
}
