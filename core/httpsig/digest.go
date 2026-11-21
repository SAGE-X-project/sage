// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package httpsig

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

// CheckDigest is step 7 of 03 §5 (Content digest, E03-13): Content-Digest
// must hold exactly one member, sha-256, a byte sequence of 32 bytes equal
// to SHA-256 of body, the message content as transferred. Any other member,
// a second member, a parameter, another type or length, and a value that
// does not parse are rejected with reason digest. A body over
// lim.MaxDigestBodyBytes (at most 16 MiB) is rejected as too large. A
// message without Content-Digest passes only with an empty body; a request
// or response with a body must also cover content-digest (step 2), so its
// field is always signed.
func CheckDigest(h http.Header, body []byte, lim core.Limits) error {
	if err := lim.Validate(); err != nil {
		return err
	}
	if int64(len(body)) > lim.MaxDigestBodyBytes {
		return reject(obs.ReasonTooLarge, "body exceeds the Content-Digest limit")
	}
	v, err := oneLine(h, FieldContentDigest, false)
	if err != nil {
		return err
	}
	if v == "" && !hasField(h, FieldContentDigest) {
		if len(body) > 0 {
			return reject(obs.ReasonDigest, "body without Content-Digest")
		}
		return nil
	}
	members, err := parseDictionary(v)
	if err != nil || len(members) != 1 {
		return reject(obs.ReasonDigest, "Content-Digest is not one dictionary member")
	}
	m := members[0]
	if m.key != "sha-256" || m.inner || len(m.item.params) != 0 || m.item.value.kind != kindBytes ||
		len(m.item.value.text) != sha256.Size {
		return reject(obs.ReasonDigest, "Content-Digest is not one 32-byte sha-256 member")
	}
	sum := sha256.Sum256(body)
	if subtle.ConstantTimeCompare(sum[:], []byte(m.item.value.text)) != 1 {
		return reject(obs.ReasonDigest, "Content-Digest does not match the body")
	}
	return nil
}
