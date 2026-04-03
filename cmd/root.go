package cmd

import "github.com/spf13/cobra"

func Execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "nuon-ext-webhook-server",
		Short:         "Webhook receiver for Nuon flow step lifecycle events",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newServeCommand())

	return root
}
