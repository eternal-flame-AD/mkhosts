package main

import (
	"io/ioutil"
	"os"
	"path"
	"text/template"
)

type internalDomainListFile struct {
	Name    string
	Content string
}

func main() {
	here, _ := os.Getwd()
	projdir := path.Join(here, "./")
	files, _ := ioutil.ReadDir(path.Join(projdir, "domainlists/"))
	resList := make([]internalDomainListFile, 0)
	for _, file := range files {
		fname := file.Name()
		thisfn := path.Join(projdir, "domainlists/", fname)
		filecontent, _ := ioutil.ReadFile(thisfn)
		resList = append(resList, internalDomainListFile{fname, string(filecontent)})
	}
	template, err := template.New("internaldomainlist").Parse(`// generated by go run buildutils.go; DO NOT EDIT
package main

var InternalDomainLists = map[string]string{
{{range .}}	"{{.Name}}": ` + "`" + `{{.Content}}` + "`,\n" + `{{end}}}
`)
	if err != nil {
		panic(err)
	}
	outfile, err := os.Create(path.Join(projdir, "internaldomainlist.go"))
	defer outfile.Close()
	if err != nil {
		panic(err)
	}
	if err := template.Execute(outfile, resList); err != nil {
		panic(err)
	}
}