package shared_domain

const accountMask = "****************"

// MaskAccountNumber oculta el número de cuenta y deja los últimos 4 caracteres.
func MaskAccountNumber(n string) string {
	r := []rune(n)
	if len(r) > 4 {
		r = r[len(r)-4:]
	}
	return accountMask + string(r)
}
