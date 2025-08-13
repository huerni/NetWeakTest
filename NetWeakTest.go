package main

import (
	"CraneNetWeak/util"
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func main() {

	nodeListStr := flag.String("N", "", "input node list")
	options := flag.String("options", "", "input options")
	sleepTime := flag.Duration("T", 30, "input time")
	logLevel := flag.String("L", "info", "log level")
	mininetOption := flag.String("type", "", "mininet option")

	flag.Parse()

	util.InitLogger(*logLevel)

	if *nodeListStr == "" {
		log.Error("input node list is empty")
		flag.Usage()
		os.Exit(1)
	}

	nodeList, err := util.ParseHostList(*nodeListStr)
	if !err {
		log.Error("parse node list error: %v", err)
		flag.Usage()
		os.Exit(1)
	}

	if *mininetOption == "reset" {
		for _, nodeName := range nodeList {
			nodePid, err := util.GetPidWithNodeName2(nodeName)
			if err != nil {
				log.Errorf("[%s] %s", nodeName, err.Error())
				return
			}
			err, output := util.ExecDelCmd(nodePid, nodeName)
			if err != nil {
				log.Errorf("[%s] %s, cmd output: ", nodeName, err.Error(), output)
				return
			}
			log.Infof("[%s] Network weak stopped", nodeName)
		}
		return
	} else if *mininetOption == "show" {
		for _, nodeName := range nodeList {
			nodePid, err := util.GetPidWithNodeName2(nodeName)
			if err != nil {
				log.Errorf("[%s] %s", nodeName, err.Error())
				continue
			}

			err, output := util.ExecShowCmd(nodePid, nodeName)
			if err != nil {
				log.Errorf("[%s] exec bash cmd error %s: %s", nodeName, err.Error(), output)
				continue
			}
			log.Infof("[%s] %s", nodeName, output)
		}
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"bash"}
	}

	done := make(chan struct{})

	go util.RunCmd(args, done)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	defer func() {
		cancel()
		wg.Wait()
		log.Info("exiting.....")
		// TODO: 检测miniet，存在时stop mininet
	}()

	for _, nodeName := range nodeList {
		wg.Add(1)
		go func(ctx context.Context, nodeName string, options string, sleepTime time.Duration) {
			defer wg.Done()
			nodePid, err := util.GetPidWithNodeName(ctx, nodeName)
			if err != nil {
				if ctx.Err() == context.Canceled || strings.Contains(err.Error(), "signal: interrupt") {
					return
				}
				log.Errorf("[%s] %s", nodeName, err.Error())
				return
			}

			log.Infof("[%s] Starting Network weak", nodeName)
			for {
				select {
				case <-ctx.Done():
					log.Infof("[%s] Network weak stopping...", nodeName)
					return
				case <-time.After(sleepTime * time.Second):
					var actualOption string
					if options != "" {
						actualOption = options
					} else {
						actualOption = util.GetRandomOption()
					}

					err, output := util.ExecReplaceCmd(nodePid, nodeName, actualOption)
					if err != nil {
						if ctx.Err() == context.Canceled || strings.Contains(err.Error(), "signal: interrupt") {
							return
						}
						log.Errorf("[%s] %s, output: %s", nodeName, err.Error(), output)
					}

					if len(output) > 0 {
						log.Infof("[%s] %s", nodeName, output)
					}

					if log.GetLevel() == log.DebugLevel {
						err, output = util.ExecShowCmd(nodePid, nodeName)
						if err != nil {
							log.Infof("[%s] Network weak stopping...", nodeName)
							if ctx.Err() == context.Canceled || strings.Contains(err.Error(), "signal: interrupt") {
								return
							}
							log.Errorf("[%s] %s output: %s", nodeName, err.Error(), output)
						}
						log.Debugf("[%s] %s", output)
					}
				}
			}

		}(ctx, nodeName, *options, *sleepTime)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-done:

	case sig := <-sigs:
		log.Info("receive signal:", sig)
	}
}
