env GOOS=linux GOARCH=386 go build -o mkhosts-linux-386 -v github.com/eternal-flame-AD/mkhosts
env GOOS=linux GOARCH=amd64 go build -o mkhosts-linux-amd64 -v github.com/eternal-flame-AD/mkhosts
env GOOS=windows GOARCH=386 go build -o mkhosts-windows-386.exe -v github.com/eternal-flame-AD/mkhosts
env GOOS=windows GOARCH=amd64 go build -o mkhosts-windows-amd64.exe -v github.com/eternal-flame-AD/mkhosts
upx --best mkhosts-*
