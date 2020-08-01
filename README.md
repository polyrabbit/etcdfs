# etcdfs - A FUSE filesystem backed by etcd

[![ci](https://github.com/polyrabbit/etcdfs/workflows/ci/badge.svg)](https://github.com/polyrabbit/etcdfs/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/polyrabbit/my-token/pulls)
[![Go Report Card](https://goreportcard.com/badge/github.com/polyrabbit/etcdfs)](https://goreportcard.com/report/github.com/polyrabbit/etcdfs)

Tired of typing `etcdctl`? why not mount it to local filesystem and open in your favorite editors?

## Example

Wondering how Kubernetes organizes data in etcd? After mounting it locally, we can use [VS Code](https://code.visualstudio.com/) to get the whole picture:

![etcd-of-a-kubernetes](https://user-images.githubusercontent.com/2657334/88920326-36d84e80-d29f-11ea-93c9-8e5d1fbd4a56.png)

_Hint: steps to mount Kubernetes etcd locally:_

```bash
$ # scp etcd certificates to a local directory (keep them carefully)
$ scp -r <kubernetes-master-ip>:/etc/kubernetes/pki/etcd .
$ # mount to a local directory
$ etcdfs --endpoints=<kubernetes-master-ip>:2379 --cacert etcd/ca.crt --key etcd/server.key --cert etcd/server.crt mnt
$ # open it in VS code
$ code mnt
```

## Installation

#### Homebrew

```bash
# WIP
```

#### `curl | bash` style downloads to `/usr/local/bin`
```bash
$ curl -sfL https://raw.githubusercontent.com/polyrabbit/etcdfs/master/.godownloader.sh | bash -s -- -d -b /usr/local/bin
```

#### Using [Go](https://golang.org/)
```bash
$ go get -u github.com/polyrabbit/etcdfs
```

## Usage

```bash
$ etcdfs
Mount etcd to local file system - find help/update from https://github.com/polyrabbit/etcdfs

Usage:
  etcdfs [mount-point] [flags]

Flags:
      --endpoints strings       etcd endpoints (default [127.0.0.1:2379])
      --dial-timeout duration   dial timeout for client connections (default 2s)
      --read-timeout duration   timeout for reading and writing to etcd (default 3s)
  -v, --verbose                 verbose output
      --enable-pprof            enable runtime profiling data via HTTP server. Address is at "http://localhost:9327/debug/pprof"
      --cert string             identify secure client using this TLS certificate file
      --key string              identify secure client using this TLS key file
      --cacert string           verify certificates of TLS-enabled secure servers using this CA bundle
      --mount-options strings   options are passed as -o string to fusermount (default [nonempty])
  -h, --help                    help for etcdfs
```

_Notice: `etcdfs` has a very similar CLI syntax to `etcdctl`._

## Limitations

* Etcdfs depends on the [FUSE](https://en.wikipedia.org/wiki/Filesystem_in_Userspace) kernel module which only supports Linux and macOS(?). It needs to be installed first:
    * Linux: `yum/apt-get install -y fuse`
    * macOS: install [OSXFUSE](https://osxfuse.github.io/)
* Keys in etcd should have a hierarchical structure to fit the filesystem tree model. And currently the only supported hierarchy separator is `/` (the same as *nix), more will be supported in the future. 
* Currently only etcd v3 is supported.

## TODO

- [x] ~~When building a directory, all keys belonging to that directory can be skipped~~
- [ ] Support hierarchy separators other than `/` in etcd
- [ ] Watch for file/directory changes

### Credits

 * Inspired by [rssfs](https://github.com/dertuxmalwieder/rssfs)
 * Based on [go-fuse](https://github.com/hanwen/go-fuse) binding

## License

The MIT License (MIT) - see [LICENSE.md](https://github.com/polyrabbit/etcdfs/blob/master/LICENSE) for more details
