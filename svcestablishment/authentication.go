package svcestablishment

import (
	"io/ioutil"
	"log"
)

const (
	privateKeyPath = "./config/app"
	publicKeyPath  = "./config/app.pub"
)

var verifKey, signKey []byte

// InitKey Initialize the key used in authentication thanks to a rsa key saved
func InitKey() {
	var err error

	signKey, err = ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Fatal("Error reading private key at : " + privateKeyPath)
	}

	verifKey, err = ioutil.ReadFile(publicKeyPath)
	if err != nil {
		log.Fatal("Error reading public key : " + publicKeyPath)
	}
}

/* MW Authentication interface */
func ServiceAuthenticationMiddleware() Middleware {
	return func(next IService) IService {
		return serviceAuthenticationMiddleware{
			next: next,
		}
	}
}

type serviceAuthenticationMiddleware struct {
	next IService
}
