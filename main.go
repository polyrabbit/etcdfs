package main

import (
	_ "net/http/pprof"
	"path/filepath"

	"github.com/polyrabbit/etcdfs/config"
	"github.com/polyrabbit/etcdfs/etcd"
	"github.com/polyrabbit/etcdfs/fs"
	"github.com/sirupsen/logrus"
)

func main() {
	if !config.Execute() {
		return
	}
	client := etcd.MustNew()
	defer client.Close()
	mountPoint, err := filepath.Abs(config.MountPoint)
	if err != nil {
		logrus.WithError(err).WithField("mountPoint", mountPoint).Fatal("Failed to get abs file path")
		return
	}
	server := fs.MustMount(mountPoint, client)
	go server.ListenForUnmount()
	logrus.Infof("Mounted to %q, use ctrl+c to terminate.", mountPoint)
	server.Wait()
}
