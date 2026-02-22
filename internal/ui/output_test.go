package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	u := New(true)
	if !u.color {
		t.Error("New(true) should enable color")
	}
	u = New(false)
	if u.color {
		t.Error("New(false) should disable color")
	}
}

func TestColorize_Enabled(t *testing.T) {
	u := New(true)
	got := u.colorize(colorGreen, "hello")
	want := colorGreen + "hello" + colorReset
	if got != want {
		t.Errorf("colorize() = %q, want %q", got, want)
	}
}

func TestColorize_Disabled(t *testing.T) {
	u := New(false)
	got := u.colorize(colorGreen, "hello")
	if got != "hello" {
		t.Errorf("colorize() with color disabled = %q, want %q", got, "hello")
	}
}

func TestBold_Enabled(t *testing.T) {
	u := New(true)
	got := u.Bold("text")
	want := colorBold + "text" + colorReset
	if got != want {
		t.Errorf("Bold() = %q, want %q", got, want)
	}
}

func TestBold_Disabled(t *testing.T) {
	u := New(false)
	if got := u.Bold("text"); got != "text" {
		t.Errorf("Bold() with color disabled = %q, want %q", got, "text")
	}
}

func TestNewWriter(t *testing.T) {
	var buf bytes.Buffer
	u := NewWriter(&buf, false)

	u.Info("hello %s", "world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("Info() output = %q, expected to contain %q", buf.String(), "hello world")
	}

	buf.Reset()
	u.Success("done")
	if !strings.Contains(buf.String(), "done") {
		t.Errorf("Success() output = %q, expected to contain %q", buf.String(), "done")
	}

	buf.Reset()
	u.Warn("oops")
	if !strings.Contains(buf.String(), "oops") {
		t.Errorf("Warn() output = %q, expected to contain %q", buf.String(), "oops")
	}

	buf.Reset()
	u.Step("section")
	if !strings.Contains(buf.String(), "section") {
		t.Errorf("Step() output = %q, expected to contain %q", buf.String(), "section")
	}
}
