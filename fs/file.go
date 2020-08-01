package fs

import (
	"context"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
)

// Callers should have lock held
func (n *Node) resizeUnlocked(sz uint64) {
	if sz > uint64(cap(n.content)) {
		buf := make([]byte, sz)
		copy(buf, n.content)
		n.content = buf
	} else {
		n.content = n.content[:sz]
	}
}

// Open gets value from etcd, and saves it in "content" for later read
func (n *Node) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	if n.content == nil {
		if rc, err := n.client.GetValue(ctx, n.path); err != nil {
			logrus.WithError(err).WithField("path", n.path).Errorf("Failed to get value from etcd")
			return nil, 0, syscall.EIO
		} else {
			n.rwMu.Lock()
			n.content = rc
			n.rwMu.Unlock()
		}
	}
	logrus.WithField("path", n.path).WithField("length", len(n.content)).Debug("Node Open")
	return n, fuse.FOPEN_DIRECT_IO, fs.OK
}

// Read returns bytes from "content", which should be filled by a prior Open operation
func (n *Node) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	n.rwMu.RLock()
	defer n.rwMu.RUnlock()
	logrus.WithField("path", n.path).Debug("Node Read")

	end := int(off) + len(dest)
	if end > len(n.content) {
		end = len(n.content)
	}
	// We could copy to the `dest` buffer, but since we have a
	// []byte already, return that.
	return fuse.ReadResultData(n.content[off:end]), fs.OK
}

// Write saves to the internal "content" buffer
func (n *Node) Write(ctx context.Context, fh fs.FileHandle, buf []byte, off int64) (uint32, syscall.Errno) {
	n.rwMu.Lock()
	defer n.rwMu.Unlock()
	logrus.WithField("path", n.path).WithField("length", len(buf)).Debug("Node Write")
	sz := int64(len(buf))
	if off+sz > int64(len(n.content)) {
		n.resizeUnlocked(uint64(off + sz))
	}
	copy(n.content[off:], buf)
	return uint32(sz), 0
}

// Create actually writes an empty value into etcd (as a placeholder)
func (n *Node) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	fullPath := filepath.Join(n.path, string(filepath.Separator), name)
	logrus.WithField("path", fullPath).Debug("Node Create")
	child := Node{
		path:   fullPath,
		client: n.client,
		isLeaf: true,
	}
	_, err := child.Write(ctx, nil, []byte{}, 0)
	return n.NewInode(ctx, &child, fs.StableAttr{Mode: child.getMode(child.isLeaf), Ino: n.inodeHash(child.path)}), nil, 0, err
}

// Flush puts file content into etcd
func (n *Node) Flush(ctx context.Context, fh fs.FileHandle) syscall.Errno {
	logrus.WithField("path", n.path).Debug("Node Flush")
	n.rwMu.RLock()
	defer n.rwMu.RUnlock()
	if err := n.client.PutValue(ctx, n.path, n.content); err != nil {
		logrus.WithError(err).WithField("path", n.path).Errorf("Failed to put value into etcd")
		return syscall.EIO
	}
	return fs.OK
}

// Some editors (eg. Vim) need to call Fsync, so implement it here as a no-op
func (n *Node) Fsync(ctx context.Context, f fs.FileHandle, flags uint32) syscall.Errno {
	logrus.WithField("path", n.path).Debug("Node Fsync")
	return fs.OK
}

// Unlink removes a key from etcd
func (n *Node) Unlink(ctx context.Context, name string) syscall.Errno {
	fullPath := filepath.Join(n.path, string(filepath.Separator), name)
	logrus.WithField("path", fullPath).Debug("Node Unlink")
	if err := n.client.DeleteKey(ctx, fullPath); err != nil {
		logrus.WithError(err).WithField("path", fullPath).Errorf("Failed to delete key from etcd")
		return syscall.EIO
	}
	return fs.OK
}

// Implement Setattr to support truncation
func (n *Node) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if sz, ok := in.GetSize(); ok {
		n.rwMu.Lock()
		n.resizeUnlocked(sz)
		n.rwMu.Unlock()
	}
	if errno := n.Flush(ctx, nil); errno != fs.OK {
		return errno
	}
	return n.Getattr(ctx, fh, out)
}

var (
	_ fs.NodeUnlinker  = &Node{}
	_ fs.NodeCreater   = &Node{}
	_ fs.NodeOpener    = &Node{}
	_ fs.FileReader    = &Node{}
	_ fs.NodeWriter    = &Node{}
	_ fs.NodeFlusher   = &Node{}
	_ fs.NodeFsyncer   = &Node{}
	_ fs.NodeSetattrer = &Node{}
)
