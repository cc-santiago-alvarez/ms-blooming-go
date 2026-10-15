package utils_adapters

type UtilsPorts interface {
	// RandomAlnum genera un texto alfanumérico de n caracteres para token y reference; nunca para _id.
	RandomAlnum(n int) string
}
