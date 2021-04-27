package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	time string
)

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&timef, "time", "t", "", "new post date time (format: 2006-01-02)")
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "generate new post file",
	Long:  "generate new post file",
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		if id == "" {
			fmt.Println("id is required")
			os.Exit(1)
		}

		t := time.Now()
		year := strconv.Itoa(t.Year())
		dirpath := filepath.Join(workdir, year)

		err := createDir(dirpath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		mdname := id + ".md"
		p := filepath.Join(dirpath, mdname)

		f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()

		template := fmt.Sprintf(`---
title: Try Go
cover: img/gopher.png
date: %+v
id: %s
description: Go is a programming language
tags:
    - golang
    - markdown
---

## Overview
`, t.Format("2006/01/02"), id)
		_, err = f.Write([]byte(template))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
