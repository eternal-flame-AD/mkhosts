# mkhosts
一个自动生成hosts文件绕过dns污染的工具，采用DNSoverHTTPS绕过国内DNS,适用于P站等未遭到全面TCP/IP封禁的网站，自动测试tcp链接可靠性，解决各ISP情况不同有的别人能用的hosts自己却用不了的问题

## Installation

```bash
go install github.com/eternal-flame-AD/mkhosts
```

## Usage

```bash
mkhosts www.pixiv.net accounts.pixiv.net app-api.pixiv.net
```
之后将生成的结果复制到hosts文件中即可

## Notices

mkhosts仅仅提供干净的dns解析结果，不能提高链接安全性和可靠性，**请注意合法使用**

## TODO

- 自动写入hosts文件
- 读取现有hosts文件并做更新