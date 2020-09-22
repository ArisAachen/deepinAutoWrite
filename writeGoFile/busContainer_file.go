package writeGoFile

import (
	"go/ast"
)

type DBusContainer struct {
	busSlice []*DBusElem
}

// DBusElem
type DBusElem struct {
	// real name
	DBusPath      string
	DBusObjName   string
	DBusInterface string

	// identity
	DBusConst string
	DBusInfo  *StructInfo
}

type StructInfo struct {
	StructName string
	Elem       string
}

func NewDBusContainer() *DBusContainer {
	return &DBusContainer{
		busSlice: []*DBusElem{},
	}
}

func (container *DBusContainer) GetDBusElemByObj(objName string) *DBusElem {
	for _, elem := range container.busSlice {
		if elem.DBusObjName == objName {
			return elem
		}
	}
	return nil
}

func (container *DBusContainer) AddDBusElem(elem ...*DBusElem) {
	if container.busSlice == nil {
		return
	}
	container.busSlice = append(container.busSlice, elem...)
}

func (container *DBusContainer) RefreshDBusPath(astFiles []*ast.File) {
	// refresh DBusPath
	for _, elem := range container.busSlice {
		// check if DBusPath is empty
		if elem.DBusPath == "" {
			for _, astFile := range astFiles {
				objs := astFile.Scope.Objects
				for key, obj := range objs {
					if elem.DBusConst == key {
						elem.DBusPath = GetDBusPathFromObj(obj)
					}
				}
			}
		}
	}
}

func (container *DBusContainer) RefreshDBusObj(astFiles []*ast.File) {
	for _, elem := range container.busSlice {
		if elem.DBusObjName == "" {
			for _, astFile := range astFiles {
				objs := astFile.Scope.Objects
				for key, obj := range objs {
					if elem.DBusInfo.StructName == key {
						elem.DBusObjName = GetDBusObjNameFromObj(obj, elem.DBusInfo)
					}
				}
			}
		}
	}
}

func (container *DBusContainer) RefreshDBusInterface(astFiles []*ast.File) {
	for _, elem := range container.busSlice {
		if elem.DBusObjName == "" {
			continue
		}
		for _, astFile := range astFiles {
			objs := astFile.Decls
			for _, obj := range objs {
				if funcDecl, ok := obj.(*ast.FuncDecl); ok {
					funcName := funcDecl.Name
					if funcName == nil {
						continue
					}
					if funcName.Name != "GetInterfaceName" {
						continue
					}
					recvList := funcDecl.Recv.List
					if len(recvList) == 0 {
						continue
					}
					recvType := recvList[0].Type
					starExpr, ok := recvType.(*ast.StarExpr)
					if !ok {
						continue
					}
					expr := starExpr.X
					xIdent, ok := expr.(*ast.Ident)
					if !ok {
						continue
					}

					if xIdent.Name == elem.DBusObjName {
						bodyList := funcDecl.Body.List
						for _, body := range bodyList {
							if rtStmt, ok := body.(*ast.ReturnStmt); ok {
								rlt := rtStmt.Results
								if len(rlt) == 0 {
									continue
								}
								ident, ok := rlt[0].(*ast.Ident)
								if !ok {
									continue
								}
								if ident.Obj == nil {
									continue
								}
								itf := GetDBusPathFromObj(ident.Obj)
								elem.DBusInterface = itf
							}
						}
					}
				}
			}
		}
	}
}
