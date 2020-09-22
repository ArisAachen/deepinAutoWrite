package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unicode"

	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"

	gofile "./writeGoFile"
)

// interface type
var _implementerIfc *types.Interface

// interface method, use to check if module implement this method
const tmpCode = `
package dbusutil

type Implementer interface {
	GetInterfaceName() string
}
`

// mark need replaced interfaces in module file
var replaceInterface = "replaceInterfaceMark"

// mark need replaced package in module file
var replacePackage = "replacePackageMark"

// unit test name
var writeTestFileName = "Test_%s_DBusWriteXml_test.go"

var writeXml = false
var writeGo = false

// func main
func main() {
	// read param
	var file string
	flag.StringVar(&file, "filePath", "", "")
	flag.BoolVar(&writeXml, "writeXml", false, "")
	flag.BoolVar(&writeGo, "writeGo", false, "")
	flag.Parse()
	// set log flags
	log.SetFlags(log.Lshortfile)
	// parse code
	parseTmpCode()

	// convert absolute path
	if filepath.IsAbs(file) {
		// get current path
		current, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
			return
		}
		relFile, err := filepath.Rel(current, file)
		if err != nil {
			log.Printf("convert absolute path to rel path failed, err: %v \n", err)
			return
		}
		file = relFile
	}

	// write and execute unittest
	err := WriteAndExecuteUnitTest(file)
	if err != nil {
		log.Println("write and execute file failed, err: ", err)
	}
}

// write and execute unit test
func WriteAndExecuteUnitTest(file string) error {
	// walk filepath
	err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
		// check if open file failed
		if err != nil {
			return err
		}
		// check if path is empty
		if path == "" {
			return nil
		}
		// check if is dir
		if !info.IsDir() {
			return nil
		}
		// get interface map
		interfacesMap, busObjects, err := GetInterfaces(path)
		if err != nil {
			log.Println("get interface map failed, err: ", err)
			return err
		}

		// check if interface map dont has instance implement interface method
		if len(interfacesMap) == 0 {
			log.Println(path, " interface is empty")
			return nil
		}

		// unique slice
		for key, value := range interfacesMap {
			interfacesMap[key] = UniqueSlice(value)
		}

		if writeGo {
			for pkg, busObject := range busObjects {
				sf := gofile.NewSourceFile(pkg)

				sf.AddGoImport("errors")
				sf.AddGoImport("fmt")
				sf.AddGoImport("unsafe")
				sf.AddGoImport("github.com/godbus/dbus")
				sf.AddGoImport("pkg.deepin.io/lib/dbusutil")
				sf.AddGoImport("pkg.deepin.io/lib/dbusutil/proxy")

				sf.GoBody.Pn("/* prevent compile error */")
				sf.GoBody.Pn("var _ = errors.New")
				sf.GoBody.Pn("var _ dbusutil.SignalHandlerId")
				sf.GoBody.Pn("var _ = fmt.Sprintf")
				sf.GoBody.Pn("var _ unsafe.Pointer")
				sf.GoBody.Pn("")

				sf.GoBody.WriteDBusObjects(busObject)
				_ = sf.Print()
			}
		}

		if writeXml {
			// create xml file
			for pkg, interfaces := range interfacesMap {
				// check if package is empty or instances is nil
				if pkg == "" || len(interfaces) == 0 {
					continue
				}
				// get replace text
				replaceText := FormatImplementers(interfaces)
				// target file path
				// replace instance in module
				resultText := ReplaceText(ModuleStr, replaceInterface, replaceText)
				targetPath := fmt.Sprintf(path+"/"+writeTestFileName, pkg)
				// replace package in target path
				resultText = ReplaceText(resultText, replacePackage, "package "+pkg)
				fObj, err := os.Create(targetPath)
				if err != nil {
					log.Println("open or create target file failed, err: ", err)
					continue
				}
				// write string to file
				_, err = fObj.WriteString(resultText)
				if err != nil {
					log.Println("write string to file failed, err: ", err)
					continue
				}
				// check if need update file, if not, dont need to create unit test file
				err = ExecuteUnitTest(path)
				if err != nil {
					log.Println("execute unit test failed, err: ", err)
					continue
				}
				if err := os.Remove(targetPath); err != nil {
					log.Println("remove file failed, err: ", err)
					return err
				}
			}
		}
		return nil
	})
	return err
}

func GetInterfaces(filepath string) (map[string][]string, map[string][]*gofile.DBusObject, error) {
	interfacesMap := make(map[string][]string)
	busObjects := make(map[string][]*gofile.DBusObject)
	var busContainer = gofile.NewDBusContainer()
	cfg := &packages.Config{
		Mode: packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, filepath)
	if err != nil {
		log.Println("load package error, ", err)
		return nil, nil, err
	}
	for _, pkg := range pkgs {
		var files []*ast.File
		fSet := token.NewFileSet()
		if len(pkg.GoFiles) == 0 {
			continue
		}
		for _, goFile := range pkg.GoFiles {
			f, err := parser.ParseFile(fSet, goFile, nil, 0)
			if err != nil {
				log.Println(err)
				continue
			}
			files = append(files, f)
		}

		info := types.Info{
			Defs:   make(map[*ast.Ident]types.Object),
			Scopes: make(map[ast.Node]*types.Scope),
		}
		var conf types.Config
		conf.Error = func(err error) {
		}
		conf.Importer = &importer{}
		_, _ = conf.Check(pkg.PkgPath, fSet, files, &info)

		for iNode, _ := range info.Scopes {
			if iNode, ok := iNode.(*ast.File); ok {
				busEls := gofile.GetDBusPathName(iNode)
				if busEls == nil {
					continue
				}
				busContainer.AddDBusElem(busEls...)
			}
		}

		// refresh
		busContainer.RefreshDBusObj(files)
		busContainer.RefreshDBusPath(files)
		busContainer.RefreshDBusInterface(files)

		for ident, obj := range info.Defs {
			if obj == nil {
				continue
			}
			named, ok := obj.Type().(*types.Named)
			if !ok {
				continue
			}
			_, ok = named.Underlying().(*types.Struct)
			if !ok {
				// 不是 struct
				continue
			}
			pNamed := types.NewPointer(named)
			if !types.Implements(pNamed, _implementerIfc) {
				// 没有实现 Implementer 接口
				continue
			}
			if unicode.IsLower(rune(ident.Name[0])) {
				continue
			}

			busElem := busContainer.GetDBusElemByObj(ident.Name)
			busObject := gofile.NewDBusObject()
			busObject.SetDBusPath(busElem.DBusPath)
			busObject.SetTypesNamed(named)
			busObjects[obj.Pkg().Name()] = append(busObjects[obj.Pkg().Name()], busObject)

			element := fmt.Sprintf("&%v{}", ident)
			interfacesMap[obj.Pkg().Name()] = append(interfacesMap[obj.Pkg().Name()], element)

		}
	}
	return interfacesMap, busObjects, nil
}

// record _implementerIfc message
func parseTmpCode() {
	// parse tmp code
	fSet := token.NewFileSet()
	f, err := parser.ParseFile(fSet, "", tmpCode, 0)
	if err != nil {
		log.Println(err)
	}

	info := types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	}
	var conf types.Config
	pkg, err := conf.Check("dbusutil", fSet, []*ast.File{f}, &info)
	if err != nil {
		log.Println(err)
	}
	_ = pkg
	// find implementer in info
	impl := findDef(info, "Implementer")
	if impl != nil {
		// check if is interface
		if types.IsInterface(impl) {
			_implementerIfc = impl.Underlying().(*types.Interface)
		}
	}
}
