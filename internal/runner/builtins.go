package runner

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BuiltinVars returns a map of built-in generator variables that are pre-injected
// into every test run. These are regenerated once per call so each run gets fresh values.
func BuiltinVars() map[string]interface{} {
	return map[string]interface{}{
		"uuid":        generateUUID(),
		"ulid":        generateULID(),
		"randomInt":   generateRandomInt(),
		"randomEmail": generateRandomEmail(),
		"randomPhone": generateRandomPhone(),
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

// generateRandomInt generates a random integer between 0 and 999999 (inclusive).
func generateRandomInt() int {
	return generateRandomIntInRange(0, 999999)
}

// generateRandomIntInRange returns a random integer in [min, max] inclusive.
// If min > max, they are swapped to ensure a valid range.
func generateRandomIntInRange(min, max int) int {
	if min > max {
		min, max = max, min
	}
	rangeSize := big.NewInt(int64(max - min + 1))
	n, err := rand.Int(rand.Reader, rangeSize)
	if err != nil {
		return min
	}
	return min + int(n.Int64())
}

// parseCustomDuration parses offset strings like "+14d", "-2h", "30m", "-5s"
func parseCustomDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration offset")
	}

	multiplier := time.Duration(1)
	cleanStr := s
	if strings.HasPrefix(s, "+") {
		cleanStr = s[1:]
	} else if strings.HasPrefix(s, "-") {
		multiplier = time.Duration(-1)
		cleanStr = s[1:]
	}

	if cleanStr == "" {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	if strings.HasSuffix(cleanStr, "d") {
		daysStr := strings.TrimSuffix(cleanStr, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("invalid days value in duration %q: %w", s, err)
		}
		return time.Duration(days) * 24 * time.Hour * multiplier, nil
	}

	dur, err := time.ParseDuration(cleanStr)
	if err != nil {
		return 0, fmt.Errorf("invalid time duration %q: %w", s, err)
	}

	return dur * multiplier, nil
}

// GeneratorFunc is a function that generates a value from a comma-separated arguments string.
type GeneratorFunc func(args string) (interface{}, error)

// GetGeneratorFuncs returns a map of parametrized generator functions keyed by variable name.
// These support the ${func(arg1,arg2)} syntax in variable interpolation.
func GetGeneratorFuncs() map[string]GeneratorFunc {
	return map[string]GeneratorFunc{
		"randomInt": func(args string) (interface{}, error) {
			parts := strings.SplitN(args, ",", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("randomInt requires exactly 2 arguments (min,max), got %d", len(parts))
			}
			min, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("randomInt min argument is not a valid integer: %s", parts[0])
			}
			max, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("randomInt max argument is not a valid integer: %s", parts[1])
			}
			return generateRandomIntInRange(min, max), nil
		},
		"timestamp": func(args string) (interface{}, error) {
			return strconv.FormatInt(time.Now().Unix(), 10), nil
		},
		"timestampMs": func(args string) (interface{}, error) {
			return strconv.FormatInt(time.Now().UnixMilli(), 10), nil
		},
		"dateNow": func(args string) (interface{}, error) {
			format := "2006-01-02 15:04:05"
			if args != "" {
				format = strings.Trim(strings.TrimSpace(args), "\"'")
			}
			return time.Now().Format(format), nil
		},
		"dateOffset": func(args string) (interface{}, error) {
			parts := strings.SplitN(args, ",", 2)
			if len(parts) < 1 || strings.TrimSpace(parts[0]) == "" {
				return nil, fmt.Errorf("dateOffset requires at least duration offset")
			}
			offsetStr := strings.Trim(strings.TrimSpace(parts[0]), "\"'")
			format := "2006-01-02"
			if len(parts) > 1 {
				format = strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			}

			duration, err := parseCustomDuration(offsetStr)
			if err != nil {
				return nil, err
			}

			return time.Now().Add(duration).Format(format), nil
		},
		"base64": func(args string) (interface{}, error) {
			data := strings.Trim(strings.TrimSpace(args), "\"'")
			return base64.StdEncoding.EncodeToString([]byte(data)), nil
		},
		"base64Decode": func(args string) (interface{}, error) {
			data := strings.Trim(strings.TrimSpace(args), "\"'")
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64: %w", err)
			}
			return string(decoded), nil
		},
		"urlencode": func(args string) (interface{}, error) {
			data := strings.Trim(strings.TrimSpace(args), "\"'")
			return url.QueryEscape(data), nil
		},
		"urldecode": func(args string) (interface{}, error) {
			data := strings.Trim(strings.TrimSpace(args), "\"'")
			decoded, err := url.QueryUnescape(data)
			if err != nil {
				return nil, fmt.Errorf("failed to decode url: %w", err)
			}
			return decoded, nil
		},
		"hash": func(args string) (interface{}, error) {
			parts := strings.SplitN(args, ",", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("hash requires exactly 2 arguments (algo, data)")
			}
			algo := strings.ToLower(strings.TrimSpace(parts[0]))
			data := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

			switch algo {
			case "md5":
				h := md5.Sum([]byte(data))
				return hex.EncodeToString(h[:]), nil
			case "sha1":
				h := sha1.Sum([]byte(data))
				return hex.EncodeToString(h[:]), nil
			case "sha256":
				h := sha256.Sum256([]byte(data))
				return hex.EncodeToString(h[:]), nil
			default:
				return nil, fmt.Errorf("unsupported hashing algorithm: %s", algo)
			}
		},
		"hmac": func(args string) (interface{}, error) {
			parts := strings.SplitN(args, ",", 3)
			if len(parts) != 3 {
				return nil, fmt.Errorf("hmac requires exactly 3 arguments (algo, key, message)")
			}
			algo := strings.ToLower(strings.TrimSpace(parts[0]))
			key := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			message := strings.Trim(strings.TrimSpace(parts[2]), "\"'")

			var mac []byte
			switch algo {
			case "md5":
				h := hmac.New(md5.New, []byte(key))
				h.Write([]byte(message))
				mac = h.Sum(nil)
			case "sha1":
				h := hmac.New(sha1.New, []byte(key))
				h.Write([]byte(message))
				mac = h.Sum(nil)
			case "sha256":
				h := hmac.New(sha256.New, []byte(key))
				h.Write([]byte(message))
				mac = h.Sum(nil)
			default:
				return nil, fmt.Errorf("unsupported hmac hashing algorithm: %s", algo)
			}
			return hex.EncodeToString(mac), nil
		},
		"randomString": func(args string) (interface{}, error) {
			parts := strings.SplitN(args, ",", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("randomString requires exactly 2 arguments (length, charset)")
			}
			length, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil || length <= 0 {
				return nil, fmt.Errorf("randomString length must be a positive integer: %s", parts[0])
			}
			charset := strings.ToLower(strings.Trim(strings.TrimSpace(parts[1]), "\"'"))

			var chars string
			switch charset {
			case "alpha":
				chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
			case "numeric":
				chars = "0123456789"
			case "alphanumeric":
				chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			default:
				return nil, fmt.Errorf("unsupported charset: %s", charset)
			}

			result := make([]byte, length)
			for i := 0; i < length; i++ {
				n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
				if err != nil {
					return nil, fmt.Errorf("failed to generate random character: %w", err)
				}
				result[i] = chars[n.Int64()]
			}
			return string(result), nil
		},
		"randomEmail": func(args string) (interface{}, error) {
			return generateRandomEmail(), nil
		},
		"randomPhone": func(args string) (interface{}, error) {
			countryOrPrefix := strings.Trim(strings.TrimSpace(args), "\"'")
			if countryOrPrefix == "" {
				return generateRandomPhone(), nil
			}

			// If it's a known ISO country code, generate matching pattern
			upper := strings.ToUpper(countryOrPrefix)
			if config, exists := countryPhonePrefixes[upper]; exists {
				return generateRandomDigitsWithPrefix(config.Prefix, config.DigitLength), nil
			}

			// If it's a raw plus/numeric prefix (e.g. "+351" or "351")
			cleanPrefix := countryOrPrefix
			if !strings.HasPrefix(cleanPrefix, "+") {
				if _, err := strconv.Atoi(cleanPrefix); err == nil {
					cleanPrefix = "+" + cleanPrefix
				}
			}
			if strings.HasPrefix(cleanPrefix, "+") {
				// Default to appending 9 random digits for custom prefixes
				return generateRandomDigitsWithPrefix(cleanPrefix, 9), nil
			}

			return generateRandomPhone(), nil
		},
		"toUpper": func(args string) (interface{}, error) {
			val := strings.Trim(strings.TrimSpace(args), "\"'")
			return strings.ToUpper(val), nil
		},
		"toLower": func(args string) (interface{}, error) {
			val := strings.Trim(strings.TrimSpace(args), "\"'")
			return strings.ToLower(val), nil
		},
		"trim": func(args string) (interface{}, error) {
			val := strings.Trim(strings.TrimSpace(args), "\"'")
			return strings.TrimSpace(val), nil
		},
		"trimPrefix": func(args string) (interface{}, error) {
			parts := strings.SplitN(args, ",", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("trimPrefix requires exactly 2 arguments (value,prefix)")
			}
			value := strings.Trim(strings.TrimSpace(parts[0]), "\"'")
			prefix := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			return strings.TrimPrefix(value, prefix), nil
		},
		"trimSuffix": func(args string) (interface{}, error) {
			parts := strings.SplitN(args, ",", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("trimSuffix requires exactly 2 arguments (value,suffix)")
			}
			value := strings.Trim(strings.TrimSpace(parts[0]), "\"'")
			suffix := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			return strings.TrimSuffix(value, suffix), nil
		},
		"split": func(args string) (interface{}, error) {
			parts := strings.SplitN(args, ",", 3)
			if len(parts) != 3 {
				return nil, fmt.Errorf("split requires exactly 3 arguments (value,delimiter,index)")
			}
			value := strings.Trim(strings.TrimSpace(parts[0]), "\"'")
			delimiter := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			if delimiter == "" {
				return nil, fmt.Errorf("split delimiter cannot be empty")
			}
			indexRaw := strings.Trim(strings.TrimSpace(parts[2]), "\"'")
			index, err := strconv.Atoi(indexRaw)
			if err != nil || index < 0 {
				return nil, fmt.Errorf("split index must be a non-negative integer: %s", indexRaw)
			}
			segments := strings.Split(value, delimiter)
			if index >= len(segments) {
				return nil, fmt.Errorf("split index %d out of range (split produced %d segments)", index, len(segments))
			}
			return segments[index], nil
		},
	}
}

// countryPhonePrefixes defines standard dialing/mobile prefix profiles for major countries.
var countryPhonePrefixes = map[string]struct {
	Prefix      string
	DigitLength int
}{
	"ID": {"+628", 9},
	"US": {"+1", 10},
	"CA": {"+1", 10},
	"GB": {"+447", 9},
	"UK": {"+447", 9},
	"SG": {"+658", 7},
	"JP": {"+8190", 8},
	"DE": {"+491", 10},
	"FR": {"+336", 8},
	"AU": {"+614", 8},
	"IN": {"+919", 9},
	"CN": {"+861", 10},
	"BR": {"+55119", 8},
	"RU": {"+79", 9},
	"ZA": {"+277", 8},
	"KR": {"+8210", 8},
	"NL": {"+316", 8},
	"ES": {"+346", 8},
	"IT": {"+393", 9},
	"MY": {"+601", 8},
	"PH": {"+639", 9},
	"TH": {"+668", 8},
	"VN": {"+849", 8},
}

// generateRandomDigitsWithPrefix appends random numeric digits of a given length to a prefix string.
func generateRandomDigitsWithPrefix(prefix string, length int) string {
	var sb strings.Builder
	sb.WriteString(prefix)
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		sb.WriteString(fmt.Sprintf("%d", n.Int64()))
	}
	return sb.String()
}

// generateRandomEmail generates a random email address at example.com.
func generateRandomEmail() string {
	suffix, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("user_%06d@example.com", suffix.Int64())
}

// generateRandomPhone generates a random Indonesian-format phone number (+62 prefix).
func generateRandomPhone() string {
	return generateRandomDigitsWithPrefix("+628", 9)
}

// LoadGherkioEnvVars loads all OS environment variables starting with the GHERKIO_ prefix.
func LoadGherkioEnvVars() map[string]interface{} {
	vars := make(map[string]interface{})
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			if strings.HasPrefix(key, "GHERKIO_") {
				vars[key] = parts[1]
			}
		}
	}
	return vars
}
