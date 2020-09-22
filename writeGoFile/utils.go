package writeGoFile

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"reflect"
	"strings"
	"unicode"
)

// check if var is property
func IsProperty(object *types.Var) bool {
	if object == nil {
		return false
	}
	// check if is title
	name := object.Name()
	if unicode.IsLower(rune(name[0])) {
		return false
	}

	// check if type is signal
	if name == "signals" || name == "methods" {
		return false
	}

	// check if type is valid
	var buf bytes.Buffer
	types.WriteType(&buf, object.Type(), nil)
	if buf.String() == "invalid type" {
		return false
	}
	return true
}

func IsSignals(object *types.Var) bool {
	if object == nil {
		return false
	}
	// check if is title
	name := object.Name()

	// check if type is signal
	if name == "signals" {
		return true
	}

	return false
}

func IsMethod(object *types.Func) bool {
	if object == nil {
		return false
	}

	// check if is title
	name := object.Name()

	if unicode.IsLower(rune(name[0])) {
		return false
	}

	if name == "GetInterfaceName" {
		return false
	}

	return true
}

func IsInterface(object *types.Func) bool {
	if object == nil {
		return false
	}

	// check if is title
	name := object.Name()

	if unicode.IsLower(rune(name[0])) {
		return false
	}

	if name == "GetInterfaceName" {
		return true
	}

	return false
}

func writeNewObject(sb *SourceBody, object *DBusObject) {
	sb.Pn("func New%s(conn *dbus.Conn) %s {", object.TypeName,
	)

	sb.Pn("obj := new(%s)", object.TypeName)

	sb.Pn("obj.Object.Init_(conn, %s, %s)", object.interfaceName, object.busPath)

	sb.Pn("return %obj")
	sb.Pn("}\n")
}

func writeStruct(sb *SourceBody, object *DBusObject) {
	log.Println("Object", object.TypeName)
	sb.Pn("type %s struct {", object.TypeName)
	sb.Pn("%s // interface %s", object.ObjectName, object.interfaceName)
	sb.Pn("proxy.Object")
	sb.Pn("}\n")
}

func writeImplementerMethods(sb *SourceBody, object *DBusObject) {
	sb.Pn("type %s struct{}", object.ObjectName)

	sb.Pn("func (v *%s) GetObject_() *proxy.Object {", object.ObjectName)
	sb.Pn("    return (*proxy.Object)(unsafe.Pointer(v))")
	sb.Pn("}\n")

	sb.Pn("func (*%s) GetInterfaceName_() string {", object.ObjectName)
	sb.Pn("    return %q", object.interfaceName)
	sb.Pn("}\n")
}

func writeMethod(sb *SourceBody, ObjectName string, method *types.Func) {
	// sb.Pn("// method %s\n", method.Name())
	methodName := strings.Title(method.Name())

	// check if method name is GetInterface
	if method.Name() == "GetInterfaceName" {
		return
	}

	// convert to signature
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		log.Print("convert to signature failed")
		return
	}
	// get params
	params := filterTuple(signature.Params())
	paramsComma := ", "
	if len(params) == 0 {
		paramsComma = ""
	}
	// GoXXX
	sb.Pn("func (v *%s) Go%s(flags dbus.Flags, ch chan *dbus.Call%s) *dbus.Call {",
		ObjectName, methodName, paramsComma+getArgsProto(params))
	sb.Pn("    return v.GetObject_().Go_(v.GetInterfaceName_()+\".%s\", flags, ch %s)",
		methodName, getArgsName(params))
	sb.Pn("}\n")

	// get results
	results := filterTuple(signature.Results())
	if len(results) > 0 {
		sb.Pn("func (*%s) Store%s(call *dbus.Call) (%s,err error) {", ObjectName,
			methodName, getArgsProto(results))
		sb.Pn("    err = call.Store(%s)", getArgsName(results))
		sb.Pn("    return")
		sb.Pn("}\n")
		sb.Pn("func (v *%s) %s(flags dbus.Flags %s) (%s, err error) {",
			ObjectName, methodName, getArgsProto(params),
			getArgsProto(results))
		sb.Pn("    return v.Store%s(", method.Name())
		sb.Pn("<-v.Go%s(flags, make(chan *dbus.Call, 1)%s%s).Done)",
			methodName, paramsComma, getArgsName(params))
		sb.Pn("}\n")
	} else {
		sb.Pn("func (v *%s) %s(flags dbus.Flags%s%s) error {",
			ObjectName, methodName, paramsComma, getArgsProto(params))
		sb.Pn("    return (<-v.Go%s(flags, make(chan *dbus.Call, 1)%s%s).Done).Err",
			methodName, paramsComma, getArgsName(params))
		sb.Pn("}\n")
	}
}

func writeProperty(sb *SourceBody, ObjectName string, prop *types.Var) {
	sb.Pn("// property %s %s\n", prop.Name(), prop.Type().String())

	propType := getPropType(prop)
	if propType != "" {
		sb.Pn("func (v *%s) %s() %s {", ObjectName, prop.Name(), propType)
		sb.Pn("    return %s{", propType)
		sb.Pn("        Impl: v,")
		sb.Pn("        Name: %q,", prop.Name())
		sb.Pn("    }")
		sb.Pn("}\n")
	} else {
		sb.Pn("func (v *%s) %s() %s {", ObjectName, prop.Name(), prop.Type().String())
		sb.Pn("    return %s{", prop.Type().String())
		sb.Pn("        Impl: v,")
		sb.Pn("    }")
		sb.Pn("}\n")

		sb.Pn("type %s struct {", prop.Type().String())
		sb.Pn("Impl proxy.Implementer")
		sb.Pn("}\n")

		if strings.Contains(prop.Name(), "read") {
			writePropGet(sb, prop, "p.Name")
		}
		if strings.Contains(prop.Name(), "write") {
			writePropSet(sb, prop, "p.Name")
		}
		writePropConnectChanged(sb, prop, "p.Name")
	}
}

func writePropGet(sb *SourceBody, prop *types.Var, propName string) {
	sb.Pn("func (p %s) Get(flags dbus.Flags) (value %s, err error) {",
		prop.Name(), prop.Type().String())
	sb.Pn("err = p.Impl.GetObject_().GetProperty_(flags, p.Impl.GetInterfaceName_(),")
	sb.Pn("%s, &value)", propName)
	sb.Pn("    return")
	sb.Pn("}\n")
}

func writePropSet(sb *SourceBody, prop *types.Var, propName string) {
	sb.Pn("func (p %s) Set(flags dbus.Flags, value %s) error {",
		prop.Name(), prop.Type().String())
	sb.Pn("return p.Impl.GetObject_().SetProperty_(flags,"+
		" p.Impl.GetInterfaceName_(), %s, value)", propName)
	sb.Pn("}\n")
}

func writePropConnectChanged(sb *SourceBody, prop *types.Var, propName string) {
	sb.Pn("func (p %s) ConnectChanged(cb func(hasValue bool, value %s)) error {",
		prop.Name(), prop.Type().String())
	sb.Pn("if cb == nil {")
	sb.Pn("    return errors.New(\"nil callback\")")
	sb.Pn("}")
	sb.Pn("cb0 := func(hasValue bool, value interface{}) {")

	sb.Pn("if hasValue {")
	sb.Pn("    var v %s", prop.Type().String())
	sb.Pn("    err := dbus.Store([]interface{}{value}, &v)")
	sb.Pn("    if err != nil {")
	sb.Pn("        return")
	sb.Pn("    }")
	sb.Pn("    cb(true, v)")
	sb.Pn("} else {")
	sb.Pn("    cb(false, %s)", "nil")
	sb.Pn("}")

	sb.Pn("}") // end cb0

	sb.Pn("return p.Impl.GetObject_().ConnectPropertyChanged_(p.Impl.GetInterfaceName_(),")
	sb.Pn("%s, cb0)", propName)
	sb.Pn("}\n")
}

func writeSignal(sb *SourceBody, ObjectName string, signal *types.Var) {
	sb.Pn("// signal %s\n", signal.Name())
	methodName := strings.Title(signal.Name())
	log.Print(methodName)

	obj, ok := signal.Type().(*types.Struct)
	if !ok {
		return
	}
	var elms []*types.Var
	for oIndex := 0; oIndex < obj.NumFields(); oIndex++ {
		pVar := obj.Field(oIndex)
		if isInvalidType(pVar) {
			continue
		}
		elms = append(elms, pVar)
	}
	log.Print(elms)
	sb.Pn("func (v *%s) Connect%s(cb func(%s)) (dbusutil.SignalHandlerId, error) {",
		ObjectName, methodName, getArgsProto(elms))
	sb.Pn("if cb == nil {")
	sb.Pn("   return 0, errors.New(\"nil callback\")")
	sb.Pn("}")
	sb.Pn("obj := v.GetObject_()")
	sb.Pn("rule := fmt.Sprintf(")
	sb.writeStr(`"type='signal',interface='%s',member='%s',path='%s',sender='%s'",` + "\n")
	sb.Pn("v.GetInterfaceName_(), %q, obj.Path_(), obj.ServiceName_())\n", signal.Name())
	sb.Pn("sigRule := &dbusutil.SignalRule{")
	sb.Pn("Path: obj.Path_(),")
	sb.Pn("Name: v.GetInterfaceName_() + \".%s\",", signal.Name())
	sb.Pn("}")
	sb.Pn("handlerFunc := func(sig *dbus.Signal) {")

	if len(elms) > 0 {
		for _, arg := range elms {
			var buf bytes.Buffer
			types.WriteType(&buf, arg.Type(), nil)
			argType := buf.String()
			sb.Pn("var %s %s", arg.Name(), argType)
		}
		sb.Pn("err := dbus.Store(sig.Body %s)", getArgsName(elms))
		sb.Pn("if err == nil {")
		sb.Pn("    cb(%s)", getArgsName(elms))
		sb.Pn("}")
	} else {
		sb.Pn("cb()")
	}
	sb.Pn("}\n") // end handlerFunc
	sb.Pn("return obj.ConnectSignal_(rule, sigRule, handlerFunc)")
	sb.Pn("}\n")
}

func isInvalidType(elem *types.Var) bool {
	var buf bytes.Buffer
	types.WriteType(&buf, elem.Type(), nil)
	if buf.String() == "*invalid type" {
		return true
	}
	return false
}

var propBaseTypeMap = []string{
	"byte",
	"bool",
	"int16",
	"uint16",
	"int32",
	"uint32",
	"int64",
	"uint64",
	"double",
	"string",
	"objectPath",
}

func getPropType(ty *types.Var) string {
	// if is bcType type
	if bcType, ok := ty.Type().(*types.Basic); ok {
		if IsExitItem(bcType.Name(), propBaseTypeMap) {
			return "proxy.Prop" + strings.Title(bcType.Name())
		}
	}
	// if is slice
	if scType, ok := ty.Type().(*types.Slice); ok {
		if IsExitItem(scType.Elem().String(), propBaseTypeMap) {
			return "proxy.Prop" + strings.Title(scType.Elem().String()) + "Array"
		}
	}
	return ""
}

// get proto
func getArgsProto(args []*types.Var) string {
	if len(args) == 0 {
		return ""
	}
	var bufProto bytes.Buffer
	for pIndex := 0; pIndex < len(args); pIndex++ {
		argName := fmt.Sprintf("arg_%d", pIndex)
		pVar := args[pIndex]
		if pVar.Name() != "" {
			argName = pVar.Name()
		}
		if isInvalidType(pVar) {
			continue
		}

		// get types value
		var buf bytes.Buffer
		types.WriteType(&buf, pVar.Type(), nil)

		// add to proto
		bufProto.WriteString(argName + " " + buf.String() + ",")
	}
	return strings.TrimRight(bufProto.String(), ",")
}

// get proto
func getArgsName(args []*types.Var) string {
	if len(args) == 0 {
		return ""
	}
	var bufProto bytes.Buffer
	for aIndex := 0; aIndex < len(args); aIndex++ {
		argName := fmt.Sprintf("arg_%d", aIndex)
		pVar := args[aIndex]
		if pVar.Name() != "" {
			argName = pVar.Name()
		}
		if isInvalidType(pVar) {
			continue
		}

		// get types value
		var buf bytes.Buffer
		types.WriteType(&buf, pVar.Type(), nil)

		// add to proto
		bufProto.WriteString(argName + ",")
	}
	return strings.TrimRight(bufProto.String(), ",")
}

// filter tuple
func filterTuple(tuple *types.Tuple) []*types.Var {
	var elms []*types.Var
	if tuple.Len() == 0 {
		return nil
	}
	// filter invalid type
	for tIndex := 0; tIndex < tuple.Len(); tIndex++ {
		elem := tuple.At(tIndex)
		if isInvalidType(elem) {
			continue
		}
		elms = append(elms, elem)
	}
	return elms
}

func IsExitItem(source interface{}, array interface{}) bool {
	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)
		for index := 0; index < s.Len(); index++ {
			if reflect.DeepEqual(source, s.Index(index).Interface()) {
				return true
			}
		}
	}
	return false
}

func GetDBusPathName(file *ast.File) []*DBusElem {
	if file == nil {
		return nil
	}
	for _, astDecl := range file.Decls {
		if funcDecl, ok := astDecl.(*ast.FuncDecl); ok {
			bodyList := funcDecl.Body.List
			var callExpr *ast.CallExpr
			for _, stmt := range bodyList {
				if expr, ok := stmt.(*ast.ExprStmt); ok {
					X := expr.X
					if X == nil {
						continue
					}
					if callExpr, ok = X.(*ast.CallExpr); !ok {
						continue
					}
				} else if assign, ok := stmt.(*ast.AssignStmt); ok {
					for _, expr := range assign.Rhs {
						if callExpr, ok = expr.(*ast.CallExpr); !ok {
							continue
						}
					}
				}
				if callExpr == nil {
					continue
				}
				Func := callExpr.Fun
				if Func == nil {
					continue
				}
				if selector, ok := Func.(*ast.SelectorExpr); ok {
					if selector.Sel.Name == "Export" {
						argsLen := len(callExpr.Args)
						if argsLen < 2 {
							return nil
						}
						busPath, busConst := GetDBusPathFromExpr(callExpr.Args[0])
						if busPath == "" && busConst == "" {
							return nil
						}
						var els []*DBusElem
						for index := 1; index < argsLen; index++ {
							busObj, info := GetObjectNameFromExpr(callExpr.Args[index])
							if busObj == "" && info == nil {
								continue
							}
							elem := &DBusElem{
								DBusPath:    busPath,
								DBusObjName: busObj,
								DBusConst:   busConst,
								DBusInfo:    info,
							}
							els = append(els, elem)
						}
						if len(els) == 0 {
							return nil
						}
						return els
					}
				}
			}
		}
	}
	return nil
}

func GetDBusObjNameFromObj(object *ast.Object, info *StructInfo) string {
	typeSpec, ok := object.Decl.(*ast.TypeSpec)
	if !ok {
		return ""
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return ""
	}
	fields := structType.Fields.List
	for _, field := range fields {
		if len(field.Names) < 1 {
			continue
		}
		name := field.Names[0]
		if name.Name == info.Elem {
			nameType := field.Type
			ident, ok := nameType.(*ast.Ident)
			if ok {
				return ident.Name
			}
			if starExp, ok := nameType.(*ast.StarExpr); ok {
				X := starExp.X
				xIdent, ok := X.(*ast.Ident)
				if ok {
					return xIdent.Name
				}
			}

			return ""
		}
	}
	return ""
}

func GetDBusPathFromObj(obj *ast.Object) string {
	valueSpec, ok := obj.Decl.(*ast.ValueSpec)
	if !ok {
		return ""
	}
	values := valueSpec.Values
	if len(values) == 0 {
		return ""
	}
	bc, ok := values[0].(*ast.BasicLit)
	if !ok {
		return ""
	}
	return bc.Value
}

func GetDBusPathFromExpr(expr ast.Expr) (string, string) {
	astIdent, ok := expr.(*ast.Ident)
	if !ok {
		return "", ""
	}
	busConst := astIdent.Name
	if astIdent.Obj == nil {
		return "", busConst
	}
	busPath := GetDBusPathFromObj(astIdent.Obj)
	return busPath, busConst
}

func GetObjectNameFromExpr(expr ast.Expr) (string, *StructInfo) {
	unaryExpr, ok := expr.(*ast.UnaryExpr)
	if ok {
		X := unaryExpr.X
		if X == nil {
			return "", nil
		}
		secExpr, ok := X.(*ast.SelectorExpr)
		if !ok {
			return "", nil
		}
		xIdent, ok := secExpr.X.(*ast.Ident)
		if !ok {
			return "", nil
		}
		astField, ok := xIdent.Obj.Decl.(*ast.Field)
		if !ok {
			return "", nil
		}
		starExpr, ok := astField.Type.(*ast.StarExpr)
		if !ok {
			return "", nil
		}
		xName, ok := starExpr.X.(*ast.Ident)
		if !ok {
			return "", nil
		}
		sel := secExpr.Sel
		if sel == nil {
			return "", nil
		}
		info := &StructInfo{
			StructName: xName.Name,
			Elem:       sel.Name,
		}
		return "", info
	}
	secExpr, ok := expr.(*ast.SelectorExpr)
	if ok {
		xIdent, ok := secExpr.X.(*ast.Ident)
		if !ok {
			return "", nil
		}
		astField, ok := xIdent.Obj.Decl.(*ast.Field)
		if !ok {
			return "", nil
		}
		starExpr, ok := astField.Type.(*ast.StarExpr)
		if !ok {
			return "", nil
		}
		xName, ok := starExpr.X.(*ast.Ident)
		if !ok {
			return "", nil
		}
		sel := secExpr.Sel
		if sel == nil {
			return "", nil
		}
		info := &StructInfo{
			StructName: xName.Name,
			Elem:       sel.Name,
		}
		return "", info
	}
	if ident, ok := expr.(*ast.Ident); ok {
		obj := ident.Obj
		if obj == nil {
			return "", nil
		}
		assign, ok := obj.Decl.(*ast.AssignStmt)
		if !ok {
			return "", nil
		}
		busObj := GetObjReturn(assign)
		return busObj, nil
	}

	return "", nil
}

func GetObjReturn(assign *ast.AssignStmt) string {
	rhs := assign.Rhs
	rh := rhs[0]
	callExpr, ok := rh.(*ast.CallExpr)
	if !ok {
		return ""
	}
	Func := callExpr.Fun
	if Func == nil {
		return ""
	}
	astFunc, ok := Func.(*ast.Ident)
	if !ok {
		return ""
	}
	obj := astFunc.Obj
	if obj == nil {
		return ""
	}
	funcDecl, ok := obj.Decl.(*ast.FuncDecl)
	if !ok {
		return ""
	}
	rtList := funcDecl.Type.Results.List
	if len(rtList) < 0 {
		return ""
	}
	field, ok := rtList[0].Type.(*ast.StarExpr)
	if !ok {
		return ""
	}
	X, ok := field.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return X.Name
}
