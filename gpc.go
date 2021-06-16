package gpc

import (
	"errors"
	"log"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// method structure with reflect info
type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

// service structure with name and reflect info
type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

// struct for goroutine procedure call
type GPC struct {
	serviceMap sync.Map
	gpcBase
}

// create gpc
func NewGPC(options ...GPCOption) *GPC {
	gpc := &GPC{}
	for _, option := range options {
		option(gpc)
	}
	gpc.Init()
	// 在这里给Run中调用的处理函数赋值，有点丑，目前只能这么做，算是最简单的做法了
	gpc.callMethodFunc = gpc.callMethod
	return gpc
}

func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return isExported(t.Name()) || t.PkgPath() == ""
}

func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if method.Type.NumIn() != 3 {
			if reportErr {
				log.Printf("gpc.Register: method %q has %d input parameters; need exactly three\n", mname, mtype.NumIn())
			}
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Printf("gpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
			}
			continue
		}
		// Second arg must be a pointer.
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Printf("gpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Printf("gpc.Register: reply type of method %q is not exported: %q\n", mname, replyType)
			}
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			if reportErr {
				log.Printf("gpc.Register: method %q has %d output parameters; need exactly one\n", mname, mtype.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			if reportErr {
				log.Printf("gpc.Register: return type of method %q is %q, must be error\n", mname, returnType)
			}
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

func (g *GPC) Register(rcvr interface{}) error {
	s := &service{}
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		s := "gpc.Register: no service name for type" + s.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	s.name = sname
	s.method = suitableMethods(s.typ, true)
	if len(s.method) == 0 {
		str := ""
		method := suitableMethods(reflect.PtrTo(s.typ), false)
		if len(method) != 0 {
			str = "gpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "gpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}
	if _, dup := g.serviceMap.LoadOrStore(sname, s); dup {
		return errors.New("gpc: service already defined: " + sname)
	}
	return nil
}

func (g *GPC) getMethod(method string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(method, ".")
	if dot < 0 {
		err = errors.New("gpc: service/method request ill-formed: " + method)
		return
	}
	serviceName := method[:dot]
	methodName := method[dot+1:]

	svci, o := g.serviceMap.Load(serviceName)
	if !o {
		err = errors.New("gpc: can't find service " + method)
		return
	}

	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("gpc: can't find method " + method)
		return
	}
	return
}

func (g *GPC) callMethod(method string, param interface{}, result interface{}) error {
	service, mtype, err := g.getMethod(method)
	function := mtype.method.Func
	returnValues := function.Call([]reflect.Value{service.rcvr, reflect.ValueOf(param), reflect.ValueOf(result)})
	errInter := returnValues[0].Interface()
	if errInter != nil {
		err = errInter.(error)
	}
	return err
}
