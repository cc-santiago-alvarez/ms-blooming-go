package bondBound_domain

// BondKeyCounter es el último número de clave reservado de un tipo de bono, en la colección bondKeyCounter.
type BondKeyCounter struct {
	ID  string `bson:"_id"` // bondTypeId
	Seq int    `bson:"seq"`
}
