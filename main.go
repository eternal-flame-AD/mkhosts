package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
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
	domainNameRegex = regexp.MustCompile(`[0-9\p{L}][0-9\p{L}-\.]{1,61}[0-9\p{L}]\.[0-9\p{L}][\p{L}-]*[\p{L}]+`)
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

func testIP(ip string) *ping.Result {
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
		return nil, errors.New(fmt.Sprintf("%s: Invalid domain name format", name))
	}
	resp, err := MakeDNSQuery(name, "A", verifyDNSSEC, insecure).Do()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("%s: %s", name, err.Error()))
	}
	if !insecure && verifyDNSSEC && !resp.DNSSECVerified {
		return nil, errors.New(fmt.Sprintf("%s: DNSSEC Verify Failed", name))
	}
	records := make([]HostsRecord, 0)
	for _, answer := range resp.Answer {
		if answer.Type == 1 {
			testresult := testIP(answer.Data)
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
		return nil, errors.New(fmt.Sprintf("%s: No available IPs", name))
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
	  mkhosts [<domains>|-f <domainlist>|--file <domainlist>]... [-s|--dnssec][-i|--insecure][-w|--write]
	  mkhosts -h | --help
	Options:
	  -s --dnssec      require DNSSEC validation
	  -i --insecure    accept incorrect DNSSEC signatures
	  -w --write       write hosts directly(requires priviledge)
	  -f --file        read domains from domainlist
	  `
	args, _ := docopt.ParseDoc(usage)
	errors := make([]string, 0)
	domainfiles := args["<domainlist>"].([]string)
	domains := args["<domains>"].([]string)
	for _, fn := range domainfiles {
		content, err := ioutil.ReadFile(fn)
		contentstr := string(content)
		if err != nil {
			errstr := fmt.Sprintf("Error reading domainlist %s: %s\n", fn, err.Error())
			errors = append(errors, errstr)
			fmt.Println(errstr)
			continue
		}
		var LineBreak string
		switch {
		case strings.Contains(contentstr, "\r\n"):
			LineBreak = "\r\n"
			break
		case strings.Contains(contentstr, "\n"):
			LineBreak = "\n"
			break
		case strings.Contains(contentstr, "\r"):
			LineBreak = "\r"
			break
		default:
			LineBreak = "\n"
		}
		contentlines := strings.Split(contentstr, LineBreak)
		for _, line := range contentlines {
			domain := domainNameRegex.FindString(line)
			if len(domain) > 0 {
				domains = append(domains, domain)
			}
		}
	}
	domains = removeRepByLoop(domains)

	dnssec := args["--dnssec"] != nil && args["--dnssec"] != 0
	insecure := args["--insecure"] != nil && args["--insecure"] != 0
	writehosts := args["--write"] != nil && args["--write"] != 0
	results := make([]HostsRecord, 0)

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
				results = append(results, *hosts)
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
		fmt.Println(fmt.Sprintf("%s %s", resultline.ip, resultline.hostname))
	}
	if writehosts {
		err := addHosts(results)
		if err != nil {
			fmt.Println(err.Error())
		}
	}

}
