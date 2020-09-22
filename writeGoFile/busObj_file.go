package writeGoFile

import (
	"go/types"
	"log"
)

type DBusObject struct {
	// identify
	TypeName   string
	ObjectName string

	// export interface
	serviceName   string
	busPath       string
	interfaceName string

	// Properties
	properties []*types.Var

	// method
	methods []*types.Func

	//signal
	signals []*types.Var
}

func NewDBusObject() *DBusObject {
	budObject := &DBusObject{
		TypeName:      "",
		ObjectName:    "",
		serviceName:   "",
		busPath:       "",
		interfaceName: "",
		properties:    []*types.Var{},
		methods:       []*types.Func{},
		signals:       []*types.Var{},
	}
	return budObject
}

func (o *DBusObject) SetPackageName(packageName string) {
	o.ObjectName = packageName
}

func (o *DBusObject) SetServiceName(service string) {
	o.serviceName = service
}

func (o *DBusObject) SetInterfaceName(bus string) {
	o.interfaceName = bus
}

func (o *DBusObject) SetDBusPath(busPath string) {
	o.busPath = busPath
}

// set type
func (o *DBusObject) SetTypesNamed(named *types.Named) {
	// add properties and signals
	fields, ok := named.Underlying().(*types.Struct)
	if ok {
		for tIndex := 0; tIndex < fields.NumFields(); tIndex++ {
			field := fields.Field(tIndex)
			// judge type
			if IsProperty(field) {
				// if var type is property, add to property
				o.AddProperty(field)
			} else if IsSignals(field) {
				// if var type is signals
				pointer, ok := field.Type().(*types.Pointer)
				if !ok {
					continue
				}
				// check if is struct
				signals, ok := pointer.Elem().(*types.Struct)
				if !ok {
					continue
				}
				// add signal
				for sIndex := 0; sIndex < signals.NumFields(); sIndex++ {
					o.AddSignal(signals.Field(sIndex))
				}
			}
		}
	}

	// add methods
	for mIndex := 0; mIndex < named.NumMethods(); mIndex++ {
		method := named.Method(mIndex)
		if IsMethod(method) {
			o.AddMethod(method)
		} else if IsInterface(method) {
			o.interfaceName = ""
		}
	}
	log.Print("end")
}

func (o *DBusObject) SetMethods(methods []*types.Func) {
	o.methods = methods
}

func (o *DBusObject) AddProperty(property *types.Var) {
	o.properties = append(o.properties, property)
}

func (o *DBusObject) GetProperties() []*types.Var {
	return o.properties
}

func (o *DBusObject) AddSignal(signal *types.Var) {
	o.signals = append(o.signals, signal)
}

func (o *DBusObject) GetSignals() []*types.Var {
	return o.signals
}

func (o *DBusObject) AddMethod(method *types.Func) {
	o.methods = append(o.methods, method)
}
