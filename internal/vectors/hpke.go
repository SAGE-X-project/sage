package vectors

import (
	"bytes"
	"crypto/ecdh"
	"crypto/sha256"
	"errors"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
)

const (
	vecCtxID   = "ctx-7d2f0c1e-4b8a-4f5e-9c3d-1a2b3c4d5e6f"
	vecInitDID = vecDIDA
	vecRespDID = vecDIDB
	vecKID     = "kid-3f6a9c12-8b7e-4d21-a5c4-0e1f2a3b4c5d"
	vecHPKENon = "9c1d2e3f-4a5b-6c7d-8e9f-0a1b2c3d4e5f"
)

func x25519FromLabel(label string) (*ecdh.PrivateKey, error) {
	return ecdh.X25519().NewPrivateKey(seed32(label))
}

func hpkeSuite() Suite {
	return Suite{
		Name: "hpke",
		Description: "HPKE profile (RFC 9180 base mode, X25519-HKDF-SHA256, HKDF-SHA256, ChaCha20-Poly1305, export only): " +
			"info and export-context strings, the E2E secret combiner, traffic-key derivation and the acknowledgement tag. " +
			"HPKE encapsulation itself is randomised and is covered by the session vectors through the exporter secret.",
		Cases: []Case{
			{
				Name: "info-and-export-context", Mode: ModeDeterministic,
				Description: "DefaultInfo(ctx, initDID, respDID) and DefaultExportContext(ctx) byte strings.",
				Input:       map[string]any{"context_id": vecCtxID, "init_did": vecInitDID, "resp_did": vecRespDID},
				Produce: func(in map[string]any) (map[string]any, error) {
					ctx, _ := str(in, "context_id")
					a, _ := str(in, "init_did")
					b, _ := str(in, "resp_did")
					info := hpke.DefaultInfo(ctx, a, b)
					ectx := hpke.DefaultExportContext(ctx)
					ih := sha256.Sum256(info)
					eh := sha256.Sum256(ectx)
					return map[string]any{
						"info": string(info), "info_hex": hx(info), "info_sha256": hx(ih[:]),
						"export_context": string(ectx), "export_context_hex": hx(ectx), "export_context_sha256": hx(eh[:]),
					}, nil
				},
			},
			{
				Name: "x25519-e2e-shared-secret", Mode: ModeDeterministic,
				Description: "Raw X25519 Diffie-Hellman between the two ephemeral keys (ssE2E). Both private keys are derived from their labels.",
				Input:       map[string]any{"init_label": labelX25519A, "resp_label": labelX25519B},
				Produce: func(in map[string]any) (map[string]any, error) {
					a, err := x25519FromLabel(in["init_label"].(string))
					if err != nil {
						return nil, err
					}
					b, err := x25519FromLabel(in["resp_label"].(string))
					if err != nil {
						return nil, err
					}
					ss, err := a.ECDH(b.PublicKey())
					if err != nil {
						return nil, err
					}
					return map[string]any{
						"init_private": hx(a.Bytes()), "init_public": hx(a.PublicKey().Bytes()),
						"resp_private": hx(b.Bytes()), "resp_public": hx(b.PublicKey().Bytes()),
						"shared_secret": hx(ss),
					}, nil
				},
			},
			{
				Name: "combine-secrets", Mode: ModeDeterministic,
				Description: "seed = HKDF-Expand(HKDF-Extract(SHA-256, ikm = exporterHPKE || ssE2E, salt = exportCtx), \"SAGE-HPKE+E2E-Combiner\", 32).",
				Input: map[string]any{
					"exporter_hpke": hx(seed32("sage-spec/vectors/hpke/exporter")),
					"ss_e2e":        hx(seed32("sage-spec/vectors/hpke/ss-e2e")),
					"context_id":    vecCtxID,
				},
				Produce: func(in map[string]any) (map[string]any, error) {
					exp, err := unhex(in, "exporter_hpke")
					if err != nil {
						return nil, err
					}
					ss, err := unhex(in, "ss_e2e")
					if err != nil {
						return nil, err
					}
					ectx := hpke.DefaultExportContext(in["context_id"].(string))
					seed, err := hpke.CombineSecrets(exp, ss, ectx)
					if err != nil {
						return nil, err
					}
					return map[string]any{"export_context_hex": hx(ectx), "seed": hx(seed)}, nil
				},
			},
			{
				Name: "traffic-keys", Mode: ModeDeterministic,
				Description: "DeriveTrafficKeys(seed): HMAC-SHA256 counter expansion with labels SAGE-c2s:key, SAGE-c2s:iv, SAGE-s2c:key, SAGE-s2c:iv, SAGE-cb-v1.",
				Input:       map[string]any{"seed": hx(seed32("sage-spec/vectors/hpke/seed"))},
				Produce: func(in map[string]any) (map[string]any, error) {
					seed, err := unhex(in, "seed")
					if err != nil {
						return nil, err
					}
					tk := hpke.DeriveTrafficKeys(seed)
					return map[string]any{
						"c2s_key": hx(tk.C2SKey), "c2s_iv": hx(tk.C2SIV),
						"s2c_key": hx(tk.S2CKey), "s2c_iv": hx(tk.S2CIV),
						"channel_binding": hx(tk.CB),
					}, nil
				},
			},
			{
				Name: "ack-tag", Mode: ModeDeterministic,
				Description: "MakeAckTag(seed, ctxID, nonce, kid, info, exportCtx, enc, ephC, ephS, initDID, respDID): " +
					"ackKey = expand(seed, \"SAGE-ack-key-v1\", 32); tag = HMAC-SHA256(ackKey, \"SAGE-ack-msg|v1|\" || len16(ctx)||ctx || len16(nonce)||nonce || len16(kid)||kid || SHA256(0x00||b_1 || 0x00||b_2 ...)).",
				Input: map[string]any{
					"seed": hx(seed32("sage-spec/vectors/hpke/seed")), "context_id": vecCtxID, "nonce": vecHPKENon, "kid": vecKID,
					"init_did": vecInitDID, "resp_did": vecRespDID,
					"enc": hx(seed32("sage-spec/vectors/hpke/enc")), "eph_c": hx(seed32("sage-spec/vectors/hpke/ephC")), "eph_s": hx(seed32("sage-spec/vectors/hpke/ephS")),
				},
				Produce: func(in map[string]any) (map[string]any, error) {
					seed, err := unhex(in, "seed")
					if err != nil {
						return nil, err
					}
					enc, _ := unhex(in, "enc")
					ephC, _ := unhex(in, "eph_c")
					ephS, _ := unhex(in, "eph_s")
					ctx, _ := str(in, "context_id")
					a, _ := str(in, "init_did")
					b, _ := str(in, "resp_did")
					info := hpke.DefaultInfo(ctx, a, b)
					ectx := hpke.DefaultExportContext(ctx)
					tag := hpke.MakeAckTag(seed, ctx, in["nonce"].(string), in["kid"].(string), info, ectx, enc, ephC, ephS, []byte(a), []byte(b))
					return map[string]any{"binds_order": []string{"info", "export_context", "enc", "eph_c", "eph_s", "init_did", "resp_did"}, "ack_tag": hx(tag)}, nil
				},
			},
			{
				Name: "hpke-export-roundtrip", Mode: ModeVerify,
				Description: "HPKE base-mode encapsulation to the responder's X25519 key with Export(exportCtx, 32). enc is random, so the vector stores enc and the exporter secret and the check re-derives the exporter on the receiving side.",
				Input:       map[string]any{"resp_label": labelX25519B, "context_id": vecCtxID, "init_did": vecInitDID, "resp_did": vecRespDID},
				Produce: func(in map[string]any) (map[string]any, error) {
					b, err := x25519FromLabel(in["resp_label"].(string))
					if err != nil {
						return nil, err
					}
					ctx, _ := str(in, "context_id")
					info := hpke.DefaultInfo(ctx, in["init_did"].(string), in["resp_did"].(string))
					ectx := hpke.DefaultExportContext(ctx)
					enc, exporter, err := keys.HPKEDeriveSharedSecretToX25519Peer(b.PublicKey(), info, ectx, 32)
					if err != nil {
						return nil, err
					}
					return map[string]any{"resp_public": hx(b.PublicKey().Bytes()), "enc": hx(enc), "exporter": hx(exporter)}, nil
				},
				Verify: func(in, out map[string]any) error {
					b, err := x25519FromLabel(in["resp_label"].(string))
					if err != nil {
						return err
					}
					enc, err := unhex(out, "enc")
					if err != nil {
						return err
					}
					want, err := unhex(out, "exporter")
					if err != nil {
						return err
					}
					ctx, _ := str(in, "context_id")
					info := hpke.DefaultInfo(ctx, in["init_did"].(string), in["resp_did"].(string))
					ectx := hpke.DefaultExportContext(ctx)
					got, err := keys.HPKEOpenSharedSecretWithX25519Priv(b, enc, info, ectx, 32)
					if err != nil {
						return err
					}
					if !bytes.Equal(got, want) {
						return errors.New("exporter secret mismatch")
					}
					return nil
				},
			},
		},
	}
}
