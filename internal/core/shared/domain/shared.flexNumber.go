package shared_domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// FlexNumber es un número de un documento legacy, que puede estar guardado como double, entero o string.
type FlexNumber float64

// UnmarshalBSONValue acepta double, int32, int64 y string numérico. null, undefined y el string vacío
// quedan en 0; un string que no es número, NaN o infinito es un error.
func (n *FlexNumber) UnmarshalBSONValue(t byte, data []byte) error {
	rv := bson.RawValue{Type: bson.Type(t), Value: data}

	switch rv.Type {
	case bson.TypeDouble:
		if v, ok := rv.DoubleOK(); ok {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("FlexNumber: %v no es un número finito", v)
			}
			*n = FlexNumber(v)
			return nil
		}
	case bson.TypeInt32:
		if v, ok := rv.Int32OK(); ok {
			*n = FlexNumber(v)
			return nil
		}
	case bson.TypeInt64:
		if v, ok := rv.Int64OK(); ok {
			*n = FlexNumber(v)
			return nil
		}
	case bson.TypeString:
		if s, ok := rv.StringValueOK(); ok {
			return n.parse(s)
		}
	case bson.TypeNull, bson.TypeUndefined:
		*n = 0
		return nil
	default:
		return fmt.Errorf("FlexNumber: tipo BSON %s no soportado", rv.Type)
	}
	return fmt.Errorf("FlexNumber: valor BSON %s malformado", rv.Type)
}

// parse lee un string numérico; el string vacío es 0.
func (n *FlexNumber) parse(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		*n = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("FlexNumber: %q no es un número", s)
	}
	*n = FlexNumber(v)
	return nil
}

// Int devuelve la parte entera, truncada hacia cero como parseInt.
func (n FlexNumber) Int() int {
	return int(math.Trunc(float64(n)))
}
