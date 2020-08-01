package etcd

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"github.com/polyrabbit/etcdfs/config"
	"github.com/sirupsen/logrus"
	v3 "go.etcd.io/etcd/v3/clientv3"
	"go.etcd.io/etcd/v3/pkg/transport"
	"go.uber.org/zap"
)

type Client struct {
	client *v3.Client
}

func MustNew() *Client {
	// Disable noisy zap log
	disabledZapConfig := zap.NewDevelopmentConfig()
	disabledZapConfig.OutputPaths = []string{"/dev/null"}
	disabledZapConfig.ErrorOutputPaths = []string{"/dev/null"}

	tlsinfo := transport.TLSInfo{}
	tlsinfo.CertFile = config.CertFile
	tlsinfo.KeyFile = config.KeyFile
	tlsinfo.TrustedCAFile = config.TrustedCAFile
	var tlsConf *tls.Config
	if !tlsinfo.Empty() {
		tlsConf, _ = tlsinfo.ClientConfig()
	}

	etcdClient, err := v3.New(v3.Config{
		Endpoints:        config.Endpoints,
		DialTimeout:      config.DialTimeout,
		AutoSyncInterval: time.Minute,
		LogConfig:        &disabledZapConfig,
		TLS:              tlsConf,
	})
	if err != nil {
		logrus.WithError(err).Fatal("Failed to new etcd client")
	}
	c := &Client{etcdClient}
	// Get a random key. As long as we can get the response without an error, the server is health.
	// Otherwise we fail fast to avoid propagating errors to filesystem layer.
	_, err = c.GetValue(context.TODO(), "_ping")
	if err != nil {
		logrus.WithError(err).WithField("endpoints", strings.Join(config.Endpoints, ",")).Fatal("etcd server does not respond")
	}
	return c
}

// List keys beginning at certain prefix, with ability to specify range-end
func (c *Client) ListKeys(ctx context.Context, prefix string, opts ...v3.OpOption) ([]string, bool, error) {
	defaultOpts := []v3.OpOption{v3.WithPrefix(), v3.WithKeysOnly(), v3.WithSerializable()}
	defaultOpts = append(defaultOpts, opts...)
	ctx, _ = context.WithTimeout(ctx, config.CommandTimeOut)
	resp, err := c.client.Get(ctx, prefix, defaultOpts...)
	if err != nil {
		return nil, false, err
	}
	keys := make([]string, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		keys[i] = string(kv.Key)
	}
	return keys, resp.More, nil
}

func (c *Client) GetValue(ctx context.Context, key string, opts ...v3.OpOption) ([]byte, error) {
	defaultOpts := []v3.OpOption{v3.WithSerializable()}
	defaultOpts = append(defaultOpts, opts...)
	ctx, _ = context.WithTimeout(ctx, config.CommandTimeOut)
	resp, err := c.client.Get(ctx, key, defaultOpts...)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	return resp.Kvs[0].Value, nil
}

func (c *Client) PutValue(ctx context.Context, key string, value []byte, opts ...v3.OpOption) error {
	ctx, _ = context.WithTimeout(ctx, config.CommandTimeOut)
	_, err := c.client.Put(ctx, key, string(value), opts...)
	return err
}

func (c *Client) DeleteKey(ctx context.Context, key string, opts ...v3.OpOption) error {
	ctx, _ = context.WithTimeout(ctx, config.CommandTimeOut)
	_, err := c.client.Delete(ctx, key, opts...)
	return err
}

func (c *Client) Close() error {
	return c.client.Close()
}
