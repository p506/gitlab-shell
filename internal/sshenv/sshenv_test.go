package sshenv

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitlab-shell/internal/testhelper"
)

func TestGitProtocolVersion(t *testing.T) {
	cleanup, err := testhelper.Setenv(GitProtocolEnv, "2")
	require.NoError(t, err)
	defer cleanup()

	require.Equal(t, GitProtocolVersion(), "2")
}

func TestIsSSHConnection(t *testing.T) {
	cleanup, err := testhelper.Setenv(SSHConnectionEnv, "127.0.0.1 0")
	require.NoError(t, err)
	defer cleanup()

	require.Equal(t, IsSSHConnection(), true)
}

func TestLocalAddr(t *testing.T) {
	cleanup, err := testhelper.Setenv(SSHConnectionEnv, "127.0.0.1 0")
	require.NoError(t, err)
	defer cleanup()

	require.Equal(t, LocalAddr(), "127.0.0.1")
}

func TestEmptyLocalAddr(t *testing.T) {
	require.Equal(t, LocalAddr(), "")
}

func TestOriginalCommand(t *testing.T) {
	cleanup, err := testhelper.Setenv(SSHOriginalCommandEnv, "git-receive-pack")
	require.NoError(t, err)
	defer cleanup()

	require.Equal(t, OriginalCommand(), "git-receive-pack")
}
