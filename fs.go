package main

import (
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

var CachePeriod = time.Duration(5 * time.Minute)

type FileNode interface {
	nodefs.Node

	Name() string
	IsDir() bool
}

type StringFile struct {
	nodefs.Node

	name  string
	value string
}

func (f *StringFile) Name() string {
	return f.name
}

func (f *StringFile) IsDir() bool {
	return false
}

func (f *StringFile) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Size = uint64(len([]byte(f.value)) + 1)
	out.Mode = fuse.S_IFREG | 0644
	return fuse.OK
}

func (f *StringFile) Open(flags uint32, ctx *fuse.Context) (nodefs.File, fuse.Status) {
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, fuse.EPERM
	}
	return nodefs.NewDataFile([]byte(f.value + "\n")), fuse.OK
}

type BoolFile struct {
	nodefs.Node

	name  string
	value bool
}

func (f *BoolFile) Name() string {
	return f.name
}

func (f *BoolFile) IsDir() bool {
	return false
}

func (f *BoolFile) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Size = 2
	out.Mode = fuse.S_IFREG | 0644
	return fuse.OK
}

func (f *BoolFile) Open(flags uint32, ctx *fuse.Context) (nodefs.File, fuse.Status) {
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, fuse.EPERM
	}
	if f.value {
		return nodefs.NewDataFile([]byte("1\n")), fuse.OK
	}
	return nodefs.NewDataFile([]byte("0\n")), fuse.OK
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
			mode = fuse.S_IFDIR | 0755
		} else {
			mode = fuse.S_IFREG | 0644
		}
		entries[i] = fuse.DirEntry{
			Name: f.Name(),
			Mode: mode,
		}

		p.NewChild(f.Name(), f.IsDir(), f)
	}
	return entries, fuse.OK
}

func lookup(dir DirNode, out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	_, status := openDir(dir)
	if status != fuse.OK {
		return nil, status
	}
	c := dir.Inode().GetChild(name)
	if c == nil {
		return nil, fuse.ENOENT
	}
	status = c.Node().GetAttr(out, nil, context)
	if status != fuse.OK {
		return nil, status
	}
	return c, fuse.OK
}

type RootDir struct {
	nodefs.Node

	server *fuse.Server

	user *User

	lastFetchAt time.Time
	cache       []*Gist
}

func (dir *RootDir) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	return openDir(dir)
}

func (dir *RootDir) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return lookup(dir, out, name, context)
}

func (dir *RootDir) Name() string {
	panic("root directory has no names")
}

func (dir *RootDir) List() ([]FileNode, error) {
	if now := time.Now(); dir.lastFetchAt.Add(CachePeriod).Before(time.Now()) {
		var err error
		dir.cache, err = dir.user.FetchGists()
		if err != nil {
			return nil, err
		}
		dir.lastFetchAt = now
	}

	children := make([]FileNode, len(dir.cache))
	for i, gist := range dir.cache {
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

func (f *GistDir) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Mode = fuse.S_IFDIR | 0755
	out.Ctime = uint64(f.gist.CreatedAt.Unix())
	out.Mtime = uint64(f.gist.UpdatedAt.Unix())

	return fuse.OK
}

func (dir *GistDir) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	return openDir(dir)
}

func (dir *GistDir) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return lookup(dir, out, name, context)
}

func (dir *GistDir) List() ([]FileNode, error) {
	files := dir.gist.Files
	children := make([]FileNode, len(files)+1) // +1 : meta directory
	var index int
	for name, file := range files {
		children[index] = &File{
			Node: nodefs.NewDefaultNode(),
			name: name,
			file: file,
		}
		index++
	}
	children[len(children)-1] = &GistMetaDir{
		Node: nodefs.NewDefaultNode(),
		gist: dir.gist,
	}
	return children, nil
}

type GistMetaDir struct {
	nodefs.Node

	gist *Gist
}

func (f *GistMetaDir) Name() string {
	return ".gist"
}

func (f *GistMetaDir) IsDir() bool {
	return true
}

func (f *GistMetaDir) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Mode = fuse.S_IFDIR | 0755
	out.Ctime = uint64(f.gist.CreatedAt.Unix())
	out.Mtime = uint64(f.gist.UpdatedAt.Unix())

	return fuse.OK
}
func (f *GistMetaDir) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	return openDir(f)
}

func (dir *GistMetaDir) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return lookup(dir, out, name, context)
}

func (f *GistMetaDir) List() ([]FileNode, error) {
	return []FileNode{
		&StringFile{
			Node:  nodefs.NewDefaultNode(),
			name:  "description",
			value: f.gist.Description,
		},
		&StringFile{
			Node:  nodefs.NewDefaultNode(),
			name:  "id",
			value: f.gist.Id,
		},
		&BoolFile{
			Node:  nodefs.NewDefaultNode(),
			name:  "public",
			value: f.gist.Public,
		},
	}, nil
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
	out.Mode = fuse.S_IFREG | 0644

	// TODO ctime/mtime from revision?
	out.Ctime = uint64(f.file.Gist.CreatedAt.Unix())
	out.Mtime = uint64(f.file.Gist.UpdatedAt.Unix())

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
