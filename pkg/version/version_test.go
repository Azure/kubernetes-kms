package version

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	BuildDate = "Now"
	BuildVersion = "version"
	GitCommit = "hash"

	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintVersion()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- strings.TrimSpace(buf.String())
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `{"BuildVersion":"version","GitCommit":"hash","BuildDate":"Now"}`
	if !strings.EqualFold(out, expected) {
		t.Fatalf("string doesn't match, expected %s, got %s", expected, out)
	}
}

func TestGetUserAgent(t *testing.T) {
	BuildDate = "Now"
	BuildVersion = "version"
	GitCommit = "hash"

	userAgent := GetUserAgent()
	expectedUserAgent := fmt.Sprintf("k8s-kms-keyvault/version (%s/%s) hash/Now", runtime.GOOS, runtime.GOARCH)
	if !strings.EqualFold(userAgent, expectedUserAgent) {
		t.Fatalf("string doesn't match, expected %s, got %s", expectedUserAgent, userAgent)

	}
}
