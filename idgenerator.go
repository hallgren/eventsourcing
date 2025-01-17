package eventsourcing

import (
	"crypto/rand"
)

// idFunc is a global function that generates aggregate id's.
// It could be changed from the outside via the SetIDFunc function.
var idFunc = randSeq

// SetIDFunc is used to change how aggregate ID's are generated
// default is a random string
func SetIDFunc(f func() string) {
	idFunc = f
}

func randSeq() string {
	id, err := generateRandomString(20)
	if err != nil {
		return ""
	}
	return id
}

func generateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	bytes, err := generateRandomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}
