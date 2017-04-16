package main

import (
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

var CachePeriod = time.Duration(5 * time.Minute)

type Node interface {
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
	List() ([]Node, error)
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

	client *Client

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

func (dir *RootDir) List() ([]Node, error) {
	if now := time.Now(); dir.lastFetchAt.Add(CachePeriod).Before(time.Now()) {
		var err error
		dir.cache, err = dir.client.FetchGists()
		if err != nil {
			return nil, err
		}
		dir.lastFetchAt = now
	}

	children := make([]Node, len(dir.cache))
	for i, gist := range dir.cache {
		children[i] = &GistDir{
			Node:   nodefs.NewDefaultNode(),
			client: dir.client,
			gist:   gist,
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
		Node:   nodefs.NewDefaultNode(),
		client: NewClient(username, password),
	}
}

type GistDir struct {
	nodefs.Node

	client *Client

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

func (dir *GistDir) List() ([]Node, error) {
	files := dir.gist.Files
	children := make([]Node, len(files)+1) // +1 : meta directory
	var index int
	for name, file := range files {
		children[index] = &FileNode{
			Node: nodefs.NewDefaultNode(),
			name: name,
			file: &File{
				File:     nodefs.NewDefaultFile(),
				gist:     dir.gist,
				gistFile: file,
				client:   dir.client,
				name:     name,
			},
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

func (f *GistMetaDir) List() ([]Node, error) {
	return []Node{
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

type FileNode struct {
	nodefs.Node

	name string

	file *File
}

func (dir *FileNode) Name() string {
	return dir.name
}

func (dir *FileNode) IsDir() bool {
	return false
}

func (f *FileNode) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Size = f.file.gistFile.Size
	out.Mode = fuse.S_IFREG | 0644

	// TODO ctime/mtime from revision?
	out.Ctime = uint64(f.file.fetchedAt.Unix())
	out.Mtime = uint64(f.file.localMtime.Unix())

	return fuse.OK
}

func (f *FileNode) Open(flags uint32, ctx *fuse.Context) (nodefs.File, fuse.Status) {
	err := f.file.FetchGistFile()
	if err != nil {
		return nil, fuse.ToStatus(err)
	}
	return f.file, fuse.OK
}

func (f *FileNode) Truncate(file nodefs.File, size uint64, context *fuse.Context) (code fuse.Status) {
	return f.file.Truncate(size)
}

type File struct {
	nodefs.File
	name string

	client   *Client
	gistFile *GistFile
	gist     *Gist

	content    []byte
	localMtime time.Time
	fetchedAt  time.Time
}

func (f *File) FetchGistFile() error {
	if f.content == nil {
		var err error
		f.content, err = f.client.FetchContent(f.gistFile.RawUrl)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *File) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	return fuse.ReadResultData(f.content), fuse.OK
}

func (f *File) Write(data []byte, off int64) (written uint32, code fuse.Status) {
	if int(off)+len(data) > len(f.content) {
		index := len(f.content) - int(off)
		copy(f.content[off:], data[:index])
		f.content = append(f.content, data[index:]...)
	} else {
		copy(f.content[off:], data)
	}

	f.localMtime = time.Now()

	return uint32(len(data)), fuse.OK
}

func (f *File) Truncate(size uint64) fuse.Status {
	f.content = nil
	return fuse.OK
}

func (f *File) Flush() fuse.Status {
	if !f.localMtime.After(f.fetchedAt) {
		return fuse.OK
	}

	err := f.client.UpdateContent(f.gist.Id, f.name, string(f.content))
	if err != nil {
		return fuse.ToStatus(err)
	}
	f.fetchedAt = f.localMtime
	return fuse.OK
}
