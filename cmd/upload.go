package cmd

import "github.com/spf13/cobra"

func uploadCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:	"upload",
		Short:	"Upload docker image.",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	return cmd
}