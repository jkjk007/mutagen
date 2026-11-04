package agent

import (
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// Install installs the current binary to the appropriate location for an agent
// binary with the current Mutagen version.
func Install() error {
	// Compute the destination.
	destination, err := installPath()
	if err != nil {
		return fmt.Errorf("unable to compute agent destination: %w", err)
	}

	// Compute the path to the current executable.
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("unable to determine executable path: %w", err)
	}

	// Relocate the current executable to the installation path.
	if err = filesystem.Rename(nil, executablePath, nil, destination, true); err != nil {
		return fmt.Errorf("unable to relocate agent executable: %w", err)
	}

	// Success.
	return nil
}

// install attempts to probe an endpoint and install the appropriate agent
// binary over the specified transport.
func install(logger *logging.Logger, transport Transport, prompter string) error {
	// Detect the target platform.
	goos, goarch, posix, err := probe(transport, prompter)
	if err != nil {
		return fmt.Errorf("unable to probe remote platform: %w", err)
	}

	// Find the appropriate agent binary. Ensure that it's cleaned up when we're
	// done with it.
	if err := prompting.Message(prompter, "Extracting agent..."); err != nil {
		return fmt.Errorf("unable to message prompter: %w", err)
	}
	agentExecutable, err := ExecutableForPlatform(goos, goarch, "")
	if err != nil {
		return fmt.Errorf("unable to get agent for platform: %w", err)
	}
	defer os.Remove(agentExecutable)

	// Copy the agent to the remote. We use a unique identifier for the
	// temporary destination. For Windows remotes, we add a ".exe" suffix, which
	// will automatically make the file executable on the remote (POSIX systems
	// are handled separately below). For POSIX systems, we add a dot prefix to
	// hide the executable.
	if err := prompting.Message(prompter, "Copying agent..."); err != nil {
		return fmt.Errorf("unable to message prompter: %w", err)
	}
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("unable to generate UUID for agent copying: %w", err)
	}
	destination := BaseName + randomUUID.String()
	if goos == "windows" {
		destination += ".exe"
	}
	if posix {
		destination = "." + destination
	}
	if err = transport.Copy(agentExecutable, destination); err != nil {
		return fmt.Errorf("unable to copy agent binary: %w", err)
	}

	// Ensure that the copied agent binary is executable on POSIX remotes. This
	// is required because the copy operation does not preserve the executability
	// bit: it is lost when copying from Windows (which has no such bit), and it
	// can also be cleared by the remote's umask when the file is created. Note
	// that we do this unconditionally rather than only for Windows hosts (the
	// assumption that a POSIX host preserves the bit during copy does not hold
	// for all transports and remote configurations), and that it's a no-op on
	// Windows POSIX remotes.
	if posix {
		if err := prompting.Message(prompter, "Setting agent executability..."); err != nil {
			return fmt.Errorf("unable to message prompter: %w", err)
		}
		executabilityCommand := fmt.Sprintf("chmod +x %s", destination)
		if err := run(transport, executabilityCommand); err != nil {
			return fmt.Errorf("unable to set agent executability: %w", err)
		}
	}

	// Invoke the remote installation.
	if err := prompting.Message(prompter, "Installing agent..."); err != nil {
		return fmt.Errorf("unable to message prompter: %w", err)
	}
	var installCommand string
	if posix {
		installCommand = fmt.Sprintf("./%s %s", destination, CommandInstall)
	} else {
		installCommand = fmt.Sprintf("%s %s", destination, CommandInstall)
	}
	if err := run(transport, installCommand); err != nil {
		return fmt.Errorf("unable to invoke agent installation: %w", err)
	}

	// Success.
	return nil
}
