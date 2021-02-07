package iaria

import (
	"net"
	"os/exec"
	"strconv"
)

// Secret which is passed to aria
var Secret = "nilsocket/labdh"

// StartAria downloader to listen
func StartAria(ariaArgs []string, concDls int) (*exec.Cmd, int, error) {

	portNum := closedPortNum()

	rpcPort := "--rpc-listen-port=" + strconv.Itoa(portNum)

	maxConcDls := "--max-concurrent-downloads=" + strconv.Itoa(concDls)

	ariaArgs = append(ariaArgs, rpcPort, maxConcDls)

	cmd, err := launch(ariaArgs)

	return cmd, portNum, err
}

func launch(ariaArgs []string) (*exec.Cmd, error) {
	args := append(ariaArgs, "--enable-rpc", "--rpc-listen-all", "--rpc-secret="+Secret)
	cmd := exec.Command("aria2c", args...)

	cmd.Start()
	return cmd, nil
}

func closedPort(port int) bool {
	conn, err := net.Dial("tcp", net.JoinHostPort("localhost", strconv.Itoa(port)))
	if err == nil && conn != nil {
		conn.Close()
		return false
	}
	return true
}

func closedPortNum() int {
	for i := 1024; i < 65535; i++ {
		if closedPort(i) {
			return i
		}
	}
	return 0
}
