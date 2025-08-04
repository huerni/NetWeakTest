package main

import (
	pb "CraneNetWeak/generated/protos"
	"CraneNetWeak/server"
	"CraneNetWeak/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	log.SetLevel(log.DebugLevel)

	socket, err := util.GetUnixSocket("/var/craned/crane_test.sock", 0666)
	if err != nil {
		log.Fatalf("Failed to listen on unix socket: %s", err.Error())
	}

	serverOptions := []grpc.ServerOption{
		grpc.KeepaliveParams(util.ServerKeepAliveParams),
		grpc.KeepaliveEnforcementPolicy(util.ServerKeepAlivePolicy),
	}

	s := grpc.NewServer(serverOptions...)
	pb.RegisterCraneTestServer(s, &server.CraneTestServiceServer{})

	log.Info("gRPC server listening on /var/craned/crane_test.sock")

	if err := s.Serve(socket); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
