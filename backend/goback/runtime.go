package goback

import (
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
	(1 << DTY_INT) | (1 << DTY_INT):       func(a duck, b duck) duck { return a.(int64) + b.(int64) },
	(1 << DTY_FLOAT) | (1 << DTY_FLOAT):   func(a duck, b duck) duck { return a.(float64) + b.(float64) },
	(1 << DTY_INT) | (1 << DTY_FLOAT):     func(a duck, b duck) duck { return float64(a.(int64)) + b.(float64) },
	(1 << DTY_STRING) | (1 << DTY_STRING): func(a duck, b duck) duck { return a.(string) + b.(string) },
	(1 << DTY_INT) | (1 << DTY_STRING):    func(a duck, b duck) duck { return strconv.Itoa(int(a.(int64))) + b.(string) },
	(1 << DTY_FLOAT) | (1 << DTY_STRING):  func(a duck, b duck) duck { return __pk_duck_ftoa(a.(float64)) + b.(string) },
}

var __pk_dot_asym_add = map[uint32]map[uint32]func(duck, duck) duck{
	DTY_INT: map[uint32]func(duck, duck) duck{
		DTY_INT:    func(a duck, b duck) duck { return a.(int64) + b.(int64) },
		DTY_FLOAT:  func(a duck, b duck) duck { return float64(a.(int64)) + b.(float64) },
		DTY_STRING: func(a duck, b duck) duck { return strconv.Itoa(int(a.(int64))) + b.(string) },
	},
	DTY_FLOAT: map[uint32]func(duck, duck) duck{
		DTY_INT:    func(a duck, b duck) duck { return a.(float64) + float64(b.(int64)) },
		DTY_FLOAT:  func(a duck, b duck) duck { return a.(float64) + b.(float64) },
		DTY_STRING: func(a duck, b duck) duck { return __pk_duck_ftoa(a.(float64)) + b.(string) },
	},
	DTY_STRING: map[uint32]func(duck, duck) duck{
		DTY_INT:    func(a duck, b duck) duck { return a.(string) + strconv.Itoa(int(b.(int64))) },
		DTY_FLOAT:  func(a duck, b duck) duck { return a.(string) + __pk_duck_ftoa(b.(float64)) },
		DTY_STRING: func(a duck, b duck) duck { return a.(string) + b.(string) },
	},
}

func goty_to_dty(k reflect.Kind) uint32 {
	if k == reflect.Int64 {
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

func Pduck_add(a interface{}, b interface{}) interface{} {
	// look up the types of the inputs as DTY_ integers
	atk := goty_to_dty(reflect.TypeOf(a).Kind())
	btk := goty_to_dty(reflect.TypeOf(b).Kind())

	// lookup operation
	// generate lookup key
	if op1, ok1 := __pk_dot_asym_add[atk]; ok1 {
		if finalOp, ok2 := op1[btk]; ok2 {
			return finalOp(a, b)
		}
	}

	panic("unsupported type")
}
func __pk_duck_ftoa(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}
