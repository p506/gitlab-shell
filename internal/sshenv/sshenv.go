package sshenv

import (
	"os"
	"strings"
)

const (
	// GitProtocolEnv defines the ENV name holding the git protocol used
	GitProtocolEnv = "GIT_PROTOCOL"
	// SSHConnectionEnv defines the ENV holding the SSH connection
	SSHConnectionEnv = "SSH_CONNECTION"
	// SSHOriginalCommandEnv defines the ENV containing the original SSH command
	SSHOriginalCommandEnv = "SSH_ORIGINAL_COMMAND"
)

// GitProtocolVersion returns the git protocol version
func GitProtocolVersion() string {
	return os.Getenv(GitProtocolEnv)
}

// IsSSHConnection returns true if `SSH_CONNECTION` is set
func IsSSHConnection() bool {
	ok := os.Getenv(SSHConnectionEnv)
	return ok != ""
}

// LocalAddr returns the connection address from ENV string
func LocalAddr() string {
	address := os.Getenv(SSHConnectionEnv)

	if address != "" {
		return strings.Fields(address)[0]
	}
	return ""
}

// OriginalCommand returns the original SSH command
func OriginalCommand() string {
	return os.Getenv(SSHOriginalCommandEnv)
}
