package configs

import (
	"errors"
	"fmt"
	"github.com/logrusorgru/aurora"
	"net"
)

var (
	ok   = aurora.Bold(aurora.Green("[ OK ]"))
	fail = aurora.Bold(aurora.Red("[FAIL]"))
	warn = aurora.Bold(aurora.Yellow("[WARN]"))
)

func PrintResult(msg string, err error) {
	if err == nil {
		fmt.Printf("%-60s%s\n", msg, ok)
		return
	} else {
		fmt.Printf("%-60s%s : %s\n", msg, fail, err)
	}
}

func PrintWarning(msg string, warning string) {
	fmt.Printf("%-60s%s : %s\n", msg, warn, warning)
}

func CheckAddress(protocol string, ip string, port int) {
	msg := fmt.Sprintf("checking protocol : %s", protocol)
	var err error = nil
	if protocol != "tcp" && protocol != "udp" {
		err = errors.New("invalid protocol")
	}
	PrintResult(msg, err)

	msg = fmt.Sprintf("checking ip address : %s", ip)
	err = nil
	if ip := net.ParseIP(ip); ip == nil {
		err = errors.New("invalid ip address")
	}
	PrintResult(msg, err)

	msg = fmt.Sprintf("checking port number : %d", port)
	err = nil
	if port > 65535 || port < 1 {
		err = errors.New("invalid port number")
	}
	PrintResult(msg, err)
}
