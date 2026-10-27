package unrelated

import (
	"fmt"
	"os"
	"testing"
)

// DB の TestMain と同じ初期化境界が選択外でも実行されるかを観測する。
func TestMain(m *testing.M) {
	if err := recordEvent("TestMain"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

func TestUnrelatedFirst(t *testing.T) {
	recordRun(t)
}

func TestUnrelatedSecond(t *testing.T) {
	recordRun(t)
}

func TestUnrelatedThird(t *testing.T) {
	recordRun(t)
}

func recordRun(t *testing.T) {
	t.Helper()
	if err := recordEvent(t.Name()); err != nil {
		t.Fatal(err)
	}
}

func recordEvent(name string) error {
	path := os.Getenv("MUTATION_MAPPING_LOG")
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(name + "\n")
	return err
}
