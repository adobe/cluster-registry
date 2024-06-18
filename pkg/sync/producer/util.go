package producer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func fingerprint(obj interface{}) string {
	b, _ := json.Marshal(obj)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", b)))
	return fmt.Sprintf("%x", h.Sum(nil))
}
