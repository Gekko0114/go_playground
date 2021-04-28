package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/po3rin/post/cmd/postctl/store"
	"github.com/spf13/cobra"
)

var (
	bucket string
)

func init() {
	rootCmd.AddCommand(mediaCmd)
	mediaCmd.Flags().StringVarP(&bucket, "bucket", "b", "", "S3 bucket name")
}

func isSupportedMedia(path string) bool {
	e := filepath.Ext(path)
	return e == ".png" || e == ".jpeg" || e == ".jpg" || e == ".gif"
}

var mediaCmd = &cobra.Command{
	Use: "media",
	Short: "controls media",
	Long: "controls media",
	Run: func(cmd *cobra.Command, args []string) {
		filepath := args[0]
		if filepath == "" {
			fmt.Println("filepath arg is required")
			os.Exit(1)
		}
		if !isSupportedMedia(filepath) {
			fmt.Println("unsupported")
			os.Exit(1)
		}
		resutl, err := store.New(bucket, "media").Upload(filepath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = os.Remove(filepath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(resutl)
	}
}