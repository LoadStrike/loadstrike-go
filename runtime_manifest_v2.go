package loadstrike

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

const (
	runtimeManifestSchemaVersion      = uint32(2)
	runtimeManifestSignatureVersion   = "publisher-ed25519-v2"
	runtimeManifestSignatureDomain    = "loadstrike.runtime-manifest"
	runtimeManifestSignaturePurpose   = "publisher-artifact-authentication"
	maxRuntimeManifestBytes           = 64 * 1024
	maxRuntimeManifestPayloadBytes    = 16 * 1024
	maxRuntimeArtifactBytes           = 128 * 1024 * 1024
	maxRuntimeManifestSignatures      = 3
	maxRuntimePublisherTrustedKeys    = 8
	maxRuntimePublisherKeyringBytes   = 16 * 1024
	maxRuntimeManifestSupportSeconds  = uint64(1_095 * 24 * 60 * 60)
	maxRuntimeManifestActivationSkew  = uint64(5 * 60)
	minRuntimeManifestUnixSecond      = uint64(1_700_000_000)
	maxRuntimeManifestUnixSecond      = uint64(4_102_444_800)
	runtimeManifestProductionOrigin   = "https://licensing.loadstrike.com"
	runtimeManifestPublisher          = "LoadStrike"
	runtimeManifestRepository         = "Meticulis/LoadStrike"
	runtimeManifestSourceRef          = "refs/heads/main"
	runtimeManifestWorkflowIdentity   = "Meticulis/LoadStrike/.github/workflows/main-delivery.yml@refs/heads/main"
	runtimeManifestWrapperModule      = "loadstrike.com/sdk/go"
	runtimeManifestAttestationMedia   = "application/vnd.dev.sigstore.bundle+json;version=0.3"
	runtimeManifestPublisherKeyPrefix = "sha256:"
)

var (
	// runtimePublisherTrustedKeysJSON is assigned exactly once by the generated
	// release-only source file. The public wrapper intentionally contains no
	// fallback key or network-fetched trust root.
	runtimePublisherTrustedKeysJSON string

	runtimeCanonicalVersionPattern = regexp.MustCompile(
		`^v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`,
	)
	runtimeCanonicalToolchainPattern = regexp.MustCompile(
		`^go(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`,
	)
	runtimeCanonicalGitObjectPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
	runtimeCanonicalKeyIDPattern     = regexp.MustCompile(
		`^sha256:[0-9a-f]{64}$`,
	)
	runtimeCanonicalAttestationPattern = regexp.MustCompile(
		`^sha256:[0-9a-f]{64}$`,
	)
)

type runtimePublisherTrustedKey struct {
	Algorithm  string
	KeyID      string
	PublicKey  ed25519.PublicKey
	Activation uint64
	NotAfter   uint64
}

var runtimeManifestClaimFields = map[string]struct{}{
	"arch":                 {},
	"attestationDigest":    {},
	"attestationMediaType": {},
	"buildFlags":           {},
	"byteLength":           {},
	"downloadUrl":          {},
	"executable":           {},
	"issuedAt":             {},
	"manifestVersion":      {},
	"notBefore":            {},
	"os":                   {},
	"protocol":             {},
	"publisher":            {},
	"releaseId":            {},
	"repository":           {},
	"runAttempt":           {},
	"runId":                {},
	"runtimeVersion":       {},
	"sdk":                  {},
	"sha256":               {},
	"sourceRef":            {},
	"sourceSha":            {},
	"toolchain":            {},
	"validUntil":           {},
	"workflowIdentity":     {},
	"wrapperCommit":        {},
	"wrapperGoModSum":      {},
	"wrapperModule":        {},
	"wrapperSum":           {},
	"wrapperTagObject":     {},
	"wrapperTree":          {},
	"wrapperVersion":       {},
}

func parseRuntimePublisherKeyring(raw string) ([]runtimePublisherTrustedKey, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("runtime publisher trust keyring is not provisioned")
	}
	if len(raw) > maxRuntimePublisherKeyringBytes {
		return nil, errors.New("runtime publisher public keyring exceeds the safety limit")
	}
	if !isStrictASCII([]byte(raw)) {
		return nil, errors.New("runtime publisher public keyring must be strict ASCII JSON")
	}

	entries, err := decodeStrictJSONArray(
		[]byte(raw),
		maxRuntimePublisherKeyringBytes,
		"runtime publisher public keyring",
	)
	if err != nil {
		return nil, err
	}
	if len(entries) < 1 || len(entries) > maxRuntimePublisherTrustedKeys {
		return nil, fmt.Errorf(
			"runtime publisher public keyring must contain one to %d keys",
			maxRuntimePublisherTrustedKeys,
		)
	}

	keys := make([]runtimePublisherTrustedKey, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for index, entry := range entries {
		source := fmt.Sprintf("runtime publisher public key %d", index+1)
		fields, err := decodeStrictJSONObject(
			bytes.NewReader(entry),
			int64(len(entry)),
			source,
			map[string]struct{}{
				"activation":         {},
				"algorithm":          {},
				"keyId":              {},
				"notAfter":           {},
				"publicKeyRawBase64": {},
			},
		)
		if err != nil {
			return nil, err
		}
		if len(fields) != 5 {
			return nil, fmt.Errorf("%s has unknown or missing fields", source)
		}

		var value struct {
			Activation         uint64 `json:"activation"`
			Algorithm          string `json:"algorithm"`
			KeyID              string `json:"keyId"`
			NotAfter           uint64 `json:"notAfter"`
			PublicKeyRawBase64 string `json:"publicKeyRawBase64"`
		}
		if err := json.Unmarshal(entry, &value); err != nil {
			return nil, fmt.Errorf("decode %s: %w", source, err)
		}
		if value.Algorithm != runtimeManifestSignatureVersion {
			return nil, fmt.Errorf("%s algorithm is unsupported", source)
		}
		publicKey, err := decodeCanonicalPaddedBase64(
			value.PublicKeyRawBase64,
			ed25519.PublicKeySize,
			"publicKeyRawBase64",
		)
		if err != nil {
			return nil, err
		}
		fingerprint := sha256.Sum256(publicKey)
		expectedKeyID := runtimeManifestPublisherKeyPrefix +
			hex.EncodeToString(fingerprint[:])
		if value.KeyID != expectedKeyID {
			return nil, errors.New(
				"runtime publisher key fingerprint does not match the raw public key",
			)
		}
		if _, duplicate := seen[value.KeyID]; duplicate {
			return nil, errors.New(
				"runtime publisher public keyring contains a duplicate fingerprint",
			)
		}
		if value.Activation < minRuntimeManifestUnixSecond ||
			value.Activation >= value.NotAfter ||
			value.NotAfter > maxRuntimeManifestUnixSecond {
			return nil, errors.New(
				"runtime publisher key activation interval is invalid",
			)
		}
		keys = append(keys, runtimePublisherTrustedKey{
			Algorithm:  value.Algorithm,
			KeyID:      value.KeyID,
			PublicKey:  append(ed25519.PublicKey(nil), publicKey...),
			Activation: value.Activation,
			NotAfter:   value.NotAfter,
		})
		seen[value.KeyID] = struct{}{}
	}
	return keys, nil
}

func configuredRuntimePublisherKeyring() ([]runtimePublisherTrustedKey, error) {
	return parseRuntimePublisherKeyring(runtimePublisherTrustedKeysJSON)
}

func parseRuntimeManifestV2(raw []byte) (runtimeArtifactManifest, error) {
	if len(raw) < 1 || len(raw) > maxRuntimeManifestBytes {
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest descriptor has an invalid size",
		)
	}
	if !isStrictASCII(raw) {
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest descriptor must be strict ASCII JSON",
		)
	}

	envelopeFields, err := decodeStrictJSONObject(
		bytes.NewReader(raw),
		maxRuntimeManifestBytes,
		"runtime manifest descriptor",
		map[string]struct{}{
			"claims":        {},
			"schemaVersion": {},
			"signatures":    {},
		},
	)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if len(envelopeFields) != 3 {
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest envelope has unknown or missing fields",
		)
	}

	claimsRaw := envelopeFields["claims"]
	claimFields, err := decodeStrictJSONObject(
		bytes.NewReader(claimsRaw),
		int64(len(claimsRaw)),
		"runtime manifest claims",
		runtimeManifestClaimFields,
	)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if len(claimFields) != len(runtimeManifestClaimFields) {
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest claims have unknown or missing fields",
		)
	}

	signatureEntries, err := decodeStrictJSONArray(
		envelopeFields["signatures"],
		maxRuntimeManifestBytes,
		"runtime manifest signatures",
	)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if len(signatureEntries) < 1 ||
		len(signatureEntries) > maxRuntimeManifestSignatures {
		return runtimeArtifactManifest{}, fmt.Errorf(
			"runtime manifest requires one to %d publisher signatures",
			maxRuntimeManifestSignatures,
		)
	}
	signatures := make([]runtimeArtifactSignature, 0, len(signatureEntries))
	for index, entry := range signatureEntries {
		source := fmt.Sprintf("runtime manifest signature %d", index+1)
		fields, err := decodeStrictJSONObject(
			bytes.NewReader(entry),
			int64(len(entry)),
			source,
			map[string]struct{}{
				"algorithm": {},
				"keyId":     {},
				"signature": {},
			},
		)
		if err != nil {
			return runtimeArtifactManifest{}, err
		}
		if len(fields) != 3 {
			return runtimeArtifactManifest{}, fmt.Errorf(
				"%s has unknown or missing fields",
				source,
			)
		}
		var signature runtimeArtifactSignature
		if err := json.Unmarshal(entry, &signature); err != nil {
			return runtimeArtifactManifest{}, fmt.Errorf(
				"decode %s: %w",
				source,
				err,
			)
		}
		signatures = append(signatures, signature)
	}

	canonical, err := canonicalizeRuntimeManifestJSON(raw)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if !bytes.Equal(canonical, raw) {
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest descriptor is not canonical JSON",
		)
	}

	var schemaVersion uint32
	if err := json.Unmarshal(envelopeFields["schemaVersion"], &schemaVersion); err != nil {
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest schemaVersion must be 2",
		)
	}
	if schemaVersion != runtimeManifestSchemaVersion {
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest schemaVersion must be 2",
		)
	}
	var claims runtimeArtifactClaims
	if err := json.Unmarshal(claimsRaw, &claims); err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf(
			"decode runtime manifest claims: %w",
			err,
		)
	}
	return runtimeArtifactManifest{
		SchemaVersion: schemaVersion,
		Claims:        claims,
		Signatures:    signatures,
		canonicalJSON: append([]byte(nil), canonical...),
	}, nil
}

func verifyRuntimeManifestV2(
	raw []byte,
	config runtimeResolverConfig,
	trustedKeys []runtimePublisherTrustedKey,
	now time.Time,
) (runtimeArtifactManifest, error) {
	if len(trustedKeys) < 1 ||
		len(trustedKeys) > maxRuntimePublisherTrustedKeys {
		return runtimeArtifactManifest{}, errors.New(
			"runtime publisher trust keyring is not provisioned",
		)
	}
	manifest, err := parseRuntimeManifestV2(raw)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if err := validateRuntimeManifestV2Claims(manifest.Claims, config, now); err != nil {
		return runtimeArtifactManifest{}, err
	}

	trusted := make(map[string]runtimePublisherTrustedKey, len(trustedKeys))
	for _, key := range trustedKeys {
		fingerprint := sha256.Sum256(key.PublicKey)
		expectedKeyID := runtimeManifestPublisherKeyPrefix +
			hex.EncodeToString(fingerprint[:])
		if key.Algorithm != runtimeManifestSignatureVersion ||
			!runtimeCanonicalKeyIDPattern.MatchString(key.KeyID) ||
			len(key.PublicKey) != ed25519.PublicKeySize ||
			key.KeyID != expectedKeyID ||
			key.Activation < minRuntimeManifestUnixSecond ||
			key.Activation >= key.NotAfter ||
			key.NotAfter > maxRuntimeManifestUnixSecond {
			return runtimeArtifactManifest{}, errors.New(
				"runtime publisher trust keyring contains an invalid key",
			)
		}
		if _, duplicate := trusted[key.KeyID]; duplicate {
			return runtimeArtifactManifest{}, errors.New(
				"runtime publisher trust keyring contains a duplicate fingerprint",
			)
		}
		trusted[key.KeyID] = key
	}

	seen := make(map[string]struct{}, len(manifest.Signatures))
	trustedValid := 0
	previousKeyID := ""
	for _, signature := range manifest.Signatures {
		if signature.Algorithm != runtimeManifestSignatureVersion ||
			!runtimeCanonicalKeyIDPattern.MatchString(signature.KeyID) {
			return runtimeArtifactManifest{}, errors.New(
				"runtime manifest signature algorithm or key fingerprint is invalid",
			)
		}
		if _, duplicate := seen[signature.KeyID]; duplicate {
			return runtimeArtifactManifest{}, errors.New(
				"runtime manifest contains duplicate publisher signatures",
			)
		}
		if previousKeyID != "" && signature.KeyID <= previousKeyID {
			return runtimeArtifactManifest{}, errors.New(
				"runtime manifest publisher signatures are not canonically ordered",
			)
		}
		decodedSignature, err := decodeCanonicalPaddedBase64(
			signature.Signature,
			ed25519.SignatureSize,
			"signature",
		)
		if err != nil {
			return runtimeArtifactManifest{}, err
		}

		key, known := trusted[signature.KeyID]
		if known {
			if key.Algorithm != signature.Algorithm {
				return runtimeArtifactManifest{}, errors.New(
					"runtime manifest signature algorithm does not match its trusted key",
				)
			}
			if manifest.Claims.IssuedAt < key.Activation ||
				manifest.Claims.ValidUntil > key.NotAfter {
				return runtimeArtifactManifest{}, errors.New(
					"runtime manifest validity exceeds a trusted publisher key interval",
				)
			}
			payload, err := runtimeManifestV2CanonicalPayload(
				manifest.Claims,
				signature.Algorithm,
				signature.KeyID,
			)
			if err != nil {
				return runtimeArtifactManifest{}, err
			}
			if !ed25519.Verify(key.PublicKey, payload, decodedSignature) {
				return runtimeArtifactManifest{}, errors.New(
					"runtime publisher signature is invalid",
				)
			}
			trustedValid++
		}
		seen[signature.KeyID] = struct{}{}
		previousKeyID = signature.KeyID
	}
	if trustedValid == 0 {
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest has no trusted valid publisher signature",
		)
	}
	return manifest, nil
}

func validateRuntimeManifestV2Claims(
	claims runtimeArtifactClaims,
	config runtimeResolverConfig,
	now time.Time,
) error {
	if claims.ManifestVersion != runtimeManifestSchemaVersion {
		return errors.New("runtime manifest manifestVersion must be 2")
	}
	if claims.SDK != "go" {
		return errors.New("runtime manifest sdk must be go")
	}
	if len(claims.RuntimeVersion) > 64 ||
		!runtimeCanonicalVersionPattern.MatchString(claims.RuntimeVersion) {
		return errors.New("runtime manifest runtimeVersion is not canonical")
	}
	expectedExecutables := map[string]string{
		"windows-amd64": "loadstrike-runtime.exe",
		"linux-amd64":   "loadstrike-runtime",
		"darwin-arm64":  "loadstrike-runtime",
	}
	platform := claims.OS + "-" + claims.Arch
	executable, supported := expectedExecutables[platform]
	if !supported || claims.Executable != executable {
		return errors.New("runtime manifest platform is unsupported")
	}
	if claims.Protocol < 1 || claims.Protocol > uint32(^uint32(0)>>1) {
		return errors.New("runtime manifest protocol is invalid")
	}
	if !isLowerHex64(claims.SHA256) {
		return errors.New("runtime manifest sha256 is not canonical")
	}
	if claims.ByteLength < 1 || claims.ByteLength > maxRuntimeArtifactBytes {
		return errors.New("runtime manifest byteLength is invalid")
	}
	expectedURL := fmt.Sprintf(
		"%s/runtime/%s/go/%s/%s",
		runtimeManifestProductionOrigin,
		claims.RuntimeVersion,
		platform,
		claims.Executable,
	)
	if claims.DownloadURL != expectedURL {
		return errors.New(
			"runtime manifest downloadUrl is not the production artifact URL",
		)
	}
	if claims.IssuedAt < minRuntimeManifestUnixSecond ||
		claims.IssuedAt > claims.NotBefore ||
		claims.NotBefore > claims.ValidUntil ||
		claims.ValidUntil > maxRuntimeManifestUnixSecond ||
		claims.NotBefore-claims.IssuedAt > maxRuntimeManifestActivationSkew ||
		claims.ValidUntil-claims.IssuedAt > maxRuntimeManifestSupportSeconds {
		return errors.New("runtime manifest validity interval is invalid")
	}
	fixed := map[string][2]string{
		"publisher": {
			claims.Publisher,
			runtimeManifestPublisher,
		},
		"releaseId": {
			claims.ReleaseID,
			fmt.Sprintf("go-runtime/%s/%s", claims.RuntimeVersion, platform),
		},
		"repository": {
			claims.Repository,
			runtimeManifestRepository,
		},
		"sourceRef": {
			claims.SourceRef,
			runtimeManifestSourceRef,
		},
		"workflowIdentity": {
			claims.WorkflowIdentity,
			runtimeManifestWorkflowIdentity,
		},
		"wrapperModule": {
			claims.WrapperModule,
			runtimeManifestWrapperModule,
		},
		"wrapperVersion": {
			claims.WrapperVersion,
			claims.RuntimeVersion,
		},
		"buildFlags": {
			claims.BuildFlags,
			expectedRuntimeManifestBuildFlags(claims.RuntimeVersion),
		},
		"attestationMediaType": {
			claims.AttestationMediaType,
			runtimeManifestAttestationMedia,
		},
	}
	for name, values := range fixed {
		if values[0] != values[1] {
			return fmt.Errorf(
				"runtime manifest %s does not match the publisher identity",
				name,
			)
		}
	}
	for name, value := range map[string]string{
		"sourceSha":        claims.SourceSHA,
		"wrapperTagObject": claims.WrapperTagObject,
		"wrapperCommit":    claims.WrapperCommit,
		"wrapperTree":      claims.WrapperTree,
	} {
		if !runtimeCanonicalGitObjectPattern.MatchString(value) {
			return fmt.Errorf("runtime manifest %s is not canonical", name)
		}
	}
	if claims.RunID < 1 || claims.RunID > 9_999_999_999_999_999_999 ||
		claims.RunAttempt < 1 || claims.RunAttempt > 1_000 {
		return errors.New("runtime manifest workflow run identity is invalid")
	}
	if err := validateCanonicalGoH1(claims.WrapperSum, "wrapperSum"); err != nil {
		return err
	}
	if err := validateCanonicalGoH1(
		claims.WrapperGoModSum,
		"wrapperGoModSum",
	); err != nil {
		return err
	}
	if len(claims.Toolchain) > 32 ||
		!runtimeCanonicalToolchainPattern.MatchString(claims.Toolchain) {
		return errors.New("runtime manifest toolchain is not canonical")
	}
	if !runtimeCanonicalAttestationPattern.MatchString(
		claims.AttestationDigest,
	) {
		return errors.New(
			"runtime manifest attestationDigest is not canonical",
		)
	}

	if claims.RuntimeVersion != config.Version {
		return RuntimeMismatchError{
			ExpectedVersion: config.Version,
			ActualVersion:   claims.RuntimeVersion,
		}
	}
	if claims.OS != config.GOOS || claims.Arch != config.GOARCH {
		return fmt.Errorf(
			"runtime manifest platform mismatch: expected %s-%s, got %s-%s",
			config.GOOS,
			config.GOARCH,
			claims.OS,
			claims.Arch,
		)
	}
	if claims.Protocol != uint32(RuntimeProtocolVersion()) {
		return fmt.Errorf(
			"loadstrike runtime protocol mismatch: expected %d, got %d",
			RuntimeProtocolVersion(),
			claims.Protocol,
		)
	}
	nowUnix := now.UTC().Unix()
	if nowUnix < 0 || uint64(nowUnix) < claims.NotBefore {
		return errors.New("runtime manifest is not active")
	}
	if uint64(nowUnix) > claims.ValidUntil {
		return errors.New("runtime manifest has expired")
	}
	return nil
}

func runtimeManifestV2CanonicalPayload(
	claims runtimeArtifactClaims,
	algorithm string,
	keyID string,
) ([]byte, error) {
	if algorithm != runtimeManifestSignatureVersion {
		return nil, errors.New(
			"runtime manifest signature algorithm is unsupported",
		)
	}
	if !runtimeCanonicalKeyIDPattern.MatchString(keyID) {
		return nil, errors.New(
			"runtime manifest signature key fingerprint is invalid",
		)
	}

	type field struct {
		id    uint16
		value []byte
	}
	fields := []field{
		{1, []byte(runtimeManifestSignatureDomain)},
		{2, []byte(runtimeManifestSignaturePurpose)},
		{3, encodeRuntimeManifestUint32(runtimeManifestSchemaVersion)},
		{4, []byte(algorithm)},
		{5, []byte(keyID)},
		{10, encodeRuntimeManifestUint32(claims.ManifestVersion)},
		{11, []byte(claims.SDK)},
		{12, []byte(claims.RuntimeVersion)},
		{13, []byte(claims.OS)},
		{14, []byte(claims.Arch)},
		{15, encodeRuntimeManifestUint32(claims.Protocol)},
		{16, []byte(claims.SHA256)},
		{17, encodeRuntimeManifestUint64(claims.ByteLength)},
		{18, []byte(claims.DownloadURL)},
		{19, []byte(claims.Executable)},
		{20, encodeRuntimeManifestUint64(claims.IssuedAt)},
		{21, encodeRuntimeManifestUint64(claims.NotBefore)},
		{22, encodeRuntimeManifestUint64(claims.ValidUntil)},
		{23, []byte(claims.Publisher)},
		{24, []byte(claims.ReleaseID)},
		{25, []byte(claims.Repository)},
		{26, []byte(claims.SourceRef)},
		{27, []byte(claims.SourceSHA)},
		{28, []byte(claims.WorkflowIdentity)},
		{29, encodeRuntimeManifestUint64(claims.RunID)},
		{30, encodeRuntimeManifestUint32(claims.RunAttempt)},
		{31, []byte(claims.WrapperModule)},
		{32, []byte(claims.WrapperVersion)},
		{33, []byte(claims.WrapperSum)},
		{34, []byte(claims.WrapperGoModSum)},
		{35, []byte(claims.WrapperTagObject)},
		{36, []byte(claims.WrapperCommit)},
		{37, []byte(claims.WrapperTree)},
		{38, []byte(claims.Toolchain)},
		{39, []byte(claims.BuildFlags)},
		{40, []byte(claims.AttestationDigest)},
		{41, []byte(claims.AttestationMediaType)},
	}
	if len(fields) != 37 {
		return nil, errors.New("runtime manifest signature field count is invalid")
	}

	var payload bytes.Buffer
	if err := binary.Write(
		&payload,
		binary.BigEndian,
		uint16(len(fields)),
	); err != nil {
		return nil, err
	}
	for _, entry := range fields {
		if !runtimeManifestNumericField(entry.id) &&
			!isStrictASCII(entry.value) {
			return nil, errors.New(
				"runtime manifest signature fields must be ASCII",
			)
		}
		if err := binary.Write(
			&payload,
			binary.BigEndian,
			entry.id,
		); err != nil {
			return nil, err
		}
		if err := binary.Write(
			&payload,
			binary.BigEndian,
			uint32(len(entry.value)),
		); err != nil {
			return nil, err
		}
		if _, err := payload.Write(entry.value); err != nil {
			return nil, err
		}
	}
	if payload.Len() > maxRuntimeManifestPayloadBytes {
		return nil, errors.New(
			"runtime manifest signature payload is too large",
		)
	}
	return payload.Bytes(), nil
}

func expectedRuntimeManifestBuildFlags(version string) string {
	return "-mod=readonly -trimpath -buildvcs=false " +
		"-ldflags=-X=github.com/Meticulis/LoadStrike/sdk/go-runtime-private/" +
		"internal/engine.moduleVersion=LOADSTRIKE_RUNTIME_VERSION:" +
		version + " CGO_ENABLED=0"
}

func validateCanonicalGoH1(value string, field string) error {
	if !strings.HasPrefix(value, "h1:") {
		return fmt.Errorf(
			"runtime manifest %s must be a canonical Go h1 checksum",
			field,
		)
	}
	if _, err := decodeCanonicalPaddedBase64(
		strings.TrimPrefix(value, "h1:"),
		sha256.Size,
		field,
	); err != nil {
		return fmt.Errorf(
			"runtime manifest %s must be a canonical Go h1 checksum",
			field,
		)
	}
	return nil
}

func decodeCanonicalPaddedBase64(
	value string,
	expectedLength int,
	field string,
) ([]byte, error) {
	if value == "" || len(value) > 512 {
		return nil, fmt.Errorf("%s must be canonical padded base64", field)
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil ||
		len(decoded) != expectedLength ||
		base64.StdEncoding.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("%s must be canonical padded base64", field)
	}
	return decoded, nil
}

func canonicalizeRuntimeManifestJSON(raw []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode runtime manifest descriptor: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New(
				"runtime manifest descriptor contains trailing JSON content",
			)
		}
		return nil, fmt.Errorf(
			"decode runtime manifest descriptor trailing content: %w",
			err,
		)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf(
			"encode canonical runtime manifest descriptor: %w",
			err,
		)
	}
	return append(encoded, '\n'), nil
}

func decodeStrictJSONArray(
	raw []byte,
	maximumBytes int,
	source string,
) ([]json.RawMessage, error) {
	if len(raw) > maximumBytes {
		return nil, fmt.Errorf("%s exceeds the safety limit", source)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", source, err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '[' {
		return nil, fmt.Errorf("decode %s: JSON value must be an array", source)
	}
	var entries []json.RawMessage
	for decoder.More() {
		var entry json.RawMessage
		if err := decoder.Decode(&entry); err != nil {
			return nil, fmt.Errorf("decode %s: %w", source, err)
		}
		entries = append(entries, append(json.RawMessage(nil), entry...))
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("decode %s: %w", source, err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf(
				"decode %s: trailing JSON content is not allowed",
				source,
			)
		}
		return nil, fmt.Errorf(
			"decode %s: invalid trailing content: %w",
			source,
			err,
		)
	}
	return entries, nil
}

func isStrictASCII(value []byte) bool {
	for _, character := range value {
		if character > 0x7f {
			return false
		}
	}
	return true
}

func encodeRuntimeManifestUint32(value uint32) []byte {
	encoded := make([]byte, 4)
	binary.BigEndian.PutUint32(encoded, value)
	return encoded
}

func encodeRuntimeManifestUint64(value uint64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, value)
	return encoded
}

func runtimeManifestNumericField(id uint16) bool {
	switch id {
	case 3, 10, 15, 17, 20, 21, 22, 29, 30:
		return true
	default:
		return false
	}
}
