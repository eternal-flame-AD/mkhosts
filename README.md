# mkhosts
一个自动生成hosts文件绕过dns污染的工具，采用DNSoverHTTPS绕过国内DNS,适用于P站等未遭到全面TCP/IP封禁的网站，自动测试tcp链接可靠性，解决各ISP情况不同有的别人能用的hosts自己却用不了的问题

## Installation

## 从git下载并编译
```bash
go install github.com/eternal-flame-AD/mkhosts
```
## 下载release

从[发布页面](https://github.com/eternal-flame-AD/mkhosts/releases/latest)下载对应平台二进制文件

## Usage

mkhosts可以从每行一个的域名列表和现有的hosts文件中提取域名,也可以从cli读入域名

cli指定域名:
```bash
mkhosts www.pixiv.net accounts.pixiv.net app-api.pixiv.net
```
读入hosts/域名文件:
```bash
mkhosts -f domainlists/pixiv.txt -f mycustomdomainlist.txt
```

之后将生成的结果复制到hosts文件中即可

## Notices

mkhosts仅仅提供干净的dns解析结果，不能提高链接安全性和可靠性，**请注意合法使用**

## TODO

- 更多的domainlists
- <s>自动写入hosts文件</s>
- <s>读取现有hosts文件并做更新</s>
