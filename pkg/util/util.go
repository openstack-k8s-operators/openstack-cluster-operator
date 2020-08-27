package util

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
)

// CalculateHash computes MD5 sum of the JSONfied object passed as obj.
func CalculateHash(obj interface{}) (string, error) {
	configStr, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	configSum := md5.Sum(configStr)
	return fmt.Sprintf("%x", configSum), nil
}
