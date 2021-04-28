package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const indexName = "blog"

var (
	endpoint, user, pass string
)

var esSyncCmd = &cobra.Command{
	Use:   "essync",
	Short: "post blog contens to es",
	Long:  "post blog contens to es",
	Run: func(cmd *cobra.Command, args []string) {
		var filepath []string
		var err error

		filepaths, err = allPostFiles(filepath.Join(root, workdir))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		es, err := newEs(user, pass, endpoint)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		ctx := context.Background()
		err = es.syncEsPost(ctx, filepaths)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(esSyncCmd)
	esSyncCmd.Flags().StringVarP(&root, "root", "r", "", "Git repository root")
	esSyncCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "http://localhost:9200", "Elasticsearch Endpoint")
	esSyncCmd.Flags().StringVarP(&user, "user", "u", "", "Elasticserach User")
	esSyncCmd.Flags().StringVarP(&pass, "pass", "p", "", "Elasticsearch Pass")
}

type es struct {
	client *elastic.Client
}

func newEs(user, pass string, urls ...string) (*es, error) {
	var sniff bool
	if len(urls) > 1 {
		sniff = true
	}

	var client *elastic.Client
	var err error
	operation := func() error {
		client, err = elastic.NewClient(
			elastic.SetURL(urls...),
			elastic.SetSniff(sniff),
			elastic.SetBasicAuth(user, pass),
		)

		if err != nil {
			return errors.Wrap(err, "new Elasticsearch Client with retry")
		}
		return nil
	}

	err = backoff.Retry(operation, backoff.WithMaxRetries(
		backoff.NewExponentialBackoff(),
		10,
	))
	if err != nil {
		return nil, errors.Wrap(err, "new Elasticsearch client")
	}
	return &es{client: client}, nil
}

func (e *es) syncEsPost(ctx context.Context, files []string) error {
	for _, f := range files {
		source, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		m, err := mdMeta(source)
		if err != nil {
			return err
		}
		if m.draft {
			fmt.Printf("passed draft: %+v", m.id)
			continue
		}

		req := request{
			ID:          m.id,
			Title:       m.title,
			Body:        string(source),
			Description: m.description,
			Cover:       m.cover,
			Tags:        m.tags,
			CreatedAt:   m.date,
			UpdatedAt:   time.Now(),
			IsExternal:  m.isExternal,
			ExternalURL: m.externalURL,
		}

		_, err = e.client.Index().
			Index(indexName).
			ID(req.ID).
			BodyJson(req).
			Do(ctx)
		if err != nil {
			return err
		}
		log.Infof("sync: %+v", m.id)
	}
	return nil
}
