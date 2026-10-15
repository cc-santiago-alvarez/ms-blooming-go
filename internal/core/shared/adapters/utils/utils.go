package utils_adapters

import "crypto/rand"

const alnum = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// maxUnbiased es el mayor múltiplo de len(alnum) que entra en un byte; los bytes mayores se
// descartan para que cada carácter tenga la misma probabilidad.
const maxUnbiased = 256 - 256%len(alnum)

type Utils struct{}

func NewUtils() UtilsPorts {
	return &Utils{}
}

func (u *Utils) RandomAlnum(n int) string {
	if n <= 0 {
		return ""
	}

	out := make([]byte, 0, n)
	buf := make([]byte, n+n/4+1)
	for len(out) < n {
		_, _ = rand.Read(buf)
		for _, b := range buf {
			if int(b) >= maxUnbiased {
				continue
			}
			out = append(out, alnum[int(b)%len(alnum)])
			if len(out) == n {
				break
			}
		}
	}
	return string(out)
}
