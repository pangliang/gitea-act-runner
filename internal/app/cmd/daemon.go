// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"gitea.com/gitea/act_runner/internal/app/poll"
	"gitea.com/gitea/act_runner/internal/app/run"
	"gitea.com/gitea/act_runner/internal/pkg/client"
	"gitea.com/gitea/act_runner/internal/pkg/config"
	"gitea.com/gitea/act_runner/internal/pkg/envcheck"
	"gitea.com/gitea/act_runner/internal/pkg/labels"
	"gitea.com/gitea/act_runner/internal/pkg/ver"
)

func runDaemon(ctx context.Context, configFile *string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log.Infoln("Starting runner daemon")

		cfg, err := config.LoadDefault(*configFile)
		if err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		initLogging(cfg)

		reg, err := config.LoadRegistration(cfg.Runner.File)
		if os.IsNotExist(err) {
			log.Error("registration file not found, please register the runner first")
			return err
		} else if err != nil {
			return fmt.Errorf("failed to load registration file: %w", err)
		}

		ls := labels.Labels{}
		for _, l := range reg.Labels {
			label, err := labels.Parse(l)
			if err != nil {
				log.WithError(err).Warnf("ignored invalid label %q", l)
				continue
			}
			ls = append(ls, label)
		}
		if len(ls) == 0 {
			log.Warn("no labels configured, runner may not be able to pick up jobs")
		}

		if ls.RequireDocker() {
			if err := envcheck.CheckIfDockerRunning(ctx); err != nil {
				return err
			}
		}

		cli := client.New(
			reg.Address,
			cfg.Runner.Insecure,
			reg.UUID,
			reg.Token,
			ver.Version(),
		)

		runner := run.NewRunner(cfg, reg, cli)
		poller := poll.New(cfg, cli, runner)

		poller.Poll(ctx)

		return nil
	}
}

// initLogging setup the global logrus logger.
func initLogging(cfg *config.Config) {
	isTerm := isatty.IsTerminal(os.Stdout.Fd())
	log.SetFormatter(&log.TextFormatter{
		DisableColors: !isTerm,
		FullTimestamp: true,
	})

	if l := cfg.Log.Level; l != "" {
		level, err := log.ParseLevel(l)
		if err != nil {
			log.WithError(err).
				Errorf("invalid log level: %q", l)
		}
		if log.GetLevel() != level {
			log.Infof("log level changed to %v", level)
			log.SetLevel(level)
		}
	}
}
