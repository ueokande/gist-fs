package main

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type FileNode interface {
	nodefs.Node

	Name() string
	IsDir() bool
}

type DirNode interface {
	nodefs.Node

	Name() string
	List() ([]FileNode, error)
}

func openDir(dir DirNode) ([]fuse.DirEntry, fuse.Status) {
	files, err := dir.List()
	if err != nil {
		return nil, fuse.ToStatus(err)
	}

	p := dir.Inode()
	entries := make([]fuse.DirEntry, len(files))
	for i, f := range files {
		var mode uint32
		if f.IsDir() {
			mode = fuse.S_IFDIR | 0555
		} else {
			mode = fuse.S_IFREG | 0444
		}
		entries[i] = fuse.DirEntry{
			Name: f.Name(),
			Mode: mode,
		}

		p.NewChild(f.Name(), f.IsDir(), f)
	}
	return entries, fuse.OK
}

type RootDir struct {
	nodefs.Node

	server *fuse.Server

	user *User
}

func (dir *RootDir) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	return openDir(dir)
}

func (dir *RootDir) Name() string {
	panic("root directory has no names")
}

func (dir *RootDir) List() ([]FileNode, error) {
	gists, err := dir.user.FetchGists()
	if err != nil {
		return nil, err
	}
	children := make([]FileNode, len(gists))
	for i, gist := range gists {
		children[i] = &GistDir{
			Node: nodefs.NewDefaultNode(),
			gist: gist,
		}

	}
	return children, nil
}

func (dir *RootDir) Mount(mountpoint string) error {
	var err error

	dir.server, _, err = nodefs.MountRoot(mountpoint, dir, nil)
	if err != nil {
		return err
	}
	dir.server.Serve()
	return nil
}

func (dir *RootDir) Unmount() error {
	return dir.server.Unmount()
}

func NewRoot(username, password string) *RootDir {
	return &RootDir{
		Node: nodefs.NewDefaultNode(),
		user: &User{
			Username: username,
			Password: password,
		},
	}
}

type GistDir struct {
	nodefs.Node

	gist *Gist
}

func (dir *GistDir) Name() string {
	return dir.gist.Id
}

func (dir *GistDir) IsDir() bool {
	return true
}

func (dir *GistDir) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	return openDir(dir)
}

func (dir *GistDir) List() ([]FileNode, error) {
	files := dir.gist.ListFiles()
	children := make([]FileNode, len(files))
	var index int
	for name, file := range files {
		children[index] = &File{
			Node: nodefs.NewDefaultNode(),
			name: name,
			file: file,
		}
		index++
	}
	return children, nil
}

type File struct {
	nodefs.Node

	name string
	file *GistFile
}

func (dir *File) Name() string {
	return dir.name
}

func (dir *File) IsDir() bool {
	return false
}

func (f *File) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Size = f.file.Size
	out.Mode = fuse.S_IFREG | 0444

	return fuse.OK
}

func (f *File) Open(flags uint32, ctx *fuse.Context) (nodefs.File, fuse.Status) {
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, fuse.EPERM
	}
	content, err := f.file.FetchContent()
	if err != nil {
		return nil, fuse.ToStatus(err)
	}
	return nodefs.NewDataFile(content), fuse.OK
}
