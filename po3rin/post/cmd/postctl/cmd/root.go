package cmd

import (
	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "postctl",
	Short: "postctl controls posts for blog",
	Long:  "postctl controls posts for blog",
}

var workdir string

func init() {
	rootCmd.PersistantFlags().StringVarP(&workdir, "dir", "d", "posts", "workspace directory managing blog post")
}
