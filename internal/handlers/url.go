package handlers

// enforceHTTP adds a scheme to short links built from the configured domain.
func enforceHTTP(url string) string {
	if url[:4] != "http" {
		return "http://" + url
	}
	return url
}
