package support

import (
	"flag"
	"reflect"
)

var (
	valueType = reflect.TypeFor[flag.Value]()
)

func BindSupportedValue(v any) any {
	// Bind functions will either use *V or V depending upon what
	// supports the built-in convention values or implements Value.
	// Any built-in primitive will work as is.  However, if v is actually
	// *V but V is **W and W is a Value implementation, then unwrap this
	// so we end up with *W.  For example, instead of **FileSet, just use *FileSet.
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Ptr {
		pointsToValue := val.Elem().Type()
		if pointsToValue.Implements(valueType) {
			return reflect.New(pointsToValue.Elem()).Interface()
		}
	}

	// Primitives and other values
	return v
}
