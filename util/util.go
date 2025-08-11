package util

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/creack/pty"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

func GetPidWithNodeName(ctx context.Context, nodeName string) (string, error) {

	interval := 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	log.Infof("waiting for %s to start...", nodeName)
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return "", fmt.Errorf("[%s] timeout: node pids not found\n", nodeName)
			}
			return "", fmt.Errorf("[%s] context cancelled\n")
		case <-time.After(interval):
			cmdStr := fmt.Sprintf("pgrep -f [m]ininet:%s | head -n 1", nodeName)
			out, err := exec.Command("bash", "-c", cmdStr).Output()
			if err != nil {
				if ctx.Err() != context.Canceled && !strings.Contains(err.Error(), "signal: interrupt") {
					log.Errorf("[%s] pgrep error: %v, output: %s", nodeName, err, string(out))
				}
				continue
			}
			pid := strings.TrimSpace(string(out))
			if pid != "" {
				log.Debugf("node: %s, pid: %s", nodeName, pid)
				cmdStr = fmt.Sprintf("mnexec -a %s echo ok", pid)
				out, err := exec.Command("bash", "-c", cmdStr).Output()
				if err != nil {
					continue
				}
				if strings.TrimSpace(string(out)) == "ok" {
					log.Debugf("%s is started", nodeName)
					return pid, nil
				}
			}
			log.Debugf("%s is not ok. waiting..", nodeName)
		}
	}
}

func GetPidWithNodeName2(nodeName string) (string, error) {
	cmdStr := fmt.Sprintf("pgrep -f [m]ininet:%s | head -n 1", nodeName)
	out, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return "", err
	}
	pid := strings.TrimSpace(string(out))
	if pid == "" {
		return "", fmt.Errorf("%s is not running", nodeName)
	}

	log.Debugf("node: %s, pid: %s", nodeName, pid)
	return pid, nil
}

func GetPidWithNodeNames(ctx context.Context, nodeNames []string) (map[string]string, error) {
	nodePidMap := make(map[string]string)
	remaining := make(map[string]struct{})
	for _, name := range nodeNames {
		remaining[name] = struct{}{}
	}

	interval := 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for len(remaining) > 0 {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return nodePidMap, fmt.Errorf("timeout: not all node pids found")
			}
			return nil, fmt.Errorf("context cancelled")
		case <-time.After(interval):
			log.Debug("waiting for all nodes to start...")
			for nodeName := range remaining {
				cmdStr := fmt.Sprintf("ps -ef | grep 'mininet:%s' | grep -v grep | awk '{print $2}' | head -n 1", nodeName)
				out, err := exec.Command("bash", "-c", cmdStr).Output()
				if err != nil {
					log.Errorf("pgrep error: %v, output: %s", err, string(out))
					continue
				}
				pid := strings.TrimSpace(string(out))
				if pid != "" {
					log.Debugf("node: %s, pid: %s", nodeName, pid)
					nodePidMap[nodeName] = pid
					delete(remaining, nodeName)
				}
			}
		}
	}
	log.Debug("all nodes to start...")
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
		return fmt.Errorf("exec bash cmd error %s: %s", err.Error(), string(output))
	}
	if len(output) > 0 {
		log.Debug("exec bash cmd output: ", string(output))
	}

	return nil
}

func RunPty(args []string, done chan struct{}) {
	cmd := exec.Command(args[0], args[1:]...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Errorf("Failed to start pty: %v\n", err)
		close(done)
		return
	}
	defer func() { _ = ptmx.Close() }()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Errorf("Failed to set terminal raw: %v\n", err)
		close(done)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	ch <- syscall.SIGWINCH

	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, ptmx)

	close(done)
}

func RunCmd(args []string, done chan struct{}) {
	cmd := exec.Command(args[0], args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get stdout: %v\n", err)
		close(done)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get stderr: %v\n", err)
		close(done)
		return
	}

	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start command: %v\n", err)
		close(done)
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Fprintln(os.Stdout, scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Command finished with error: %v\n", err)
	}

	close(done)
}

func ParseHostList(hostStr string) ([]string, bool) {
	nameStr := strings.ReplaceAll(hostStr, " ", "")
	nameStr += ","

	var nameMeta string
	var strList []string
	var charQueue string

	for _, c := range nameStr {
		if c == '[' {
			if charQueue == "" {
				charQueue = string(c)
			} else {
				log.Errorln("Illegal node name string format: duplicate brackets")
				return nil, false
			}
		} else if c == ']' {
			if charQueue == "" {
				log.Errorln("Illegal node name string format: isolated bracket")
				return nil, false
			} else {
				nameMeta += charQueue
				nameMeta += string(c)
				charQueue = ""
			}
		} else if c == ',' {
			if charQueue == "" {
				strList = append(strList, nameMeta)
				nameMeta = ""
			} else {
				charQueue += string(c)
			}
		} else {
			if charQueue == "" {
				nameMeta += string(c)
			} else {
				charQueue += string(c)
			}
		}
	}
	if charQueue != "" {
		log.Errorln("Illegal node name string format: isolated bracket")
		return nil, false
	}

	regex := regexp.MustCompile(`.*\[(.*)\](\..*)*$`)
	var hostList []string

	for _, str := range strList {
		strS := strings.TrimSpace(str)
		if !regex.MatchString(strS) {
			hostList = append(hostList, strS)
		} else {
			nodes, ok := ParseNodeList(strS)
			if !ok {
				return nil, false
			}
			hostList = append(hostList, nodes...)
		}
	}
	return hostList, true
}

func ParseNodeList(nodeStr string) ([]string, bool) {
	bracketsRegex := regexp.MustCompile(`.*\[(.*)\]`)
	numRegex := regexp.MustCompile(`^\d+$`)
	scopeRegex := regexp.MustCompile(`^(\d+)-(\d+)$`)

	if !bracketsRegex.MatchString(nodeStr) {
		return nil, false
	}

	unitStrList := strings.Split(nodeStr, "]")
	endStr := unitStrList[len(unitStrList)-1]
	unitStrList = unitStrList[:len(unitStrList)-1]
	resList := []string{""}

	for _, str := range unitStrList {
		nodeNum := strings.FieldsFunc(str, func(r rune) bool {
			return r == '[' || r == ','
		})
		unitList := []string{}
		headStr := nodeNum[0]

		for _, numStr := range nodeNum[1:] {
			if numRegex.MatchString(numStr) {
				unitList = append(unitList, fmt.Sprintf("%s%s", headStr, numStr))
			} else if scopeRegex.MatchString(numStr) {
				locIndex := scopeRegex.FindStringSubmatch(numStr)
				start, err1 := strconv.Atoi(locIndex[1])
				end, err2 := strconv.Atoi(locIndex[2])
				if err1 != nil || err2 != nil {
					return nil, false
				}
				width := len(locIndex[1])
				for j := start; j <= end; j++ {
					sNum := fmt.Sprintf("%0*d", width, j)
					unitList = append(unitList, fmt.Sprintf("%s%s", headStr, sNum))
				}
			} else {
				return nil, false // Format error
			}
		}

		tempList := []string{}
		for _, left := range resList {
			for _, right := range unitList {
				tempList = append(tempList, left+right)
			}
		}
		resList = tempList
	}

	if endStr != "" {
		for i := range resList {
			resList[i] += endStr
		}
	}

	return resList, true
}

func InitLogger(level string) {
	switch level {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	default:
		log.Warnf("Invalid log level %s, using info level", level)
		log.SetLevel(log.InfoLevel)
	}
	log.SetReportCaller(false)
	log.SetFormatter(&nested.Formatter{})
}
