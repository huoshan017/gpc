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

// 方法反射信息结构
type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

// gpc服务
type service struct {
	name   string                 // 服务名
	rcvr   reflect.Value          // 服务方法的反射值
	typ    reflect.Type           // 接受这的反射类型
	method map[string]*methodType // 已注册的方法
}

// gpc结构
type GPC struct {
	serviceMap sync.Map
	gpcBase
}

// 创建一个gpc
func NewGPC(options ...GPCOption) *GPC {
	gpc := &GPC{}
	for _, option := range options {
		option(gpc.options)
	}
	gpc.init(gpc.callMethod)
	return gpc
}

// 注册一个gpc服务
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

// Run中调用的处理函数，因为go无法支持在一个类型中的方法中调用接口达到虚函数的效果
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

// 内部方法，是否已导出
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// 内部方法，是否已导出或内建类型
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return isExported(t.Name()) || t.PkgPath() == ""
}

// 内部方法，通过反射信息搜集服务的所有方法
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// 方法必须已导出
		if method.PkgPath != "" {
			continue
		}
		// 方法需要三个传入参数： 接收者，参数，回复
		if method.Type.NumIn() != 3 {
			if reportErr {
				log.Printf("gpc.Register: method %q has %d input parameters; need exactly three\n", mname, mtype.NumIn())
			}
			continue
		}
		// 第一个参数类型不能是指针
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Printf("gpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
			}
			continue
		}
		// 第二个参数必须是指针
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Printf("gpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
			}
			continue
		}
		// 回复参数类型必须已导出
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Printf("gpc.Register: reply type of method %q is not exported: %q\n", mname, replyType)
			}
			continue
		}
		// 方法必须有一个传出参数
		if mtype.NumOut() != 1 {
			if reportErr {
				log.Printf("gpc.Register: method %q has %d output parameters; need exactly one\n", mname, mtype.NumOut())
			}
			continue
		}
		// 方法的返回类型必须是error
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

// 内部方法，从方法名获得服务和反射信息
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
