package store

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

const (
	storeRaceDataDirEnv = "CENCLI_STORE_RACE_DATA_DIR"
	storeRaceGateEnv    = "CENCLI_STORE_RACE_GATE"
	storeRaceWorkerEnv  = "CENCLI_STORE_RACE_WORKER"
)

// TestNewConcurrentProcesses verifies that independent CLI processes can open
// and initialize the same database concurrently.
func TestNewConcurrentProcesses(t *testing.T) {
	const processes = 20

	dataDir := t.TempDir()
	gatePath := filepath.Join(t.TempDir(), "start")

	type worker struct {
		cmd    *exec.Cmd
		output bytes.Buffer
	}
	workers := make([]worker, processes)

	for i := range workers {
		workerID := strconv.Itoa(i)
		workers[i].cmd = exec.Command(os.Args[0], "-test.run=^TestNewConcurrentProcessWorker$")
		workers[i].cmd.Env = append(os.Environ(),
			storeRaceDataDirEnv+"="+dataDir,
			storeRaceGateEnv+"="+gatePath,
			storeRaceWorkerEnv+"="+workerID,
		)
		workers[i].cmd.Stdout = &workers[i].output
		workers[i].cmd.Stderr = &workers[i].output
		if err := workers[i].cmd.Start(); err != nil {
			t.Fatalf("start worker %d: %v", i, err)
		}
	}

	readyDeadline := time.Now().Add(10 * time.Second)
	for {
		ready := 0
		for i := range workers {
			if _, err := os.Stat(gatePath + ".ready-" + strconv.Itoa(i)); err == nil {
				ready++
			}
		}
		if ready == processes {
			break
		}
		if time.Now().After(readyDeadline) {
			t.Fatalf("only %d of %d workers became ready", ready, processes)
		}
		time.Sleep(time.Millisecond)
	}

	if err := os.WriteFile(gatePath, nil, 0o600); err != nil {
		t.Fatalf("release workers: %v", err)
	}

	for i := range workers {
		if err := workers[i].cmd.Wait(); err != nil {
			t.Errorf("worker %d failed: %v\n%s", i, err, workers[i].output.String())
		}
	}
}

func TestNewConcurrentProcessWorker(t *testing.T) {
	dataDir := os.Getenv(storeRaceDataDirEnv)
	gatePath := os.Getenv(storeRaceGateEnv)
	workerID := os.Getenv(storeRaceWorkerEnv)
	if dataDir == "" || gatePath == "" || workerID == "" {
		t.Skip("helper process")
	}

	readyPath := fmt.Sprintf("%s.ready-%s", gatePath, workerID)
	if err := os.WriteFile(readyPath, nil, 0o600); err != nil {
		t.Fatalf("signal readiness: %v", err)
	}

	gateDeadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(gatePath); err == nil {
			break
		}
		if time.Now().After(gateDeadline) {
			t.Fatal("timed out waiting for start gate")
		}
		time.Sleep(time.Millisecond)
	}

	if _, err := New(dataDir); err != nil {
		t.Fatalf("New() failed: %v", err)
	}
}
