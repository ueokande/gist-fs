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

func (g *Gist) FetchFiles() ([]*File, error) {
	return []*File{
		&File{name: "main.c", content: []byte(`printf("Hello world\n");`)},
		&File{name: "main.sh", content: []byte(`echo "Hello world\n"`)},
	}, nil
}

type File struct {
	name    string
	content []byte
}
