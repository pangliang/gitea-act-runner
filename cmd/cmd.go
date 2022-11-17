package cmd

import (
	"context"
	"os"
	"strconv"

	"gitea.com/gitea/act_runner/engine"
	"gitea.com/gitea/act_runner/runtime"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const version = "0.1.5"

type globalArgs struct {
	EnvFile string
}

func Execute(ctx context.Context) {
	// task := runtime.NewTask("gitea", 0, nil, nil)

	var gArgs globalArgs

	// ./act_runner
	rootCmd := &cobra.Command{
		Use:   "act [event name to run]\nIf no event name passed, will default to \"on: push\"",
		Short: "Run GitHub actions locally by specifying the event name (e.g. `push`) or an action name directly.",
		Args:  cobra.MaximumNArgs(1),
		// RunE:         runRoot(ctx, task),
		Version:      version,
		SilenceUsage: true,
	}
	//rootCmd.Flags().BoolP("run", "r", false, "run workflows")
	//rootCmd.Flags().StringP("job", "j", "", "run job")
	//rootCmd.PersistentFlags().StringVarP(&task.Input.ForgeInstance, "forge-instance", "", "github.com", "Forge instance to use.")
	rootCmd.PersistentFlags().StringVarP(&gArgs.EnvFile, "env-file", "", ".env", "Read in a file of environment variables.")

	// ./act_runner register
	var regArgs registerArgs
	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "Register a runner to the server",
		Args:  cobra.MaximumNArgs(0),
		RunE:  runRegister(ctx, &regArgs, gArgs.EnvFile), // must use a pointer to regArgs
	}
	registerCmd.Flags().BoolVar(&regArgs.NoInteractive, "no-interactive", false, "Disable interactive mode")
	registerCmd.Flags().StringVar(&regArgs.InstanceAddr, "instance", "", "Gitea instance address")
	registerCmd.Flags().StringVar(&regArgs.Token, "token", "", "Runner token")
	registerCmd.Flags().StringVar(&regArgs.RunnerName, "name", "", "Runner name")
	registerCmd.Flags().StringVar(&regArgs.Labels, "labels", "", "Runner tags, comma separated")
	rootCmd.AddCommand(registerCmd)

	// ./act_runner daemon
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run as a runner daemon",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runDaemon(ctx, gArgs.EnvFile),
	}
	// add all command
	rootCmd.AddCommand(daemonCmd)

	// hide completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRoot(ctx context.Context, task *runtime.Task) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		jobID, err := cmd.Flags().GetString("job")
		if err != nil {
			return err
		}

		// try to connect to docker daemon
		// if failed, exit with error
		if err := engine.Start(ctx); err != nil {
			log.WithError(err).Fatalln("failed to connect docker daemon engine")
		}

		task.BuildID, _ = strconv.ParseInt(jobID, 10, 64)
		_ = task.Run(ctx, nil)
		return nil
	}
}