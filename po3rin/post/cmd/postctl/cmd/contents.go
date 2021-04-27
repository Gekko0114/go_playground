package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var contentsCmd = &cobra.Command{
	Use:   "contents",
	Short: "manages blog contents",
	Long:  "manages blog contents",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := NewContents(prefix)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = filepath.Walk(workdir, c.Walk)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		var w io.Writer
		if lint {
			w = bytes.NewBuffer(nil)
		} else {
			f, err := os.Create(output)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			defer f.Close()
			w = f
		}
		err = c.Write(w)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if !lint {
			fmt.Println("done")
			return
		}

		buf, ok := w.(*bytes.Buffer)
		if !ok {
			fmt.Println("failed to type assertion io.Writer to *bytes.Buffer")
			os.Exit(1)
		}

		body, err := ioutil.ReadFile(output)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if !bytes.Equal(body, buf.Bytes()) {
			fmt.Println("mismatch")
			os.Exit(1)
		}
		fmt.Println("ok")
	},
}

var (
	output, prefix string
	lint           bool
)

func init() {
	rootCmd.AddCommand(contentsCmd)
	contentsCmd.Flags().StringVarP(&output, "out", "o", "CONTENTS.md", "output file path (default ./CONTENTS.md)")
	contentsCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "link prefix")
	contentsCmd.Flags().BoolVarP(&lint, "lint", "l", false, "contents table diff linter")
}

type Post struct {
	title string
	url   string
	year  string
}

type Posts []Post

func (p Posts) Len() int {
	return len(p)
}

func (p Posts) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Posts) Less(i, j int) {
	return p[i].url > p[j].url
}

type Contents struct {
	prefix string
	Posts  Posts
}

func NewContents(prefix string) (*Contents, error) {
	return &Contents{
		prefix: prefix,
		Posts:  make(Posts, 0),
	}, nil
}

func (c *Contents) Walk(path string, info os.FileInfo, err error) error {
	if !isPostMdfile(path) {
		return nil
	}

	md, err := ioutil.ReadFile(path)
	if err != nil {
		return Wrapf(err, "contents: read file in %v", path)
	}

	meta, err := mdMeta(md)
	if err != nil {
		return errors.Wrapf(err, "content: get meta in %v", path)
	}

	if meta.draft {
		return nil
	}

	url := c.prefix + "/" + path
	year := strings.Split(path, "/")[1]
	c.Posts = append(c.Posts, Post{meta.title, url, year})
	return nil
}

func (c *Contents) Write(w io.Writer) error {
	_, err := w.Write([]byte("# Blog post\n\n"))
	if err != nil {
		return errors.Wrap(err, "contents: write header")
	}

	posts := c.Posts
	sort.Sort(posts)

	var tmpYear string
	for _, p := range posts {
		if tmpYear != p.year {
			_, err = w.Write([]byte(fmt.Sprintf("## %s\n\n", p.year)))
			if err != nil {
				return errors.Wrap(err, "contents: write header")
			}
			tmpYear = p.year
		}

		_, err = w.Write([]byte(fmt.Sprintf("[%s](%s)\n\n", p.title, p.url)))
		if err != nil {
			return errors.Wrap(err, "contents: write header")
		}
	}
	return nil
}
