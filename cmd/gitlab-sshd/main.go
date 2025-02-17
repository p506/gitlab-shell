package main

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-shell/internal/config"
	"gitlab.com/gitlab-org/gitlab-shell/internal/logger"
	"gitlab.com/gitlab-org/gitlab-shell/internal/sshd"
	"gitlab.com/gitlab-org/labkit/monitoring"
)

var (
	configDir = flag.String("config-dir", "", "The directory the config is in")

	// BuildTime signifies the time the binary was build.
	BuildTime = "2021-02-16T09:28:07+01:00" // Set at build time in the Makefile
	// Version is the current version of GitLab Shell sshd.
	Version = "(unknown version)" // Set at build time in the Makefile
)

func overrideConfigFromEnvironment(cfg *config.Config) {
	if gitlabUrl := os.Getenv("GITLAB_URL"); gitlabUrl != "" {
		cfg.GitlabUrl = gitlabUrl
	}
	if gitlabTracing := os.Getenv("GITLAB_TRACING"); gitlabTracing != "" {
		cfg.GitlabTracing = gitlabTracing
	}
	if gitlabShellSecret := os.Getenv("GITLAB_SHELL_SECRET"); gitlabShellSecret != "" {
		cfg.Secret = gitlabShellSecret
	}
	if gitlabLogFormat := os.Getenv("GITLAB_LOG_FORMAT"); gitlabLogFormat != "" {
		cfg.LogFormat = gitlabLogFormat
	}
	return
}

func main() {
	flag.Parse()
	cfg := new(config.Config)
	if *configDir != "" {
		var err error
		cfg, err = config.NewFromDir(*configDir)
		if err != nil {
			log.Fatalf("failed to load configuration from specified directory: %v", err)
		}
	}
	overrideConfigFromEnvironment(cfg)
	if err := cfg.IsSane(); err != nil {
		if *configDir == "" {
			log.Warn("note: no config-dir provided, using only environment variables")
		}
		log.Fatalf("configuration error: %v", err)
	}
	logger.ConfigureStandalone(cfg)

	// Startup monitoring endpoint.
	if cfg.Server.WebListen != "" {
		go func() {
			log.Fatal(
				monitoring.Start(
					monitoring.WithListenerAddress(cfg.Server.WebListen),
					monitoring.WithBuildInformation(Version, BuildTime),
				),
			)
		}()
	}

	if err := sshd.Run(cfg); err != nil {
		log.Fatalf("Failed to start GitLab built-in sshd: %v", err)
	}
}
