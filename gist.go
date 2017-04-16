package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Client struct {
	RestClient
}

func NewClient(username, password string) *Client {
	return &Client{
		RestClient{
			Username: username,
			Password: password,
		},
	}
}

type Error struct {
	Message string
}

func (c *Client) FetchGists() ([]*Gist, error) {
	const url = "https://api.github.com/gists"

	log.Printf("GET %s\n", url)

	var gists []*Gist
	err := c.Get(url, &gists)
	if err != nil {
		return nil, err
	}

	log.Printf("Fetched %d gists", len(gists))

	return gists, nil
}

type Gist struct {
	Id          string
	Description string
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Public      bool
	Files       map[string]*GistFile
}

type GistFile struct {
	Size   uint64
	RawUrl string `json:"raw_url"`
}

type EditGistForm struct {
	Description *string
	Files       map[string]*struct {
		Filename *string
		Content  *string
	}
}

func (c *Client) FetchContent(url string) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	log.Printf("GET %s\n", url)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (c *Client) UpdateContent(id string, name string, content string) error {
	url := "https://api.github.com/gists/" + id
	form := EditGistForm{
		Files: make(map[string]*struct {
			Filename *string
			Content  *string
		}),
	}
	form.Files[name] = &struct {
		Filename *string
		Content  *string
	}{
		Content: &content,
	}

	log.Printf("PATCH %s\n", url)
	return c.Patch(url, &form, nil)
}
