package runner

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// BuiltinVars returns a map of built-in generator variables that are pre-injected
// into every test run. These are regenerated once per call so each run gets fresh values.
func BuiltinVars() map[string]interface{} {
	return map[string]interface{}{
		"uuid":          generateUUID(),
		"ulid":          generateULID(),
		"randomInt":     generateRandomInt(),
		"randomEmail":   generateRandomEmail(),
		"randomPhone":   generateRandomPhone(),
	}
}

// mu guards the ULID timestamp to ensure millisecond uniqueness within the same generation batch
var mu sync.Mutex
var lastULIDTimestamp time.Time

// generateUUID generates a UUID v4 string using crypto/rand.
func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback — should never happen on a healthy system
		return "fallback-uuid"
	}

	// Set version 4 bits
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits (10xx)
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// crockfordBase32 encodes a 10-bit value to a Crockford base32 character.
const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// ulidEncode encodes bytes to a Crockford base32 string of the given length.
func ulidEncode(bytes []byte, length int) string {
	result := make([]byte, length)
	for i := range result {
		// Take 5 bits at a time
		bitIndex := i * 5
		byteIndex := bitIndex / 8
		bitOffset := bitIndex % 8

		var val byte
		if byteIndex < len(bytes)-1 {
			// Combine bits from two bytes
			val = (bytes[byteIndex] << bitOffset) | (bytes[byteIndex+1] >> (8 - bitOffset))
		} else if byteIndex < len(bytes) {
			// Last byte
			val = bytes[byteIndex] << bitOffset
		}
		// Mask to5 bits
		val = (val >> 3) & 0x1f
		result[i] = crockfordBase32[val]
	}
	return string(result)
}

// generateULID generates a ULID string (26 characters, Crockford base32).
// Format: TTTTTTTTTTRRRRRRRRRRRRRRRR (10 chars timestamp +16 chars random)
func generateULID() string {
	now := time.Now()

	mu.Lock()
	// Ensure monotonically increasing timestamps within the same nanosecond
	if !now.After(lastULIDTimestamp) {
		now = lastULIDTimestamp.Add(1 * time.Millisecond)
	}
	lastULIDTimestamp = now
	mu.Unlock()

	// Timestamp: milliseconds since epoch
	ms := now.UnixMilli()

	// Encode timestamp (48 bits → 10 Crockford base32 chars)
	tsBytes := []byte{
		byte(ms >> 40),
		byte(ms >> 32),
		byte(ms >> 24),
		byte(ms >> 16),
		byte(ms >> 8),
		byte(ms),
	}
	tsEncoded := ulidEncode(tsBytes, 10)

	// Random: 80 bits → 16 Crockford base32 chars
	randomBytes := make([]byte, 10)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback
		return tsEncoded + "RRRRRRRRRRRRRRRR"
	}
	randomEncoded := ulidEncode(randomBytes, 16)

	return tsEncoded + randomEncoded
}

// generateRandomInt generates a random integer between 0 and 999999.
func generateRandomInt() int {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

// generateRandomEmail generates a random email address at example.com.
func generateRandomEmail() string {
	suffix, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("user_%06d@example.com", suffix.Int64())
}

// generateRandomPhone generates a random Indonesian-format phone number (+62 prefix).
func generateRandomPhone() string {
	// Generate 10 random digits after +62 (e.g. +6281234567890)
	n, _ := rand.Int(rand.Reader, big.NewInt(10000000000))
	return fmt.Sprintf("+628%d", n.Int64())
}
