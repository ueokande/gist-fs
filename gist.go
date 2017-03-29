package main

import (
	"encoding/json"
	"errors"
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

func getJson(url, username, password string, v interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errmsg Error
		d.Decode(&errmsg)
		if err != nil {
			return err
		}
		return errors.New(errmsg.Message)
	}

	return d.Decode(v)
}

func (u *User) FetchGists() ([]*Gist, error) {
	const url = "https://api.github.com/gists"

	var gists []*Gist
	err := getJson(url, u.Username, u.Password, &gists)
	if err != nil {
		return nil, err
	}

	return gists, nil
}

type Gist struct {
	Id        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Files     map[string]*GistFile
}

func (g *Gist) ListFiles() map[string]*GistFile {
	return g.Files
}

type GistFile struct {
	Size uint64
}

func (file *GistFile) FetchContent() ([]byte, error) {
	return []byte(`printf("Hello world"\n)`), nil
}
