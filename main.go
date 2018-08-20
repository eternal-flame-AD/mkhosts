package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/cloverstd/tcping/ping"
	"github.com/docopt/docopt-go"
	"github.com/gammazero/workerpool"

	"github.com/ddliu/go-httpclient"
)

const (
	CloudFlareURL = "https://1.1.1.1/dns-query"
	WfDNS         = "application/dns-udpwireformat"
	WfJSON        = "application/dns-json"
	UserAgent     = "mkhosts"
)

var (
	domainNameRegex = regexp.MustCompile(`^[0-9\p{L}][0-9\p{L}-\.]{1,61}[0-9\p{L}]\.[0-9\p{L}][\p{L}-]*[0-9\p{L}]+$`)
	QueryRetryTimes = 3
	POOL_MAXSIZE    = 10
)

type DNSQuery struct {
	name     string
	rrtype   string
	endpoint string
	dnssec   bool
	insecure bool
}

func MakeDNSQuery(name string, rrtype string, dnssec bool, insecure bool) *DNSQuery {
	return MakeDNSQueryWithCustomEndpoint(name, rrtype, dnssec, insecure, CloudFlareURL)
}

func MakeDNSQueryWithCustomEndpoint(name string, rrtype string, dnssec bool, insecure bool, endpoint string) *DNSQuery {
	return &DNSQuery{
		name:     name,
		rrtype:   rrtype,
		endpoint: endpoint,
		dnssec:   dnssec,
		insecure: insecure,
	}
}

func (c *DNSQuery) Do() (*DNSQueryResponse, error) {
	url := fmt.Sprintf("%s?ct=%s&name=%s&type=%s&do=%s&cd=%s", c.endpoint, WfJSON, c.name, c.rrtype, strconv.FormatBool(c.dnssec), strconv.FormatBool(c.insecure))
	var err error
	var respbytes []byte
	for i := 0; i < QueryRetryTimes; i++ {
		resp, err := httpclient.
			Begin().
			WithHeader("User-Agent", UserAgent).
			Get(url)
		if err != nil {
			continue
		}
		respbytes, err = resp.ReadAll()
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return nil, err
	}
	answer := &DNSQueryResponse{}
	err = json.Unmarshal(respbytes, answer)
	return answer, err
}

type DNSQueryResponse struct {
	Status               int           `json:"Status"`
	Truncated            bool          `json:"TC"`
	RecursiveDesired     bool          `json:"RD"`
	RecursiveAvailable   bool          `json:"RA"`
	DNSSECVerified       bool          `json:"AD"`
	DNSSECVerifyDisabled bool          `json:"CD"`
	Question             []DNSQuestion `json:"Question"`
	Answer               []DNSAnswer   `json:"Answer"`
}
type DNSQuestion struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}
type DNSAnswer struct {
	Type int    `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}
type HostsRecord struct {
	ip             string
	hostname       string
	avgDuration    float64
	testSucessRate float64
}

func testIP(ip string, hostname string) *ping.Result {
	proto, _ := ping.NewProtocol(ping.TCP.String())
	for _, port := range []int{80, 443} {
		target := &ping.Target{
			Timeout:  time.Second * 2,
			Interval: 3,
			Host:     hostname,
			Port:     port,
			Counter:  2,
			Protocol: proto,
		}
		pinger := ping.NewTCPing()
		pinger.SetTarget(target)
		pingerDone := pinger.Start()
		select {
		case <-pingerDone:
			break
		}
		if pinger.Result().SuccessCounter > 0 {
			return pinger.Result()
		}
	}
	return nil
}

func mkhosts(name string, verifyDNSSEC bool, insecure bool) (*HostsRecord, error) {
	if !domainNameRegex.MatchString(name) {
		return nil, errors.New(fmt.Sprintln("%s: Invalid domain name format", name))
	}
	resp, err := MakeDNSQuery(name, "A", verifyDNSSEC, insecure).Do()
	if err != nil {
		return nil, errors.New(fmt.Sprintln("%s: %s", name, err.Error()))
	}
	if !insecure && verifyDNSSEC && !resp.DNSSECVerified {
		return nil, errors.New(fmt.Sprintln("%s: DNSSEC Verify Failed", name))
	}
	records := make([]HostsRecord, 0)
	for _, answer := range resp.Answer {
		if answer.Type == 1 {
			testresult := testIP(answer.Data, name)
			if testresult.SuccessCounter > 0 {
				records = append(records, HostsRecord{
					ip:             answer.Data,
					hostname:       name,
					testSucessRate: float64(testresult.SuccessCounter) / float64(testresult.Counter),
					avgDuration:    testresult.Avg().Seconds() * 1000,
				})
			}
		}
	}
	if len(records) == 0 {
		return nil, errors.New(fmt.Sprintln("%s: No available IPs", name))
	}

	var best int = 0
	for index, record := range records {
		if record.testSucessRate > records[best].testSucessRate || record.testSucessRate == records[best].testSucessRate && record.avgDuration < records[best].avgDuration {
			best = index
		}
	}

	return &records[best], nil
}

func main() {
	usage := `mkhosts <domains> [options]
	Query words meanings via the command line.
	Example:
	  mkhosts www.pixiv.net
	  mkhosts www.pixiv.net www.github.com -s
	Usage:
	  mkhosts <domains>... [-s|--dnssec][-i|--insecure]
	  mkhosts -h | --help
	Options:
	  -s --dnssec   require DNSSEC validation
	  -i --insecure    accept incorrect DNSSEC signatures
	  `
	args, _ := docopt.ParseDoc(usage)
	domains := args["<domains>"].([]string)
	dnssec := args["--dnssec"] != nil && args["--dnssec"] != 0
	insecure := args["--insecure"] != nil && args["--insecure"] != 0
	results := make([]string, 0)
	errors := make([]string, 0)

	wp := workerpool.New(POOL_MAXSIZE)
	resultsmutex := &sync.Mutex{}
	for _, domain := range domains {
		gotdomain := make(chan bool)
		wp.Submit(func() {
			thisdomain := domain
			gotdomain <- true
			hosts, err := mkhosts(thisdomain, dnssec, insecure)
			if err != nil {
				fmt.Println(err.Error())
				errors = append(errors, err.Error())
			} else {
				resultsmutex.Lock()
				results = append(results, fmt.Sprintf("%s %s", hosts.ip, hosts.hostname))
				resultsmutex.Unlock()
			}
		})
		switch {
		case <-gotdomain:
			break
		}
	}
	wp.StopWait()

	if len(errors) != 0 {
		fmt.Println("\n\n\n=========Collected Errors===========")
		for _, errorline := range errors {
			fmt.Println(errorline)
		}
	}

	fmt.Println("\n\n\n===============Results==============")
	for _, resultline := range results {
		fmt.Println(resultline)
	}

}
