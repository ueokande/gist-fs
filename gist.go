package main

import "time"

type User struct {
}

func (r *User) FetchGists() ([]*Gist, error) {
	return []*Gist{
		&Gist{id: "abcd1234"},
		&Gist{id: "wxyz5678"},
		&Gist{id: "AABBCCDD"},
	}, nil
}

type Gist struct {
	id    string
	ctime time.Time
	mtime time.Time
}

func (g *Gist) FetchFiles() ([]*GistFile, error) {
	return []*GistFile{
		&GistFile{name: "main.c"},
		&GistFile{name: "main.sh"},
	}, nil
}

type GistFile struct {
	name string
}

func (file *GistFile) FetchContent() ([]byte, error) {
	return []byte(`printf("Hello world"\n)`), nil
}
