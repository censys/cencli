package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestRaceConditionEndToEnd reproduces the real file-level race condition that
// occurs when multiple cencli processes initialize config concurrently against
// the same data directory. Each subprocess is a real OS process with its own
// viper instance — just like production. The race is in the non-atomic
// read-modify-write of config.yaml (viper.WriteConfig + addDocCommentsToYAML).
//
// Run with: go test -run TestRaceConditionEndToEnd -count=1 -v ./internal/config/
func TestRaceConditionEndToEnd(t *testing.T) {
	const processes = 15

	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(dataDir, "config.yaml")

	type procResult struct {
		id       int
		exitCode int
		output   string
		err      error
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []procResult
	)

	// All processes start as close together as possible.
	start := make(chan struct{})

	for i := 0; i < processes; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start

			cmd := exec.Command(
				os.Args[0],
				"-test.run=^TestRaceWorker$",
				"-test.v",
			)
			cmd.Env = append(os.Environ(),
				"RACE_WORKER=1",
				fmt.Sprintf("RACE_DATA_DIR=%s", dataDir),
			)

			out, err := cmd.CombinedOutput()

			exitCode := 0
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					exitCode = ee.ExitCode()
				} else {
					exitCode = -1
				}
			}

			mu.Lock()
			results = append(results, procResult{
				id:       id,
				exitCode: exitCode,
				output:   string(out),
				err:      err,
			})
			mu.Unlock()
		}(i)
	}

	close(start)
	wg.Wait()

	// Tally process-level failures.
	var processErrors int
	for _, r := range results {
		if r.exitCode != 0 {
			processErrors++
			t.Logf("process %d exited %d:\n%s", r.id, r.exitCode, r.output)
		}
	}

	// Check the final state of config.yaml — the file all processes raced on.
	finalRaw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("cannot read final config.yaml: %v", err)
	}

	var (
		fileEmpty   bool
		fileCorrupt bool
		yamlErr     string
	)

	if len(finalRaw) == 0 {
		fileEmpty = true
	} else {
		var parsed map[string]interface{}
		if err := yaml.Unmarshal(finalRaw, &parsed); err != nil {
			fileCorrupt = true
			yamlErr = err.Error()
		}
	}

	t.Logf("--- Race Condition Results ---")
	t.Logf("  Processes launched:   %d", processes)
	t.Logf("  Process failures:     %d", processErrors)
	t.Logf("  Final file empty:     %v", fileEmpty)
	t.Logf("  Final file corrupt:   %v", fileCorrupt)
	if fileCorrupt {
		t.Logf("  YAML error:           %s", yamlErr)
		t.Logf("  File content:\n%s", finalRaw)
	}

	if processErrors > 0 || fileEmpty || fileCorrupt {
		t.Errorf("Race condition reproduced: processes_failed=%d file_empty=%v file_corrupt=%v\n"+
			"Multiple processes doing read-modify-write on config.yaml without file locking\n"+
			"causes corruption visible to concurrent or subsequent CLI invocations.",
			processErrors, fileEmpty, fileCorrupt)
	}
}

// TestRaceWorker is a subprocess helper that calls config.New() once against a
// shared data directory. Each invocation is a separate OS process — exactly
// like a real cencli CLI invocation. Only runs when spawned by the parent test.
func TestRaceWorker(t *testing.T) {
	if os.Getenv("RACE_WORKER") != "1" {
		t.Skip("skipping: only runs as subprocess of TestRaceConditionEndToEnd")
	}

	dataDir := os.Getenv("RACE_DATA_DIR")
	if dataDir == "" {
		t.Fatal("RACE_DATA_DIR not set")
	}

	cfg, cErr := New(dataDir)
	if cErr != nil {
		t.Fatalf("New() failed: %v", cErr)
	}

	// Verify the returned config is sane.
	if cfg.OutputFormat == "" {
		t.Error("config has empty output-format")
	}

	// Verify the file on disk is valid YAML right after our write.
	configPath := filepath.Join(dataDir, "config.yaml")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("cannot read config.yaml after New(): %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("config.yaml is empty immediately after New()")
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("config.yaml is corrupted after New(): %v", err)
	}

	// Check for partial writes — key fields should be present.
	requiredKeys := []string{"output-format", "streaming", "timeouts", "retry-strategy"}
	content := string(raw)
	for _, key := range requiredKeys {
		if !strings.Contains(content, key+":") {
			t.Errorf("config.yaml missing expected key %q — possible truncated write", key)
		}
	}
}
