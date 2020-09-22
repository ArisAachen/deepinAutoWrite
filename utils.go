package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/types"
	"log"
	"os/exec"
	"strings"
)

// find type in info according to name
func findDef(info types.Info, name string) types.Type {
	for ident, obj := range info.Defs {
		// check if equals
		if ident.String() == name {
			return obj.Type()
		}
	}
	return nil
}

// format sources to replace mark text
func FormatImplementers(sources []string) string {
	commaStr := strings.Join(sources, ",")
	fmt.Print(commaStr)
	// implement joint
	target := "var implementers = []dbusutil.Implementer{" + commaStr + "}"
	return target
}

// execute unit test file with go test
func ExecuteUnitTest(file string) error {
	log.Println("write to file success")
	// if update success, execute write DBus xml
	cmd := exec.Command("go", "test", file, "-count=1", "-v", "-run=TestDBusWriteXml")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// run command
	if err := cmd.Run(); err != nil {
		log.Println("run command failed, out: ", stdout.String())
		log.Println("run command failed, err: ", stderr.String())
		return errors.New(stdout.String())
	} else {
		log.Println("run command go test success")
	}
	return nil
}

func ReplaceText(source string, old string, new string) string {
	return strings.Replace(source, old, new, -1)
}

// unique string slice
func UniqueSlice(multi []string) []string {
	// create map
	uniqueMap := make(map[string]string)
	// use map to unique slice
	for _, elem := range multi {
		// check if elem is empty
		if elem == "" {
			continue
		}
		// add elem to map
		uniqueMap[elem] = elem
	}
	// create slice
	var uniqueSlice []string
	// add map elem to slice
	for _, elem := range uniqueMap {
		// check if elem is empty
		if elem == "" {
			continue
		}
		// add elem to slice
		uniqueSlice = append(uniqueSlice, elem)
	}
	return uniqueSlice
}

// importer package
type importer struct{}

func (v *importer) Import(path0 string) (*types.Package, error) {
	return types.NewPackage(path0, ""), nil
}

var ModuleStr = `
replacePackageMark
import (
	"pkg.deepin.io/lib/dbusutil"
	"testing"
)
replaceInterfaceMark
// go test -count=1 -v -run=TestDBusWriteXml
func TestDBusWriteXml(t *testing.T){
	for _, value := range implementers {
		dbusutil.WriteXML(value)
	}
}`
