package helpers

import (
	"net/url"
	"os"
	"strings"
	"errors"
)

func EnforceHTTP(url string) string {
	if url[:4] != "http" {
		return "http://" + url
	}
	return url
}

func RemoveDomainError(url string) bool {
	if url == os.Getenv("DOMAIN") {
		return false
	}
	newURL := strings.Replace(url, "http://", "", 1)
	newURL = strings.Replace(newURL, "https://", "", 1)
	newURL = strings.Replace(newURL, "www.", "", 1)
	newURL = strings.Split(newURL, "/")[0]

	return newURL != os.Getenv("DOMAIN")
}

func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid URL")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("URL must use http or https")
	}

	if u.Hostname() == "" {
		return errors.New("URL must include a hostname")
	}

	return nil
}