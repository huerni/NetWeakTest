package util

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

func GetPidWithNodeNames(nodeNames []string) (map[string]string, error) {
	nodePidMap := make(map[string]string)

	for _, nodeName := range nodeNames {
		cmdStr := fmt.Sprintf("ps -ef | grep 'mininet:%s' | grep -v grep | awk '{print $2}' | head -n 1", nodeName)
		out, err := exec.Command("bash", "-c", cmdStr).Output()
		if err != nil {
			log.Errorf("ps -ef error: ", string(out))
			return nil, err
		}
		pid := strings.TrimSpace(string(out))
		nodePidMap[nodeName] = pid
		log.Info("nodeName: ", nodeName, ", pid: ", pid)
	}
	return nodePidMap, nil
}

func GenNetemParam(option string) string {
	switch option {
	case "limit":
		return fmt.Sprintf("%s %d", option, rand.Intn(1000)+1) // 1~1000
	case "delay":
		ms := rand.Intn(500) + 1 // 1~500ms
		return fmt.Sprintf("%s %dms 10ms", option, ms)
	case "loss":
		percent := rand.Intn(100) + 1 // 1~100%
		return fmt.Sprintf("%s %d%%", option, percent)
	case "corrupt":
		percent := rand.Intn(10) + 1 // 1~10%
		return fmt.Sprintf("%s %d%%", option, percent)
	case "duplicate":
		percent := rand.Intn(10) + 1 // 1~10%
		return fmt.Sprintf("%s %d%%", option, percent)
	case "reorder":
		percent := rand.Intn(100) + 1 // 1~100%
		return fmt.Sprintf("%s %d%%", option, percent)
	case "rate":
		rates := []string{"1mbit", "5mbit", "10mbit", "100kbit"}
		return fmt.Sprintf("%s %s", option, rates[rand.Intn(len(rates))])
	default:
		log.Errorf("unknown option: %s", option)
	}

	return ""
}

func GetRandomOption() string {
	rand.Seed(time.Now().UnixNano())

	options := []string{"limit", "delay", "loss", "corrupt", "duplicate", "rate"}
	n := len(options)
	shuffled := make([]string, n)
	copy(shuffled, options)
	rand.Shuffle(n, func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	k := rand.Intn(n) + 1
	selected := shuffled[:k]
	var opts []string
	for _, opt := range selected {
		opts = append(opts, GenNetemParam(opt))
	}

	return strings.Join(opts, " ")
}

func ExecBashCmd(cmdStr string) error {
	log.Debug("ExecBashCmd: ", cmdStr)
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec bash cmd error: %s", string(output))
	}
	if len(output) > 0 {
		log.Debug("exec bash cmd output: ", string(output))
	}

	return nil
}
