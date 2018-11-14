package net

import (
	"crypto/tls"
	"encoding/json"

	"github.com/stuphlabs/pullcord/config"
)

func init() {
	e := config.RegisterResourceType(
		"pem",
		func() json.Unmarshaler {
			return new(PemConfig)
		},
	)

	if e != nil {
		panic(e)
	}
}

type PemConfig struct {
	Cert []byte
	Key  []byte
}

func (p *PemConfig) UnmarshalJSON(d []byte) error {
	var t struct {
		Cert string
		Key  string
	}

	if e := json.Unmarshal(d, &t); e != nil {
		return e
	}

	p.Cert = []byte(t.Cert)
	p.Key = []byte(t.Key)

	return nil
}

func (p *PemConfig) GetCertificate(
	*tls.ClientHelloInfo,
) (*tls.Certificate, error) {
	c, e := tls.X509KeyPair(p.Cert, p.Key)
	return &c, e
}