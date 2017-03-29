package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type User struct {
	Username string
	Password string
}

type Error struct {
	Message string
}

func (u *User) FetchGists() ([]*Gist, error) {
	const url = "https://api.github.com/gists"

	var gists []*Gist
	err := getJson(url, u.Username, u.Password, &gists)
	if err != nil {
		return nil, err
	}

	for _, gist := range gists {
		for _, file := range gist.Files {
			file.user = u
		}
	}

	log.Printf("Fetched %d gists", len(gists))

	return gists, nil
}

type Gist struct {
	Id        string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Files     map[string]*GistFile
}

func (g *Gist) ListFiles() map[string]*GistFile {
	return g.Files
}

type GistFile struct {
	Size   uint64
	RawUrl string `json:"raw_url"`

	user *User
}

func (file *GistFile) FetchContent() ([]byte, error) {
	req, err := http.NewRequest("GET", file.RawUrl, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(file.user.Username, file.user.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Fetched %s", file.RawUrl)

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
