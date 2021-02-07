package labdh

import (
	"log"
	"strings"

	"github.com/nilsocket/svach"
)

// extractName from URL
func extractName(URL string) string {
	// anything after last `/` is considered as name
	i := strings.LastIndexByte(URL, '/')
	if i == -1 {
		log.Println("Unable to extract name from URL", URL)
	}

	return svach.Name(URL[i+1:])
}
