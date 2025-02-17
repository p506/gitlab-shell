package uploadarchive

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-shell/client/testserver"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/commandargs"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/readwriter"
	"gitlab.com/gitlab-org/gitlab-shell/internal/config"
	"gitlab.com/gitlab-org/gitlab-shell/internal/testhelper"
	"gitlab.com/gitlab-org/gitlab-shell/internal/testhelper/requesthandlers"
)

func TestUploadPack(t *testing.T) {
	gitalyAddress, _ := testserver.StartGitalyServer(t)

	requests := requesthandlers.BuildAllowedWithGitalyHandlers(t, gitalyAddress)
	url := testserver.StartHttpServer(t, requests)

	output := &bytes.Buffer{}
	input := &bytes.Buffer{}

	userId := "1"
	repo := "group/repo"

	cmd := &Command{
		Config:     &config.Config{GitlabUrl: url},
		Args:       &commandargs.Shell{GitlabKeyId: userId, CommandType: commandargs.UploadArchive, SshArgs: []string{"git-upload-archive", repo}},
		ReadWriter: &readwriter.ReadWriter{ErrOut: output, Out: output, In: input},
	}

	hook := testhelper.SetupLogger()

	err := cmd.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, "UploadArchive: "+repo, output.String())

	require.True(t, testhelper.WaitForLogEvent(hook))
	entries := hook.AllEntries()
	require.Equal(t, 2, len(entries))
	require.Equal(t, logrus.InfoLevel, entries[1].Level)
	require.Contains(t, entries[1].Message, "executing git command")
	require.Contains(t, entries[1].Message, "command=git-upload-archive")
	require.Contains(t, entries[1].Message, "gl_key_type=key")
	require.Contains(t, entries[1].Message, "gl_key_id=123")
}
