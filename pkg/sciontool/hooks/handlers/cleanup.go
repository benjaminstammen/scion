/*
Copyright 2025 The Scion Authors.
*/

package handlers

import (
	"os"
	"path/filepath"

	"github.com/ptone/scion-agent/pkg/sciontool/hooks"
	"github.com/ptone/scion-agent/pkg/sciontool/log"
)

// CleanupHandler removes harness artifacts that would cause slow deletion
// on the host (e.g., dangling symlinks that trigger macOS autofs).
type CleanupHandler struct {
	// Home is the agent's home directory inside the container.
	Home string
}

// NewCleanupHandler creates a new cleanup handler using $HOME.
func NewCleanupHandler() *CleanupHandler {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/scion"
	}
	return &CleanupHandler{Home: home}
}

// Handle processes an event and performs cleanup on session end.
func (h *CleanupHandler) Handle(event *hooks.Event) error {
	if event.Name != hooks.EventSessionEnd {
		return nil
	}

	h.cleanupClaudeDebug()

	return nil
}

// cleanupClaudeDebug removes the .claude/debug directory if it exists.
// Claude Code creates symlinks in this directory that point to
// container-internal paths (e.g., /home/scion/.claude/debug/xxx.txt).
// When the worktree is deleted on the host, these dangling symlinks
// trigger macOS autofs resolution, causing multi-second stalls.
// Removing them from inside the container avoids this entirely.
func (h *CleanupHandler) cleanupClaudeDebug() {
	debugDir := filepath.Join(h.Home, ".claude", "debug")
	if _, err := os.Stat(debugDir); os.IsNotExist(err) {
		log.Debug("Claude debug dir %s does not exist, skipping cleanup", debugDir)
		return
	}
	log.Debug("Removing Claude debug dir %s", debugDir)
	if err := os.RemoveAll(debugDir); err != nil {
		log.Error("Failed to clean up %s: %v", debugDir, err)
	}
}
