package gobotexample_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cryptix/gobotexample"
	"github.com/stretchr/testify/require"
)

func TestStartStop(t *testing.T) {
	r := require.New(t)

	tRepoDir := filepath.Join("testrun", t.Name())
	os.RemoveAll(tRepoDir)
	err := os.MkdirAll(tRepoDir, 0700)
	r.NoError(err)

	// panics if there is a problem, could return an error instead
	gobotexample.Start(tRepoDir)

	time.Sleep(1 * time.Second)

	r.NoError(gobotexample.Stop(), "failed to stop after 1st start")
}
