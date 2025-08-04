package main

import (
	"CraneNetWeak/util"
	"context"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	log.SetLevel(log.DebugLevel)

	nodeListStr := flag.String("nodeList", "", "input node list")
	options := flag.String("options", "", "input options")
	sleepTime := flag.Duration("time", 30, "input time")

	flag.Parse()

	if *nodeListStr == "" {
		log.Error("input node list is empty")
		flag.Usage()
		os.Exit(1)
	}

	nodePidMap, err := util.GetPidWithNodeNames(strings.Split(*nodeListStr, ","))
	if err != nil {
		log.Fatalf(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	for nodeName, nodePid := range nodePidMap {
		wg.Add(1)
		go func(ctx context.Context, nodeName string, nodePid string, options string, sleepTime time.Duration) {
			defer wg.Done()
			log.Debugf("[%s] Starting Network weak", nodeName)
			for {
				select {
				case <-ctx.Done():
					log.Debugf("[%s] Network weak stopping", nodeName)
					cmdStr := fmt.Sprintf("sudo mnexec -a %s tc qdisc del dev %s-eth0 root", nodePid, nodeName)
					err = util.ExecBashCmd(cmdStr)
					if err != nil {
						log.Errorf("[%s] %s", nodeName, err.Error())
						return
					}
					return
				default:
					var actualOption string
					if options != "" {
						actualOption = options
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

					select {
					case <-ctx.Done():
						log.Debugf("[%s] Network weak stopping", nodeName)
						cmdStr := fmt.Sprintf("sudo mnexec -a %s tc qdisc del dev %s-eth0 root", nodePid, nodeName)
						err = util.ExecBashCmd(cmdStr)
						if err != nil {
							log.Errorf("[%s] %s", nodeName, err.Error())
							return
						}
						return
					case <-time.After(sleepTime * time.Second):
					}
				}
			}
		}(ctx, nodeName, nodePid, *options, *sleepTime)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Info("收到信号:", sig)

	cancel()
	wg.Wait()
}
