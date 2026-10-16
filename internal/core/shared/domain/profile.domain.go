package shared_domain

// PublicProfile es la parte de profile que se puede mostrar; nunca incluye password ni permisos.
// Un campo que falta en el documento queda vacío.
type PublicProfile struct {
	ID        string `json:"_id" bson:"_id"`
	PublicID  string `json:"id" bson:"id"`
	FirstName string `json:"first_name" bson:"first_name"`
	LastName  string `json:"last_name" bson:"last_name"`
	Email     string `json:"email" bson:"email"`
	AvatarURL string `json:"avatar_url" bson:"avatar_url"`
	AgencyID  string `json:"agencyId" bson:"agencyId"`
}
