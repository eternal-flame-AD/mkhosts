package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"
)

func checkForAdmin() bool {
	if runtime.GOOS == "windows" {
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		if err != nil {
			return false
		}
		return true
	} else {
		fp, err := os.OpenFile("/etc/hosts", os.O_RDWR, 0644)
		if err != nil {
			return false
		}
		defer fp.Close()
		return true
	}
}

func getHostsLocation() string {
	if runtime.GOOS == "windows" {
		return `%SystemDrive%\Windows\System32\drivers\etc\hosts`
	}
	return `/etc/hosts`
}

func addHosts(records []HostsRecord) error {
	if !checkForAdmin() {
		return errors.New("Must run with priviledged user to modify hosts")
	}
	hostslocation := getHostsLocation()
	b, err := ioutil.ReadFile(hostslocation)
	if err != nil {
		return err
	}
	originalhosts := string(b)
	var LineBreak string
	switch {
	case strings.Contains(originalhosts, "\r\n"):
		LineBreak = "\r\n"
		break
	case strings.Contains(originalhosts, "\n"):
		LineBreak = "\n"
		break
	case strings.Contains(originalhosts, "\r"):
		LineBreak = "\r"
		break
	default:
		LineBreak = "\n"
		break
	}
	for _, record := range records {
		substituionexp := regexp.MustCompile(fmt.Sprintf(`[a-fA-F\d\.\:]+\s+%s(.*)`, strings.Replace(record.hostname, ".", "\\.", -1)))
		if occurences := substituionexp.FindAllString(originalhosts, -1); len(occurences) != 0 {
			replaced := false
			for _, occurence := range occurences {
				if !replaced {
					originalhosts = strings.Replace(originalhosts, occurence, fmt.Sprintf("%s %s", record.ip, record.hostname), 1)
					replaced = true
				}
				originalhosts = strings.Replace(originalhosts, occurence, "", -1)
			}
		} else {
			if !strings.HasSuffix(originalhosts, "\n") && !strings.HasSuffix(originalhosts, "\r") {
				originalhosts += LineBreak
			}
			originalhosts += fmt.Sprintf("%s %s", record.ip, record.hostname) + LineBreak
		}
	}
	fmt.Println("\n\n\n==========Modified Hosts file ===========")
	fmt.Print(originalhosts)

	fmt.Print("Write?[y/n]")
	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil {
		return err
	}
	switch char {
	case 'y':
		fallthrough
	case 'Y':
		err := ioutil.WriteFile(hostslocation, []byte(originalhosts), 0644)
		if err != nil {
			return err
		}
		fmt.Println("Success")
		break
	default:
		fmt.Println("Giving up...")
	}

	return nil
}
