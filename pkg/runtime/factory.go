package runtime

import (
	"os"
	"os/exec"
)

func GetRuntime() Runtime {
	sandbox := os.Getenv("GEMINI_SANDBOX")
	switch sandbox {
	case "container":
		return NewAppleContainerRuntime()
	case "docker":
		return NewDockerRuntime()
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
