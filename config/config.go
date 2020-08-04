package config

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	MountPoint     string
	MountOptions   []string
	Endpoints      []string
	Verbose        bool
	DialTimeout    time.Duration
	CommandTimeOut time.Duration
	EnablePprof    bool
	// Secure config
	CertFile      string
	KeyFile       string
	TrustedCAFile string

	// Will be set by go-build
	Version string
	Rev     string
)

const (
	defaultDialTimeout    = 2 * time.Second
	defaultCommandTimeOut = 3 * time.Second
	defaultPprofAddress   = "localhost:9327"
)

var (
	rootCmd = &cobra.Command{
		Use:   fmt.Sprintf("%s [mount-point]", os.Args[0]),
		Short: "Mount etcd to local file system - find help/update at https://github.com/polyrabbit/etcdfs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			MountPoint = args[0]
			return nil
		},
	}
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "15:04:05", FullTimestamp: true})

	version := Version
	if version != "" && Rev != "" {
		version = fmt.Sprintf("%s, build %s", version, Rev)
	}
	rootCmd.Version = version

	// We use the same flags as etcd
	rootCmd.Flags().StringSliceVar(&Endpoints, "endpoints", []string{"127.0.0.1:2379"}, "etcd endpoints")
	rootCmd.Flags().DurationVar(&DialTimeout, "dial-timeout", defaultDialTimeout, "dial timeout for client connections")
	rootCmd.Flags().DurationVar(&CommandTimeOut, "read-timeout", defaultCommandTimeOut, "timeout for reading and writing to etcd")
	rootCmd.Flags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().BoolVar(&EnablePprof, "enable-pprof", false, fmt.Sprintf("enable runtime profiling data via HTTP server. Address is at %q", "http://"+defaultPprofAddress+"/debug/pprof"))

	rootCmd.Flags().StringVar(&CertFile, "cert", "", "identify secure client using this TLS certificate file")
	rootCmd.Flags().StringVar(&KeyFile, "key", "", "identify secure client using this TLS key file")
	rootCmd.Flags().StringVar(&TrustedCAFile, "cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")

	rootCmd.Flags().StringSliceVar(&MountOptions, "mount-options", []string{"nonempty"}, "options are passed as -o string to fusermount")

	rootCmd.Flags().SortFlags = false
	rootCmd.SilenceErrors = true
}

func Execute() bool {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorln(err)
		return false
	}
	if len(MountPoint) == 0 {
		return false
	}

	if Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if EnablePprof {
		go func() {
			if err := http.ListenAndServe(defaultPprofAddress, nil); err != nil {
				logrus.WithError(err).Error("Failed to serve pprof")
			}
		}()
	}
	return true
}
