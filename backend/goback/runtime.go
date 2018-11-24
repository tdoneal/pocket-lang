package goback

import (
	"math"
	"reflect"
	"strconv"
)

const (
	DTY_INT    = 1
	DTY_FLOAT  = 2
	DTY_STRING = 3
)

type duck interface{}

// TODO: build symmetrical and asymmetrical duck operation luts
var __pk_dot_add = map[uint32]func(a duck, b duck) duck{
	(1 << DTY_INT) | (1 << DTY_INT):       func(a duck, b duck) duck { return a.(int) + b.(int) },
	(1 << DTY_FLOAT) | (1 << DTY_FLOAT):   func(a duck, b duck) duck { return a.(float64) + b.(float64) },
	(1 << DTY_INT) | (1 << DTY_FLOAT):     func(a duck, b duck) duck { return float64(a.(int)) + b.(float64) },
	(1 << DTY_STRING) | (1 << DTY_STRING): func(a duck, b duck) duck { return a.(string) + b.(string) },
	(1 << DTY_INT) | (1 << DTY_STRING):    func(a duck, b duck) duck { return strconv.Itoa(int(a.(int))) + b.(string) },
	(1 << DTY_FLOAT) | (1 << DTY_STRING):  func(a duck, b duck) duck { return __pk_duck_ftoa(a.(float64)) + b.(string) },
}

var __pk_dot_asym_add = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:    func(a duck, b duck) duck { return a.(int) + b.(int) },
		DTY_FLOAT:  func(a duck, b duck) duck { return float64(a.(int)) + b.(float64) },
		DTY_STRING: func(a duck, b duck) duck { return strconv.Itoa(int(a.(int))) + b.(string) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:    func(a duck, b duck) duck { return a.(float64) + float64(b.(int)) },
		DTY_FLOAT:  func(a duck, b duck) duck { return a.(float64) + b.(float64) },
		DTY_STRING: func(a duck, b duck) duck { return __pk_duck_ftoa(a.(float64)) + b.(string) },
	},
	DTY_STRING: map[uint32]func(duck, duck) duck{
		DTY_INT:    func(a duck, b duck) duck { return a.(string) + strconv.Itoa(int(b.(int))) },
		DTY_FLOAT:  func(a duck, b duck) duck { return a.(string) + __pk_duck_ftoa(b.(float64)) },
		DTY_STRING: func(a duck, b duck) duck { return a.(string) + b.(string) },
	},
}

var __pk_dot_asym_sub = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) - b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return float64(a.(int)) - b.(float64) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(float64) - float64(b.(int)) },
		DTY_FLOAT: func(a duck, b duck) duck { return a.(float64) - b.(float64) },
	},
}

var __pk_dot_asym_mul = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) * b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return float64(a.(int)) * b.(float64) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(float64) * float64(b.(int)) },
		DTY_FLOAT: func(a duck, b duck) duck { return a.(float64) * b.(float64) },
	},
}

var __pk_dot_asym_div = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) / b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return float64(a.(int)) / b.(float64) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(float64) / float64(b.(int)) },
		DTY_FLOAT: func(a duck, b duck) duck { return a.(float64) / b.(float64) },
	},
}

var __pk_dot_asym_mod = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) % b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return math.Mod(float64(a.(int)), b.(float64)) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return math.Mod(a.(float64), float64(b.(int))) },
		DTY_FLOAT: func(a duck, b duck) duck { return math.Mod(a.(float64), b.(float64)) },
	},
}

var __pk_dot_asym_gt = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) > b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return float64(a.(int)) > b.(float64) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(float64) > float64(b.(int)) },
		DTY_FLOAT: func(a duck, b duck) duck { return a.(float64) > b.(float64) },
	},
}

var __pk_dot_asym_gteq = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) >= b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return float64(a.(int)) >= b.(float64) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(float64) >= float64(b.(int)) },
		DTY_FLOAT: func(a duck, b duck) duck { return a.(float64) >= b.(float64) },
	},
}

var __pk_dot_asym_lt = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) < b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return float64(a.(int)) < b.(float64) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(float64) < float64(b.(int)) },
		DTY_FLOAT: func(a duck, b duck) duck { return a.(float64) < b.(float64) },
	},
}

var __pk_dot_asym_lteq = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) <= b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return float64(a.(int)) <= b.(float64) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(float64) <= float64(b.(int)) },
		DTY_FLOAT: func(a duck, b duck) duck { return a.(float64) <= b.(float64) },
	},
}

var __pk_dot_asym_defeq = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(int) == b.(int) },
		DTY_FLOAT: func(a duck, b duck) duck { return float64(a.(int)) == b.(float64) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:   func(a duck, b duck) duck { return a.(float64) == float64(b.(int)) },
		DTY_FLOAT: func(a duck, b duck) duck { return a.(float64) == b.(float64) },
	},
	DTY_STRING: map[uint32]func(duck, duck) duck{
		DTY_STRING: func(a duck, b duck) duck { return a.(string) == b.(string) },
	},
}

func goty_to_dty(k reflect.Kind) uint32 {
	if k == reflect.Int {
		return DTY_INT
	} else if k == reflect.Float64 {
		return DTY_FLOAT
	} else if k == reflect.String {
		return DTY_STRING
	} else {
		panic("unsupported go kind: " + k.String())
	}
}

// overall goal: want function of two types that returns a function that operates on those types (in the proper order)
// step1: define a canonical order of types
// step2: for input types, sort by canonical order such that t0 <= t1, return flag on whether flipped
// step3: lookup the symmetrical bitmask to get the actual work function, which is always defined wrt the canonical order
// step4: call the work function with the proper arg order

func P__duck_primbinop(a duck, b duck, oplut map[uint32]map[uint32]func(duck, duck) duck) duck {
	// look up the types of the inputs as DTY_ integers
	atk := goty_to_dty(reflect.TypeOf(a).Kind())
	btk := goty_to_dty(reflect.TypeOf(b).Kind())

	// lookup operation
	// generate lookup key
	if op1, ok1 := oplut[atk]; ok1 {
		if finalOp, ok2 := op1[btk]; ok2 {
			return finalOp(a, b)
		}
	}

	panic("unsupported type")
}

func P__duck_add(a duck, b duck) duck {
	return P__duck_primbinop(a, b, __pk_dot_asym_add)
}
func P__duck_sub(a duck, b duck) duck {
	return P__duck_primbinop(a, b, __pk_dot_asym_sub)

}
func P__duck_mul(a duck, b duck) duck {
	return P__duck_primbinop(a, b, __pk_dot_asym_mul)
}
func P__duck_div(a duck, b duck) duck {
	return P__duck_primbinop(a, b, __pk_dot_asym_div)
}
func P__duck_mod(a duck, b duck) duck {
	return P__duck_primbinop(a, b, __pk_dot_asym_mod)
}
func P__duck_gt(a duck, b duck) bool {
	return P__duck_primbinop(a, b, __pk_dot_asym_gt).(bool)
}
func P__duck_gteq(a duck, b duck) bool {
	return P__duck_primbinop(a, b, __pk_dot_asym_gteq).(bool)
}
func P__duck_lt(a duck, b duck) bool {
	return P__duck_primbinop(a, b, __pk_dot_asym_lt).(bool)
}
func P__duck_lteq(a duck, b duck) bool {
	return P__duck_primbinop(a, b, __pk_dot_asym_lteq).(bool)
}
func P__duck_defeq(a duck, b duck) bool {
	return P__duck_primbinop(a, b, __pk_dot_asym_defeq).(bool)
}
func __pk_duck_ftoa(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func P__duck_field_read(obj duck, field string) duck {
	val := reflect.Indirect(reflect.ValueOf(obj))
	fieldVal := val.FieldByName(field)
	return fieldVal.Interface()
}

func P__duck_field_write(obj duck, name string, value duck) {
	structValue := reflect.Indirect(reflect.ValueOf(obj))
	structFieldValue := structValue.FieldByName(name)
	val := reflect.ValueOf(value)
	structFieldValue.Set(val)
}

func P__duck_method_call(obj duck, name string, arg duck) duck {
	objval := reflect.ValueOf(obj)
	var argvals []reflect.Value
	if arg == nil {
		argvals = []reflect.Value{}
	} else {
		argvals = []reflect.Value{reflect.Indirect(reflect.ValueOf(arg))}
	}
	method := objval.MethodByName(name)
	if !method.IsValid() {
		panic("unknown method " + name)
	}
	// outval := objval.MethodByName(name).Call([]reflect.Value{argval})[0]
	outvals := method.Call(argvals)
	if len(outvals) == 0 {
		return nil
	}
	return outvals[0].Interface()
}
