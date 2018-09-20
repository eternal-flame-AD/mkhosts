# mkhosts [![Build Status](https://travis-ci.org/eternal-flame-AD/mkhosts.svg?branch=master)](https://travis-ci.org/eternal-flame-AD/mkhosts)
一个自动生成hosts文件绕过dns污染的工具，采用DNSoverHTTPS绕过国内DNS,适用于P站等未遭到全面TCP/IP封禁的网站，自动测试tcp链接可靠性，解决各ISP情况不同有的别人能用的hosts自己却用不了的问题

## Installation

## 从git下载并编译
```bash
go get -u github.com/eternal-flame-AD/mkhosts
```
## 下载release

从[发布页面](https://github.com/eternal-flame-AD/mkhosts/releases/latest)下载对应平台二进制文件

## Usage

mkhosts可以从每行一个的域名列表和现有的hosts文件中提取域名,也可以从cli读入域名

```
mkhosts <domains> [options]
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
                pixiv
                arukas

        Test modes:
                tcping
                ssl
```

cli指定域名:
```bash
mkhosts www.pixiv.net accounts.pixiv.net app-api.pixiv.net
```
读入hosts/域名文件/内置域名列表(目前有pixiv和arukas两个):
```bash
mkhosts -f pixiv -f mycustomdomainlist.txt
```
静默执行，直接将结果追加到hosts:
```bash
sudo mkhosts -f pixiv -q >> /etc/hosts
```
将结果写入hosts（自动替换重复域名）:
```bash
sudo mkhosts -f pixiv -w
```
测试ssl握手：
```bash
mkhosts -f pixiv -m ssl
```

## Notices

mkhosts仅仅提供干净的dns解析结果，不能提高链接安全性和可靠性，**请注意合法使用**

## TODO

- 更多的domainlists
- <s>自动写入hosts文件</s>
- <s>读取现有hosts文件并做更新</s>
