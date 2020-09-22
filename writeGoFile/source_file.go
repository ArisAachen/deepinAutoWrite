package writeGoFile

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type SourceFile struct {
	Pkg       string
	GoImports []string
	GoBody    *SourceBody
}

// source file
func NewSourceFile(pkg string) *SourceFile {
	sf := &SourceFile{
		Pkg:    pkg,
		GoBody: &SourceBody{},
	}
	return sf
}

func (v *SourceFile) Print() error {
	_, err := v.WriteTo(os.Stdout)
	return err
}

func (v *SourceFile) Save(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal("fail to create file:", err)
	}
	defer f.Close()
	_, err = v.WriteTo(f)
	if err != nil {
		log.Fatal("failed to write to file:", err)
	}

	out, err := exec.Command("go", "fmt", filename).CombinedOutput()
	if err != nil {
		log.Printf("%s", out)
		log.Fatal("failed to format file:", filename)
	}
}

func (v *SourceFile) WriteTo(w io.Writer) (n int64, err error) {
	var wn int
	wn, err = io.WriteString(w, "package "+v.Pkg+"\n")
	n += int64(wn)
	if err != nil {
		return
	}

	sort.Strings(v.GoImports)
	for _, imp := range v.GoImports {
		wn, err = io.WriteString(w, "import "+imp+"\n")
		n += int64(wn)
		if err != nil {
			return
		}
	}

	wn, err = w.Write(v.GoBody.buf.Bytes())
	n += int64(wn)
	return
}

// unsafe => "unsafe"
// or x,github.com/path/ => x "path"
func (s *SourceFile) AddGoImport(imp string) {
	var importStr string
	if strings.Contains(imp, ",") {
		parts := strings.SplitN(imp, ",", 2)
		importStr = fmt.Sprintf("%s %q", parts[0], parts[1])
	} else {
		importStr = `"` + imp + `"`
	}

	for _, imp0 := range s.GoImports {
		if imp0 == importStr {
			return
		}
	}
	s.GoImports = append(s.GoImports, importStr)
}

// source body
type SourceBody struct {
	buf bytes.Buffer
}

func (v *SourceBody) writeStr(str string) {
	v.buf.WriteString(str)
}

func (v *SourceBody) Pn(format string, a ...interface{}) {
	v.P(format, a...)
	v.buf.WriteByte('\n')
}

func (v *SourceBody) P(format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	v.writeStr(str)
}

func (v *SourceBody) WriteDBusObjects(objects []*DBusObject) {
	for _, object := range objects {
		writeStruct(v, object)
		writeNewObject(v, object)
		writeImplementerMethods(v, object)

		// write method
		for _, method := range object.methods {
			writeMethod(v, object.ObjectName, method)
		}

		// write signal
		for _, signal := range object.signals {
			writeSignal(v, object.ObjectName, signal)
		}

		// write property
		for _, property := range object.properties {
			writeProperty(v, object.ObjectName, property)
		}
	}
}
