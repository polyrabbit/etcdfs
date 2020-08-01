package fs

import (
	"context"
	"hash/fnv"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/polyrabbit/etcdfs/etcd"
	"github.com/sirupsen/logrus"
	v3 "go.etcd.io/etcd/v3/clientv3"
)

// Set file owners to the current user,
// otherwise in OSX, we will fail to start.
var uid, gid uint32

func init() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	uid32, _ := strconv.ParseUint(u.Uid, 10, 32)
	gid32, _ := strconv.ParseUint(u.Gid, 10, 32)
	uid = uint32(uid32)
	gid = uint32(gid32)
}

// A tree node in filesystem, it acts as both a directory and file
type Node struct {
	fs.Inode
	client *etcd.Client
	isLeaf bool   // A leaf of the filesystem tree means it's a file
	path   string // File path to get to the current file

	rwMu    sync.RWMutex // Protect file content
	content []byte       // Internal buffer to hold the current file content
}

// A root is just a file node, with inode sets to 1 and leaf sets to false
func NewRoot(client *etcd.Client) *Node {
	return &Node{
		client: client,
		isLeaf: false,
	}
}

// List keys under a certain prefix from etcd, and output the next hierarchy level
func (n *Node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	parent := n.absPath("")
	logrus.WithField("path", parent).Debug("Node Readdir")

	entrySet := make(map[string]fuse.DirEntry)
	// We will have a fixed end.
	// A little tricky here that we use WithRange to override 'End' set by WithPrefix
	opts := []v3.OpOption{v3.WithRange(v3.GetPrefixRangeEnd(parent)),
		v3.WithSort(v3.SortByKey, v3.SortAscend), v3.WithLimit(500)}
	nextGroup := parent
	for {
		keys, moreKeys, err := n.client.ListKeys(ctx, nextGroup, opts...)
		if err != nil {
			logrus.WithError(err).WithField("path", parent).Errorf("Failed to list keys from etcd")
			return nil, syscall.EIO
		}

		var lastName string
		for _, key := range keys {
			nextLevel, hasMore := n.nextHierarchyLevel(key, parent)
			lastName = nextLevel
			if _, exist := entrySet[nextLevel]; exist {
				continue
			}
			entrySet[nextLevel] = fuse.DirEntry{
				Mode: n.getMode(!hasMore),
				Name: nextLevel,
				Ino:  n.inodeHash(nextLevel),
			}
		}
		// For a flat kv structure, we dont need to iterate all, eg.
		// /foo/1, /foo/2, /foo/3, /foo/4, /foo/5, /bar/1, /bar/2
		// when we find "/foo/1", we can skip all "/foo/xxx" folders and jump directly to "/bar/1"
		nextGroup = v3.GetPrefixRangeEnd(n.absPath(lastName)) // TODO: new path should end with "/"?

		if !moreKeys || len(keys) == 0 {
			break
		}
		if len(entrySet) > 1000 {
			logrus.Warn("Already fetched more than 1000 entries, skipping the rest for performance reason...")
			break
		}
	}

	entries := make([]fuse.DirEntry, 0, len(entrySet))
	for _, e := range entrySet {
		entries = append(entries, e)
	}
	return fs.NewListDirStream(entries), fs.OK
}

// Returns next hierarchy level and tells if we have more hierarchies
// path "/foo", parent "/" => "foo"
func (n *Node) nextHierarchyLevel(path, parent string) (string, bool) {
	baseName := strings.TrimPrefix(path, parent)
	hierarchies := strings.SplitN(baseName, string(filepath.Separator), 2)
	return filepath.Clean(hierarchies[0]), len(hierarchies) >= 2
}

func (n *Node) absPath(fileName string) string {
	return n.path + string(filepath.Separator) + fileName
}

// Find a file under the current node(directory)
func (n *Node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := n.absPath(name)
	logrus.WithField("path", fullPath).Debug("Node Lookup")
	keys, _, err := n.client.ListKeys(ctx, fullPath, v3.WithLimit(1))
	if err != nil {
		logrus.WithError(err).WithField("path", fullPath).Errorf("Failed to list keys from etcd")
		return nil, syscall.EIO
	}
	if len(keys) == 0 {
		return nil, syscall.ENOENT
	}
	key := keys[0]
	child := Node{
		path:   fullPath,
		client: n.client,
	}
	if key == fullPath {
		child.isLeaf = true
	} else if strings.HasPrefix(key, fullPath+string(filepath.Separator)) {
		child.isLeaf = false
	} else {
		return nil, syscall.ENOENT
	}
	return n.NewInode(ctx, &child, fs.StableAttr{Mode: child.getMode(child.isLeaf), Ino: n.inodeHash(child.path)}), fs.OK
}

func (n *Node) getMode(isLeaf bool) uint32 {
	if isLeaf {
		return 0644 | uint32(syscall.S_IFREG)
	} else {
		return 0755 | uint32(syscall.S_IFDIR)
	}
}

// Getattr outputs file attributes
// TODO: how to invalidate them?
func (n *Node) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = n.getMode(n.isLeaf)
	out.Size = uint64(len(n.content))
	out.Ino = n.inodeHash(n.path)
	now := time.Now()
	out.SetTimes(&now, &now, &now)
	out.Uid = uid
	out.Gid = gid
	return fs.OK
}

// Hash file path into inode number, so we can ensure the same file always gets the same inode number
func (n *Node) inodeHash(path string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(path))
	return h.Sum64()
}

var (
	_ fs.NodeGetattrer = &Node{}
	_ fs.NodeReaddirer = &Node{}
	_ fs.NodeLookuper  = &Node{}
)
