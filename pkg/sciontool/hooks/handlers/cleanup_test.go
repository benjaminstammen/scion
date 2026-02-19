/*
Copyright 2025 The Scion Authors.
*/

package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ptone/scion-agent/pkg/sciontool/hooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupHandler_RemovesClaudeDebugOnSessionEnd(t *testing.T) {
	home := t.TempDir()
	debugDir := filepath.Join(home, ".claude", "debug")
	require.NoError(t, os.MkdirAll(debugDir, 0755))

	// Create a symlink like Claude Code does (points to container-internal path)
	require.NoError(t, os.Symlink(
		"/home/scion/.claude/debug/b58ceba9.txt",
		filepath.Join(debugDir, "latest"),
	))
	// Create a regular debug file too
	require.NoError(t, os.WriteFile(
		filepath.Join(debugDir, "session.txt"), []byte("debug"), 0644,
	))

	h := &CleanupHandler{Home: home}

	err := h.Handle(&hooks.Event{
		Name: hooks.EventSessionEnd,
	})
	require.NoError(t, err)

	// debug directory should be gone
	_, err = os.Stat(debugDir)
	assert.True(t, os.IsNotExist(err), "debug dir should be removed")

	// .claude directory itself should still exist
	_, err = os.Stat(filepath.Join(home, ".claude"))
	assert.NoError(t, err, ".claude dir should still exist")
}

func TestCleanupHandler_NoOpForNonSessionEnd(t *testing.T) {
	home := t.TempDir()
	debugDir := filepath.Join(home, ".claude", "debug")
	require.NoError(t, os.MkdirAll(debugDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(debugDir, "session.txt"), []byte("debug"), 0644,
	))

	h := &CleanupHandler{Home: home}

	// Other events should not trigger cleanup
	for _, eventName := range []string{
		hooks.EventSessionStart,
		hooks.EventToolStart,
		hooks.EventToolEnd,
		hooks.EventAgentEnd,
	} {
		err := h.Handle(&hooks.Event{
			Name: eventName,
		})
		require.NoError(t, err)
	}

	// debug directory should still exist
	_, err := os.Stat(debugDir)
	assert.NoError(t, err, "debug dir should still exist for non-session-end events")
}

func TestCleanupHandler_NoErrorWhenDebugDirMissing(t *testing.T) {
	home := t.TempDir()

	h := &CleanupHandler{Home: home}

	// Should not error when .claude/debug doesn't exist
	err := h.Handle(&hooks.Event{
		Name: hooks.EventSessionEnd,
	})
	assert.NoError(t, err)
}
