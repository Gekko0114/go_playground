package chrbm

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

type Bookmark struct {
	Name string
	Path string
	URL  string
}

type BookmarkTree struct {
	Roots Roots `json:"roots"`
}

type Roots struct {
	//each type of bookmark_files
	BookmarkBar Node `json:"bookmark_bar"`
	Other       Node `json:"other"`
	Synced      Node `json:"synced"`
}

type Node struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Children []Node `json:"children"`
}

type Visitor interface {
	Visit(n Node, path string) error
}

type bookmarkRecorder struct {
	data []Bookmark
}

func newBookmarkRecorder() *bookmarkRecorder {
	return &bookmarkRecorder{
		data: make([]Bookmark, 0, 100),
	}
}

func (b *bookmarkRecorder) Visit(n Node, path string) error {
	bm := Bookmark{
		Name: n.Name,
		Path: path,
		URL:  n.URL,
	}
	//add URL (not folder)
	b.data = append(b.data, bm)
	return nil
}

func WalkEdge(n Node, visitor Visitor) error {
	return walkEdge(n, "/", visitor)
}

func walkEdge(n Node, path string, v Visitor) error {
	switch n.Type {
	case "folder":
		for _, c := range n.Children {
			p := filepath.Join(path, c.Name)
			// recursive function
			err := walkEdge(c, p, v)
			if err != nil {
				return err
			}
		}
	case "url":
		// append URL
		v.Visit(n, path)
	case "":
		return nil
	default:
		return fmt.Errorf("unsupported type: %+v", n.Type)
	}
	return nil
}

func ListBookmarks(byteJSON []byte) ([]Bookmark, error) {
	//check whether byteJSON is JSON file
	if !json.Valid(byteJSON) {
		return nil, errors.New("input is invalid format (required json only)")
	}

	var m BookmarkTree
	err := json.Unmarshal(byteJSON, &m)
	if err != nil {
		return nil, err
	}
	//get Recorder initialization
	r := newBookmarkRecorder()

	//goroutine, parallel check
	var eg errgroup.Group
	eg.Go(func() error {
		return WalkEdge(m.Roots.Other, r)
	})
	eg.Go(func() error {
		return WalkEdge(m.Roots.Synced, r)
	})
	eg.Go(func() error {
		return WalkEdge(m.Roots.BookmarkBar, r)
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return r.data, nil
}
