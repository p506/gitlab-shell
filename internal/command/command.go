package command

import (
	"context"
	"os"

	"gitlab.com/gitlab-org/gitlab-shell/internal/command/authorizedkeys"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/authorizedprincipals"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/commandargs"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/discover"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/healthcheck"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/lfsauthenticate"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/personalaccesstoken"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/readwriter"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/receivepack"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/shared/disallowedcommand"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/twofactorrecover"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/twofactorverify"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/uploadarchive"
	"gitlab.com/gitlab-org/gitlab-shell/internal/command/uploadpack"
	"gitlab.com/gitlab-org/gitlab-shell/internal/config"
	"gitlab.com/gitlab-org/gitlab-shell/internal/executable"
	"gitlab.com/gitlab-org/gitlab-shell/internal/sshenv"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"
	"gitlab.com/gitlab-org/labkit/tracing"
)

type Command interface {
	Execute(ctx context.Context) error
}

func New(e *executable.Executable, arguments []string, env sshenv.Env, config *config.Config, readWriter *readwriter.ReadWriter) (Command, error) {
	args, err := commandargs.Parse(e, arguments, env)
	if err != nil {
		return nil, err
	}

	if cmd := buildCommand(e, args, env, config, readWriter); cmd != nil {
		if config.SslCertDir != "" {
			os.Setenv("SSL_CERT_DIR", config.SslCertDir)
		}

		return cmd, nil
	}

	return nil, disallowedcommand.Error
}

// ContextWithCorrelationID() will always return a background Context
// with a correlation ID.  It will first attempt to extract the ID from
// an environment variable. If is not available, a random one will be
// generated.
func ContextWithCorrelationID() (context.Context, func()) {
	ctx, finished := tracing.ExtractFromEnv(context.Background())
	defer finished()

	correlationID := correlation.ExtractFromContext(ctx)
	if correlationID == "" {
		correlationID, err := correlation.RandomID()
		if err != nil {
			log.WithError(err).Warn("unable to generate correlation ID")
		} else {
			ctx = correlation.ContextWithCorrelation(ctx, correlationID)
		}
	}

	return ctx, finished
}

func buildCommand(e *executable.Executable, args commandargs.CommandArgs, env sshenv.Env, config *config.Config, readWriter *readwriter.ReadWriter) Command {
	switch e.Name {
	case executable.GitlabShell:
		return BuildShellCommand(args.(*commandargs.Shell), env, config, readWriter)
	case executable.AuthorizedKeysCheck:
		return buildAuthorizedKeysCommand(args.(*commandargs.AuthorizedKeys), config, readWriter)
	case executable.AuthorizedPrincipalsCheck:
		return buildAuthorizedPrincipalsCommand(args.(*commandargs.AuthorizedPrincipals), config, readWriter)
	case executable.Healthcheck:
		return buildHealthcheckCommand(config, readWriter)
	}

	return nil
}

func BuildShellCommand(args *commandargs.Shell, env sshenv.Env, config *config.Config, readWriter *readwriter.ReadWriter) Command {
	switch args.CommandType {
	case commandargs.Discover:
		return &discover.Command{Config: config, Args: args, ReadWriter: readWriter}
	case commandargs.TwoFactorRecover:
		return &twofactorrecover.Command{Config: config, Args: args, ReadWriter: readWriter}
	case commandargs.TwoFactorVerify:
		return &twofactorverify.Command{Config: config, Args: args, ReadWriter: readWriter}
	case commandargs.LfsAuthenticate:
		return &lfsauthenticate.Command{Config: config, Args: args, ReadWriter: readWriter}
	case commandargs.ReceivePack:
		return &receivepack.Command{Config: config, Env: env, Args: args, ReadWriter: readWriter}
	case commandargs.UploadPack:
		return &uploadpack.Command{Config: config, Env: env, Args: args, ReadWriter: readWriter}
	case commandargs.UploadArchive:
		return &uploadarchive.Command{Config: config, Args: args, ReadWriter: readWriter}
	case commandargs.PersonalAccessToken:
		return &personalaccesstoken.Command{Config: config, Args: args, ReadWriter: readWriter}
	}

	return nil
}

func buildAuthorizedKeysCommand(args *commandargs.AuthorizedKeys, config *config.Config, readWriter *readwriter.ReadWriter) Command {
	return &authorizedkeys.Command{Config: config, Args: args, ReadWriter: readWriter}
}

func buildAuthorizedPrincipalsCommand(args *commandargs.AuthorizedPrincipals, config *config.Config, readWriter *readwriter.ReadWriter) Command {
	return &authorizedprincipals.Command{Config: config, Args: args, ReadWriter: readWriter}
}

func buildHealthcheckCommand(config *config.Config, readWriter *readwriter.ReadWriter) Command {
	return &healthcheck.Command{Config: config, ReadWriter: readWriter}
}
