package util

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/keepalive"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"time"
)

var ServerKeepAliveParams = keepalive.ServerParameters{
	Time:    10 * time.Minute, // GRPC_ARG_KEEPALIVE_TIME_MS
	Timeout: 20 * time.Second, // GRPC_ARG_KEEPALIVE_TIMEOUT_MS
}

var ServerKeepAlivePolicy = keepalive.EnforcementPolicy{
	MinTime:             10 * time.Second, // GRPC_ARG_HTTP2_MIN_RECV_PING_INTERVAL_WITHOUT_DATA_MS
	PermitWithoutStream: true,             // GRPC_ARG_KEEPALIVE_PERMIT_WITHOUT_CALLS
	//No MaxPingStrikes available default is 2
}

func GetUnixSocket(path string, mode fs.FileMode) (net.Listener, error) {
	dir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); err == nil {
		err := os.Remove(path)
		if err != nil {
			log.Errorf("Failed to remove file %s: %v", path, err)
			return nil, fmt.Errorf("error when removing existing unix socket")
		}
	}

	socket, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	// 0600 -> only owner can access, no need to use TLS
	// 0666 -> everyone can access, insecure
	if err = os.Chmod(path, mode); err != nil {
		return nil, err
	}

	return socket, nil
}
