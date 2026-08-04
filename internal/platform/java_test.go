package platform

import "testing"

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
