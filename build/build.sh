env GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o mkhosts-linux-386 -v github.com/eternal-flame-AD/mkhosts
env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mkhosts-linux-amd64 -v github.com/eternal-flame-AD/mkhosts
env GOOS=windows GOARCH=386 go build -ldflags="-s -w" -o mkhosts-windows-386.exe -v github.com/eternal-flame-AD/mkhosts
env GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o mkhosts-windows-amd64.exe -v github.com/eternal-flame-AD/mkhosts
upx --lzma mkhosts-*
