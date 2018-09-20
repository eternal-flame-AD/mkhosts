package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

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
	QueryRetryTimes = 5
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
	ip         string
	hostname   string
	testResult TestResult
}

func mkhosts(name string, verifyDNSSEC bool, insecure bool, quiet bool, endpoint string, testmethod Tester) (*HostsRecord, error) {
	if !domainNameRegex.MatchString(name) {
		return nil, fmt.Errorf("%s: Invalid domain name format", name)
	}
	resp, err := MakeDNSQueryWithCustomEndpoint(name, "A", verifyDNSSEC, insecure, endpoint).Do()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", name, err.Error())
	}
	if !insecure && verifyDNSSEC && !resp.DNSSECVerified {
		return nil, fmt.Errorf("%s: DNSSEC Verify Failed", name)
	}

	records := make([]HostsRecord, 0)
	for _, answer := range resp.Answer {
		if answer.Type == 1 {
			testresult := testmethod.TestIP(answer.Data, name, quiet)
			if testresult.success {
				records = append(records, HostsRecord{
					ip:         answer.Data,
					hostname:   name,
					testResult: testresult,
				})
			}
		}
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%s: No available IPs", name)
	}

	var best int = 0
	for index, record := range records {
		if record.testResult.successRate > records[best].testResult.successRate || record.testResult.successRate == records[best].testResult.successRate && record.testResult.delay < records[best].testResult.delay {
			best = index
		}
	}

	return &records[best], nil
}

func determineModeFromArgs(arg interface{}) (string, error) {
	modeIsAvailable := func(mode string) bool {
		_, ok := AvailableTesters[mode]
		return ok
	}
	mode := "tcping"
	if res, ok := arg.(string); ok {
		mode = res
	}
	if res, ok := arg.([]string); ok {
		if len(res) != 0 {
			mode = res[0]
		}
	}
	if modeIsAvailable(mode) {
		return mode, nil
	} else {
		return "", errors.New("Mode not found.")
	}
}

func main() {
	usage := `mkhosts <domains> [options]
	Query words meanings via the command line.
	Example:
	  mkhosts www.pixiv.net
	  mkhosts www.pixiv.net www.github.com -s
	  mkhosts -f domainlists/pixiv.net -q >hosts
	Usage:
	  mkhosts [<domains>|-f <domainlist>|--file <domainlist>]... [-m <mode>|--mode <mode>][-s|--dnssec][-i|--insecure][-w|--write][-q|--quiet][-e <endpoint>|--endpoint <endpoint>]
	  mkhosts -h | --help
	Options:
	  -s --dnssec                  require DNSSEC validation
	  -i --insecure                accept incorrect DNSSEC signatures
	  -w --write                   write hosts directly(requires priviledge)
	  -f --file                    read domains from domainlist
	  -q --quiet                   ignore infos and errors, output hosts directly to stdout
	  -e, --endpoint <endpoint>    custom endpoint. default: https://1.1.1.1/dns-query
	  -m, --mode <mode>            test mode. default: tcping
	
	Internal domain lists:
`
	for _, val := range reflect.ValueOf(InternalDomainLists).MapKeys() {
		key := val.String()
		usage += "\t\t" + key + "\n"
	}
	usage += "\n\tTest modes:\n"

	for _, val := range reflect.ValueOf(AvailableTesters).MapKeys() {
		key := val.String()
		usage += "\t\t" + key + "\n"
	}

	args, _ := docopt.ParseDoc(usage)
	errorlist := make([]string, 0)

	domainfiles := StringSliceOrEmpty(args["<domainlist>"])

	domains := StringSliceOrEmpty(args["<domains>"])

	for _, fn := range domainfiles {
		var contentstr string
		content, ok := InternalDomainLists[fn]
		if ok {
			contentstr = content
		} else {
			content, err := ioutil.ReadFile(fn)
			contentstr = string(content)
			if err != nil {
				errstr := fmt.Sprintf("Error reading domainlist %s: %s\n", fn, err.Error())
				errorlist = append(errorlist, errstr)
				fmt.Println(errstr)
				continue
			}
		}

		LineBreak := detectLineBreakFromString(contentstr)
		contentlines := strings.Split(contentstr, LineBreak)
		for _, line := range contentlines {
			domain := domainNameRegex.FindString(line)
			if len(domain) > 0 {
				domains = append(domains, domain)
			}
		}
	}
	domains = removeRepByLoop(domains)
	if len(domains) == 0 {
		docopt.PrintHelpAndExit(errors.New("No hostname specified"), usage)
	}

	dnssec := args["--dnssec"] != nil && args["--dnssec"] != 0
	insecure := args["--insecure"] != nil && args["--insecure"] != 0
	writehosts := args["--write"] != nil && args["--write"] != 0
	quiet := args["--quiet"] != nil && args["--quiet"] != 0
	endpoint := append(StringSliceOrEmpty(args["--endpoint"]), CloudFlareURL)[0] // CloudFlareURL if empty
	testmode, err := determineModeFromArgs(args["--mode"])
	if err != nil {
		docopt.PrintHelpAndExit(err, usage)
	}
	results := make([]HostsRecord, 0)

	wp := workerpool.New(POOL_MAXSIZE)
	resultsmutex := &sync.Mutex{}
	for _, domain := range domains {
		gotdomain := make(chan bool)
		tester := AvailableTesters[testmode]
		wp.Submit(func() {
			thisdomain := domain
			thistester := tester
			gotdomain <- true
			hosts, err := mkhosts(thisdomain, dnssec, insecure, quiet, endpoint, thistester)
			if err != nil {
				fmt.Println(err.Error())
				errorlist = append(errorlist, err.Error())
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

	if !quiet && len(errorlist) != 0 {
		fmt.Println("\n\n\n=========Collected Errors===========")
		for _, errorline := range errorlist {
			fmt.Println(errorline)
		}
	}
	if len(results) != 0 {
		if !quiet {
			fmt.Println("\n\n\n===============Results==============")
		}
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

}
