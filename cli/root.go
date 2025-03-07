package cli

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regnotify",
		Short: "`regnotify`",
		Long:  "`regnotify`",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	cmd.AddCommand(newServeCommand())

	return cmd
}
