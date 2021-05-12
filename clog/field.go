package clog

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"time"
)

type fieldType int

const (
	unKnownType fieldType = iota
	boolType
	floatType
	intType
	int64Type
	uintType
	uint64Type
	uintptrType
	stringType
	objectType
	stringerType
)

type Field struct {
	key       string
	fieldType fieldType
	ival      int64
	str       string
	obj       interface{}
}

func (f *Field) WriteValue(b []byte) []byte {
	switch f.fieldType {
	case boolType:
		return strconv.AppendBool(b, f.ival == 1)
	case stringType:
		return append(b, f.str...)
	case intType:
		return strconv.AppendInt(b, int64(f.ival), 10)
	case int64Type:
		return strconv.AppendInt(b, f.ival, 10)
	case floatType:
		return strconv.AppendFloat(b, math.Float64frombits(uint64(f.ival)),
			'f', -1, 64)
	case uintType:
		return strconv.AppendUint(b, uint64(f.ival), 10)
	case objectType:
		return append(b, fmt.Sprintf("%+v", f.obj)...)
	case stringerType:
		return append(b, f.obj.(fmt.Stringer).String()...)
	case uintptrType:
		b = append(b, "0x"...)
		return strconv.AppendUint(b, uint64(f.ival), 16)
	default:

	}
	return nil
}

//Base64 转换
func Base64(key string, val []byte) Field {
	return String(key, base64.StdEncoding.EncodeToString(val))
}

//Bool
//lazily
func Bool(key string, val bool) Field {
	var ival int64
	if val {
		ival = 1
	}
	return Field{key: key, fieldType: boolType, ival: ival}
}

//
func Float64(key string, val float64) Field {
	return Field{key: key, fieldType: floatType, ival: int64(math.Float64bits(val))}
}

func Int(key string, val int) Field {
	return Field{key: key, fieldType: intType, ival: int64(val)}
}

func Uint(key string, val uint) Field {
	return Field{key: key, fieldType: uintType, ival: int64(val)}
}

func Int64(key string, val int64) Field {
	return Field{key: key, fieldType: int64Type, ival: val}
}

func Uint64(key string, val uint64) Field {
	return Field{key: key, fieldType: uintType, ival: int64(val)}
}

func Uintptr(key string, val uintptr) Field {
	return Field{key: key, fieldType: uintptrType, ival: int64(val)}
}

func String(key string, val string) Field {
	return Field{key: key, fieldType: stringType, str: val}
}

func Stringer(key string, val fmt.Stringer) Field {
	return Field{key: key, fieldType: stringerType, obj: val}
}

func Duration(key string, val time.Duration) Field {
	return Int64(key, int64(val))
}

func Object(key string, val interface{}) Field {
	return Field{key: key, fieldType: objectType, obj: val}
}
