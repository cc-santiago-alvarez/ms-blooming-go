package shared_domain

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AgencyIDs son las agencias de un documento. En los documentos legacy agencyId puede ser un string o un
// arreglo; siempre se lee y se persiste como arreglo.
type AgencyIDs []string

// UnmarshalBSONValue acepta un string, un arreglo de strings, null y undefined. El string vacío, null y
// undefined quedan en [].
func (a *AgencyIDs) UnmarshalBSONValue(t byte, data []byte) error {
	rv := bson.RawValue{Type: bson.Type(t), Value: data}

	switch rv.Type {
	case bson.TypeString:
		if s, ok := rv.StringValueOK(); ok {
			if s == "" {
				*a = AgencyIDs{}
			} else {
				*a = AgencyIDs{s}
			}
			return nil
		}
	case bson.TypeArray:
		if arr, ok := rv.ArrayOK(); ok {
			return a.fromArray(arr)
		}
	case bson.TypeNull, bson.TypeUndefined:
		*a = AgencyIDs{}
		return nil
	default:
		return fmt.Errorf("AgencyIDs: tipo BSON %s no soportado", rv.Type)
	}
	return fmt.Errorf("AgencyIDs: valor BSON %s malformado", rv.Type)
}

// fromArray lee un arreglo BSON en el que todos los elementos son strings.
func (a *AgencyIDs) fromArray(arr bson.RawArray) error {
	values, err := arr.Values()
	if err != nil {
		return fmt.Errorf("AgencyIDs: arreglo malformado: %w", err)
	}
	ids := make(AgencyIDs, 0, len(values))
	for _, v := range values {
		s, ok := v.StringValueOK()
		if !ok {
			return fmt.Errorf("AgencyIDs: elemento BSON %s no es string", v.Type)
		}
		ids = append(ids, s)
	}
	*a = ids
	return nil
}

// MarshalBSONValue persiste siempre un arreglo, también cuando está vacío.
func (a AgencyIDs) MarshalBSONValue() (byte, []byte, error) {
	t, data, err := bson.MarshalValue(a.values())
	return byte(t), data, err
}

// MarshalJSON escribe siempre un arreglo, nunca null.
func (a AgencyIDs) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.values())
}

// Normalize recorta cada id y quita los vacíos y los repetidos, conservando el orden.
func (a AgencyIDs) Normalize() AgencyIDs {
	out := make(AgencyIDs, 0, len(a))
	seen := make(map[string]bool, len(a))
	for _, id := range a {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// values devuelve los ids como []string, con [] en lugar de nil.
func (a AgencyIDs) values() []string {
	if a == nil {
		return []string{}
	}
	return []string(a)
}
