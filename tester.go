package main

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/eternal-flame-AD/tcping/ping"
)

var AvailableTesters = map[string]Tester{
	"tcping": TCPingTester{},
	"ssl":    SSLTester{},
}

type TestResult struct {
	success     bool
	delay       time.Duration
	successRate float64
}

type Tester interface {
	TestIP(ip string, hostname string, quiet bool) TestResult
}

type TCPingTester struct{}

func (_ TCPingTester) TestIP(ip string, _ string, quiet bool) TestResult {
	res := TestResult{}
	proto, _ := ping.NewProtocol(ping.TCP.String())
	for _, port := range []int{80, 443} {
		target := &ping.Target{
			Timeout:  time.Second * 2,
			Interval: 3,
			Host:     ip,
			Port:     port,
			Counter:  2,
			Protocol: proto,
		}
		pinger := ping.NewTCPing()
		pinger.SetTarget(target)
		pingerDone := pinger.Start(quiet)
		select {
		case <-pingerDone:
			break
		}
		if pinger.Result().SuccessCounter > 0 {
			res.success = true
			res.delay = pinger.Result().Avg()
			res.successRate = float64(pinger.Result().SuccessCounter) / float64(pinger.Result().Counter)
		}
	}
	return res
}

type SSLTester struct{}

func (_ SSLTester) TestIP(ip string, host string, quiet bool) TestResult {
	res := TCPingTester{}.TestIP(ip, host, quiet)
	if !res.success {
		return res
	}
	conn, err := tls.Dial("tcp", ip+":443", &tls.Config{
		ServerName: host,
	})
	if err != nil {
		fmt.Println(err)
		res.success = false
		return res
	}
	defer conn.Close()
	if err := conn.Handshake(); err != nil {
		res.success = false
		return res
	}
	return res
}
