/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package fake

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
)

func GenerateIIDByte(uuid string, kid string, alg jwa.SignatureAlgorithm, privKey interface{}) ([]byte, error) {
	payload := map[string]string{
		"instanceID": uuid,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	h := &jws.StandardHeaders{
		JWStyp:   "JOSE+JSON",
		JWSkeyID: kid,
	}
	s, err := jws.Sign(b, alg, privKey, jws.WithHeaders(h))
	if err != nil {
		return nil, err
	}

	resp := map[string]string{
		"data": string(s),
	}

	return json.Marshal(resp)
}

func GenerateIIDByteWithInvalidSignature(uuid string, kid string, alg jwa.SignatureAlgorithm, privKey interface{}) ([]byte, error) {
	payload := map[string]string{
		"instanceID": uuid,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	h := &jws.StandardHeaders{
		JWStyp:   "JOSE+JSON",
		JWSkeyID: kid,
	}
	s, err := jws.Sign(b, alg, privKey, jws.WithHeaders(h))
	if err != nil {
		return nil, err
	}

	jwsStr := string(s)
	jwsSlc := strings.Split(jwsStr, ".")
	jwsSlc[2] = base64.RawURLEncoding.EncodeToString([]byte("fake signature"))
	jwsStr = strings.Join(jwsSlc, ".")

	resp := map[string]string{
		"data": jwsStr,
	}

	return json.Marshal(resp)
}

func GenerateIIDKeysByte(kid string, alg jwa.SignatureAlgorithm, pubKey interface{}) ([]byte, error) {
	var keys []jwk.Key

	key, err := jwk.New(pubKey)
	if err != nil {
		return nil, err
	}
	if err := key.Set(jwk.KeyIDKey, kid); err != nil {
		return nil, err
	}
	if err := key.Set(jwk.KeyUsageKey, string(jwk.ForSignature)); err != nil {
		return nil, err
	}
	if err := key.Set(jwk.AlgorithmKey, alg.String()); err != nil {
		return nil, err
	}
	keys = append(keys, key)

	jwkSet := jwk.Set{
		Keys: keys,
	}

	return json.Marshal(jwkSet)
}
