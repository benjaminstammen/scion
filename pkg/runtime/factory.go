package runtime

import (
	"os"
	"os/exec"

	"github.com/ptone/scion-agent/pkg/config"
)

func GetRuntime() Runtime {
	sandbox := os.Getenv("GEMINI_SANDBOX")
	
	if sandbox == "" {
		if settings, err := config.GetGeminiSettings(); err == nil {
			switch v := settings.Tools.Sandbox.(type) {
			case string:
				sandbox = v
			case bool:
				if v {
					sandbox = "true"
				}
			}
		}
	}

	switch sandbox {
	case "container":
		return NewAppleContainerRuntime()
	case "docker":
		return NewDockerRuntime()
	case "true":
		// Fall through to auto-detection
	}

	// Auto-detection: check for available runtimes
	// On macOS, 'container' is often preferred for performance if available,
	// but both are fully supported.
	if _, err := exec.LookPath("container"); err == nil {
		return NewAppleContainerRuntime()
	}

	if _, err := exec.LookPath("docker"); err == nil {
		return NewDockerRuntime()
	}

	// Default return - the caller will handle the error if the binary is missing
	return NewAppleContainerRuntime()
}
