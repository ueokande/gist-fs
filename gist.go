package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Client struct {
	Username string
	Password string
}

type Error struct {
	Message string
}

func (c *Client) FetchGists() ([]*Gist, error) {
	const url = "https://api.github.com/gists"

	var gists []*Gist
	err := getJson(url, c.Username, c.Password, &gists)
	if err != nil {
		return nil, err
	}

	for _, gist := range gists {
		for _, file := range gist.Files {
			file.Gist = gist
		}
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
	Gist   *Gist
}

func (c *Client) FetchContent(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Fetched %s", url)

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
