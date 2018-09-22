package main

import (
	"testing"
)

func TestDNSQuery(t *testing.T) {
	query := MakeDNSQuery("www.google.com", "A", false, false)
	res, err := query.Do()
	if err != nil {
		t.FailNow()
	}
	if res.Status != 0 {
		// return status is not ok
		t.Errorf("TestDNSQuery: Remote returned with status %d", res.Status)
		t.FailNow()
	}
	if len(res.Answer) == 0 {
		//no answer detected
		t.Error("TestDNSQuery: No available answer")
		t.FailNow()
	}
}

func TestDNSSECVerified(t *testing.T) {
	query := MakeDNSQuery("dnssec-tools.org", "A", true, false)
	res, err := query.Do()
	if err != nil {
		t.Errorf("TestDNSSECVerified: *DNSQuery.Do() returned error: %s", err.Error())
		t.FailNow()
	}
	if !res.DNSSECVerified {
		t.Error("TestDNSSECVerified: DNSSECVerified is false")
		t.FailNow()
	}
	if res.DNSSECVerifyDisabled {
		t.Error("TestDNSSECVerified: DNSSECVerifyDisabled is true")
		t.FailNow()
	}
	if len(res.Answer) == 0 {
		//no answer detected
		t.Error("TestDNSSECVerified: No available answer")
		t.FailNow()
	}
}

func TestDNSSECFailed(t *testing.T) {
	query := MakeDNSQuery("dnssec-failed.org", "A", true, false)
	res, err := query.Do()
	if err != nil {
		t.Errorf("TestDNSSECFailed: *DNSQuery.Do() returned error: %s", err.Error())
		t.FailNow()
	}
	if res.DNSSECVerified {
		t.Error("TestDNSSECFailed: DNSSECVerified is true")
		t.FailNow()
	}
	if res.DNSSECVerifyDisabled {
		t.Error("TestDNSSECFailed: DNSSECVerifyDisabled is true")
		t.FailNow()
	}
	if len(res.Answer) != 0 {
		//no answer detected
		t.Error("TestDNSSECFailed: Has answer")
		t.FailNow()
	}
}

func TestDNSSECFailedInsecure(t *testing.T) {
	query := MakeDNSQuery("dnssec-failed.org", "A", true, true)
	res, err := query.Do()
	if err != nil {
		t.Errorf("TestDNSSECFailedInsecure: *DNSQuery.Do() returned error: %s", err.Error())
		t.FailNow()
	}
	if res.DNSSECVerified {
		t.Error("TestDNSSECFailedInsecure: DNSSECVerified is true")
		t.FailNow()
	}
	if !res.DNSSECVerifyDisabled {
		t.Error("TestDNSSECFailedInsecure: DNSSECVerifyDisabled is false")
		t.FailNow()
	}
	if len(res.Answer) == 0 {
		//no answer detected
		t.Error("TestDNSSECFailedInsecure: No available answer")
		t.FailNow()
	}
}

func TestTCPing(t *testing.T) {
	tester := TCPingTester{}
	res := tester.TestIP("1.1.1.1", "test", true)
	if !res.success {
		t.Error("TestTCPing: failure")
		t.FailNow()
	}
	res = tester.TestIP("224.0.0.1", "test", true) // unreachable
	if res.success {
		t.Error("TestTCPing: sucess")
		t.FailNow()
	}
}

func TestSSL(t *testing.T) {
	tester := SSLTester{}
	res := tester.TestIP("1.1.1.1", "www.cloudflare.com", true)
	if !res.success {
		t.Error("TestSSLHandshake: failure")
		t.FailNow()
	}
	res = tester.TestIP("224.0.0.1", "test", true) // unreachable
	if res.success {
		t.Error("TestSSLHandshake: sucess")
		t.FailNow()
	}
}
