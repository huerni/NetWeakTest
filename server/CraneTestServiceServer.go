package server

import (
	pb "CraneNetWeak/generated/protos"
	"CraneNetWeak/util"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type CraneTestServiceServer struct {
	pb.UnimplementedCraneTestServer

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func (c *CraneTestServiceServer) NetWeakStart(ctx context.Context, request *pb.NetWeakStartRequest) (*pb.NetWeakStartReply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &pb.NetWeakStartReply{Ok: true, Msg: "Network weak is started."}, nil
	}

	ctx2, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.started = true

	nodePidMap, err := util.GetPidWithNodeNames(request.NodeList)
	if err != nil {
		log.Error(err.Error())
		return &pb.NetWeakStartReply{Ok: false, Msg: err.Error()}, nil
	}
	for nodeName, nodePid := range nodePidMap {
		go func(ctx context.Context, nodeName string, nodePid string, option string) {
			log.Debugf("[%s] Starting Network weak", nodeName)
			for {
				select {
				case <-ctx.Done():
					log.Debugf("[%s] Network weak stopped", nodeName)
					cmdStr := fmt.Sprintf("sudo mnexec -a %s tc qdisc del dev %s-eth0 root", nodePid, nodeName)
					err = util.ExecBashCmd(cmdStr)
					if err != nil {
						log.Errorf("[%s] %s", nodeName, err.Error())
						return
					}
					return
				default:
					var actualOption string
					if option != "" {
						actualOption = option
					} else {
						actualOption = util.GetRandomOption()
					}
					cmdStr := fmt.Sprintf("sudo mnexec -a %s tc qdisc replace dev %s-eth0 root netem %s", nodePid, nodeName, actualOption)
					err := util.ExecBashCmd(cmdStr)
					if err != nil {
						log.Errorf("[%s] %s", nodeName, err.Error())
						return
					}
					cmdStr = fmt.Sprintf("sudo mnexec -a %s tc qdisc show dev %s-eth0", nodePid, nodeName)
					err = util.ExecBashCmd(cmdStr)
					if err != nil {
						log.Errorf("[%s] %s", nodeName, err.Error())
						return
					}

					time.Sleep(3 * time.Second)

					cmdStr = fmt.Sprintf("sudo mnexec -a %s tc qdisc del dev %s-eth0 root", nodePid, nodeName)
					err = util.ExecBashCmd(cmdStr)
					if err != nil {
						log.Errorf("[%s] %s", nodeName, err.Error())
						return
					}
				}
			}
		}(ctx2, nodeName, nodePid, request.Option)
	}

	return &pb.NetWeakStartReply{Ok: true, Msg: "Network weak is started."}, nil
}

func (c *CraneTestServiceServer) NetWeakStop(ctx context.Context, request *pb.Empty) (*pb.NetWeakStopReply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return &pb.NetWeakStopReply{Ok: true, Msg: "Network weak is stop."}, nil
	}

	if c.cancel == nil {
		return nil, fmt.Errorf("cancel is nil")
	}

	c.cancel()
	c.started = false

	return &pb.NetWeakStopReply{Ok: true, Msg: "Network weak is stop."}, nil
}
