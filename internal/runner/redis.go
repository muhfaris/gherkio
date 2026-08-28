package runner

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/muhfaris/gherkio/internal/model"
)

var redisCommands = map[string]bool{"get": true, "exists": true, "ttl": true, "hgetall": true}

func executeRedis(connCfg model.RedisConnection, command, key string) (*ResponseInfo, error) {
	command = strings.ToLower(strings.TrimSpace(command))
	if !redisCommands[command] {
		return nil, fmt.Errorf("unsupported Redis command %q (supported: get, exists, ttl, hgetall)", command)
	}
	timeout := 5 * time.Second
	if connCfg.Timeout != "" {
		parsed, err := time.ParseDuration(connCfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid Redis timeout %q: %w", connCfg.Timeout, err)
		}
		duration := parsed
		timeout = duration
	}

	address := strings.TrimSpace(connCfg.Address)
	if connCfg.Sentinel != nil {
		if address != "" {
			return nil, fmt.Errorf("Redis connection cannot define both address and sentinel")
		}
		var err error
		address, err = discoverRedisPrimary(*connCfg.Sentinel, timeout)
		if err != nil {
			return nil, err
		}
	}
	if address == "" {
		return nil, fmt.Errorf("Redis connection requires address or sentinel configuration")
	}

	c, err := dialRedis(address, connCfg.TLS, timeout)
	if err != nil {
		return nil, fmt.Errorf("connect to Redis %s: %w", address, err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(timeout))

	r := bufio.NewReader(c)
	if connCfg.Password != "" {
		auth := []string{"AUTH", connCfg.Password}
		if connCfg.Username != "" {
			auth = []string{"AUTH", connCfg.Username, connCfg.Password}
		}
		if _, err := redisRoundTrip(c, r, auth...); err != nil {
			return nil, fmt.Errorf("Redis authentication failed: %w", err)
		}
	}
	if connCfg.Database != 0 {
		if _, err := redisRoundTrip(c, r, "SELECT", strconv.Itoa(connCfg.Database)); err != nil {
			return nil, fmt.Errorf("select Redis database %d: %w", connCfg.Database, err)
		}
	}

	raw, err := redisRoundTrip(c, r, strings.ToUpper(command), key)
	if err != nil {
		return nil, fmt.Errorf("Redis %s %q: %w", command, key, err)
	}
	result, err := normalizeRedisResult(command, raw)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(result)
	return &ResponseInfo{Body: string(body), Parsed: result, Headers: map[string]string{}}, nil
}

func dialRedis(address string, useTLS bool, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	if !useTLS {
		return dialer.Dial("tcp", address)
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis address %q: %w", address, err)
	}
	return tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: host,
	})
}

func discoverRedisPrimary(cfg model.RedisSentinel, timeout time.Duration) (string, error) {
	if strings.TrimSpace(cfg.Master) == "" {
		return "", fmt.Errorf("Redis Sentinel master name is required")
	}
	if len(cfg.Addresses) == 0 {
		return "", fmt.Errorf("Redis Sentinel requires at least one address")
	}

	var failures []string
	for _, address := range cfg.Addresses {
		address = strings.TrimSpace(address)
		if address == "" {
			failures = append(failures, "empty address")
			continue
		}
		primary, err := querySentinel(address, cfg, timeout)
		if err == nil {
			return primary, nil
		}
		failures = append(failures, fmt.Sprintf("%s: %v", address, err))
	}
	return "", fmt.Errorf("Redis Sentinel discovery for master %q failed (%s)", cfg.Master, strings.Join(failures, "; "))
}

func querySentinel(address string, cfg model.RedisSentinel, timeout time.Duration) (string, error) {
	c, err := dialRedis(address, cfg.TLS, timeout)
	if err != nil {
		return "", err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(timeout))
	r := bufio.NewReader(c)

	if cfg.Password != "" {
		auth := []string{"AUTH", cfg.Password}
		if cfg.Username != "" {
			auth = []string{"AUTH", cfg.Username, cfg.Password}
		}
		if _, err := redisRoundTrip(c, r, auth...); err != nil {
			return "", fmt.Errorf("Sentinel authentication failed: %w", err)
		}
	}
	raw, err := redisRoundTrip(c, r, "SENTINEL", "get-master-addr-by-name", cfg.Master)
	if err != nil {
		return "", err
	}
	parts, ok := raw.([]interface{})
	if !ok || len(parts) != 2 {
		return "", fmt.Errorf("master %q was not found", cfg.Master)
	}
	host, hostOK := parts[0].(string)
	port, portOK := parts[1].(string)
	if !hostOK || !portOK || host == "" || port == "" {
		return "", fmt.Errorf("Sentinel returned an invalid primary address")
	}
	if _, err := strconv.Atoi(port); err != nil {
		return "", fmt.Errorf("Sentinel returned invalid primary port %q", port)
	}
	return net.JoinHostPort(host, port), nil
}

func redisRoundTrip(w io.Writer, r *bufio.Reader, args ...string) (interface{}, error) {
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(args)); err != nil {
		return nil, err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return nil, err
		}
	}
	return readRESP(r)
}

func readRESP(r *bufio.Reader) (interface{}, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	line := func() (string, error) {
		s, e := r.ReadString('\n')
		if e != nil {
			return "", e
		}
		return strings.TrimSuffix(strings.TrimSuffix(s, "\n"), "\r"), nil
	}
	switch prefix {
	case '+':
		return line()
	case '-':
		msg, e := line()
		if e != nil {
			return nil, e
		}
		return nil, fmt.Errorf("%s", msg)
	case ':':
		s, e := line()
		if e != nil {
			return nil, e
		}
		return strconv.ParseInt(s, 10, 64)
	case '$':
		s, e := line()
		if e != nil {
			return nil, e
		}
		n, e := strconv.Atoi(s)
		if e != nil {
			return nil, e
		}
		if n == -1 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, e = io.ReadFull(r, buf); e != nil {
			return nil, e
		}
		return string(buf[:n]), nil
	case '*':
		s, e := line()
		if e != nil {
			return nil, e
		}
		n, e := strconv.Atoi(s)
		if e != nil {
			return nil, e
		}
		if n == -1 {
			return nil, nil
		}
		items := make([]interface{}, n)
		for i := range items {
			items[i], e = readRESP(r)
			if e != nil {
				return nil, e
			}
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported RESP prefix %q", prefix)
	}
}

func normalizeRedisResult(command string, raw interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{"command": command}
	switch command {
	case "get":
		result["exists"] = raw != nil
		if raw == nil {
			result["value"] = nil
			return result, nil
		}
		value := raw
		if s, ok := raw.(string); ok {
			var decoded interface{}
			if json.Unmarshal([]byte(s), &decoded) == nil {
				value = decoded
			}
		}
		result["value"] = value
	case "exists":
		result["exists"] = raw.(int64) > 0
	case "ttl":
		result["ttl"] = raw
	case "hgetall":
		items, ok := raw.([]interface{})
		if !ok || len(items)%2 != 0 {
			return nil, fmt.Errorf("invalid HGETALL response")
		}
		value := make(map[string]interface{}, len(items)/2)
		for i := 0; i < len(items); i += 2 {
			value[fmt.Sprint(items[i])] = items[i+1]
		}
		result["exists"] = len(value) > 0
		result["value"] = value
	}
	return result, nil
}

func runRedisStep(step model.Step, env *model.Environment, vars map[string]interface{}, dryRun bool, requestDelay time.Duration) StepResult {
	started := time.Now()
	result := StepResult{Name: step.Name, Original: step}
	redisStep := step.Redis
	if redisStep == nil {
		result.Error = "redis step is missing configuration"
		return result
	}

	connectionName, err := interpolateString(redisStep.Connection, vars)
	if err != nil {
		result.Error = fmt.Sprintf("Redis connection interpolation failed: %v", err)
		return result
	}
	key, err := interpolateString(redisStep.Key, vars)
	if err != nil {
		result.Error = fmt.Sprintf("Redis key interpolation failed: %v", err)
		return result
	}
	command := strings.ToLower(redisStep.Command)
	result.Redis = &RedisInfo{Connection: connectionName, Command: command, Key: key}
	if env == nil {
		result.Error = "active environment is required for Redis steps"
		return result
	}
	connCfg, ok := env.Connections[connectionName]
	if !ok {
		result.Error = fmt.Sprintf("Redis connection %q is not defined in the active environment", connectionName)
		return result
	}
	if connCfg.Type != "" && !strings.EqualFold(connCfg.Type, "redis") {
		result.Error = fmt.Sprintf("connection %q has type %q, expected redis", connectionName, connCfg.Type)
		return result
	}
	connCfg.Address, err = interpolateString(connCfg.Address, vars)
	if err != nil {
		result.Error = fmt.Sprintf("Redis address interpolation failed: %v", err)
		return result
	}
	connCfg.Username, err = interpolateString(connCfg.Username, vars)
	if err != nil {
		result.Error = fmt.Sprintf("Redis username interpolation failed: %v", err)
		return result
	}
	connCfg.Password, err = interpolateString(connCfg.Password, vars)
	if err != nil {
		result.Error = fmt.Sprintf("Redis password interpolation failed: %v", err)
		return result
	}
	if connCfg.Sentinel != nil {
		sentinelCopy := *connCfg.Sentinel
		sentinelCopy.Addresses = append([]string(nil), connCfg.Sentinel.Addresses...)
		connCfg.Sentinel = &sentinelCopy
		connCfg.Sentinel.Master, err = interpolateString(connCfg.Sentinel.Master, vars)
		if err != nil {
			result.Error = fmt.Sprintf("Redis Sentinel master interpolation failed: %v", err)
			return result
		}
		for i, address := range connCfg.Sentinel.Addresses {
			connCfg.Sentinel.Addresses[i], err = interpolateString(address, vars)
			if err != nil {
				result.Error = fmt.Sprintf("Redis Sentinel address interpolation failed: %v", err)
				return result
			}
		}
		connCfg.Sentinel.Username, err = interpolateString(connCfg.Sentinel.Username, vars)
		if err != nil {
			result.Error = fmt.Sprintf("Redis Sentinel username interpolation failed: %v", err)
			return result
		}
		connCfg.Sentinel.Password, err = interpolateString(connCfg.Sentinel.Password, vars)
		if err != nil {
			result.Error = fmt.Sprintf("Redis Sentinel password interpolation failed: %v", err)
			return result
		}
	}

	if dryRun {
		result.Duration = time.Since(started)
		return result
	}
	attempts, interval, backoff := 1, 500, "constant"
	if step.Retry != nil {
		if step.Retry.Attempts > 0 {
			attempts = step.Retry.Attempts
		}
		if step.Retry.Interval > 0 {
			interval = step.Retry.Interval
		}
		if step.Retry.Backoff != "" {
			backoff = step.Retry.Backoff
		}
	}
	var maxDuration time.Duration
	if step.Retry != nil && step.Retry.MaxDuration != "" {
		maxDuration, err = time.ParseDuration(step.Retry.MaxDuration)
		if err != nil {
			result.Error = fmt.Sprintf("invalid maxDuration: %v", err)
			return result
		}
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		if maxDuration > 0 && time.Since(started) >= maxDuration {
			result.Error = fmt.Sprintf("maxDuration %s exceeded", step.Retry.MaxDuration)
			break
		}
		attemptStart := time.Now()
		if requestDelay > 0 {
			time.Sleep(requestDelay)
		}
		resp, execErr := executeRedis(connCfg, command, key)
		entry := RetryEntry{Attempt: attempt, Duration: time.Since(attemptStart)}
		if execErr != nil {
			entry.Error = execErr.Error()
			result.RetryHistory = append(result.RetryHistory, entry)
			if attempt == attempts {
				result.Error = execErr.Error()
				break
			}
			time.Sleep(calculateBackoff(backoff, interval, attempt))
			continue
		}
		entry.Body = resp.Body
		result.RetryHistory = append(result.RetryHistory, entry)
		result.Response = resp

		extra := make(map[string]interface{}, len(step.Expect.Extra))
		for path, expected := range step.Expect.Extra {
			if s, ok := expected.(string); ok {
				if interpolated, interpolationErr := interpolateString(s, vars); interpolationErr == nil {
					expected = interpolated
				}
			}
			extra[path] = expected
		}
		result.Assertions = runAssertions(0, resp, nil, 0, extra, "", nil)
		allPass := true
		for _, assertion := range result.Assertions {
			if !assertion.Passed {
				allPass = false
				break
			}
		}
		if allPass {
			break
		}
		if attempt < attempts {
			time.Sleep(calculateBackoff(backoff, interval, attempt))
		}
	}
	if len(result.RetryHistory) > 1 {
		result.RetryCount = len(result.RetryHistory) - 1
	}
	if result.Response != nil && result.Error == "" {
		result.Warnings = extractValues(vars, step.Save, result.Response, nil, nil)
		if step.Save != nil {
			result.SavedVars = make(map[string]interface{})
			for name := range step.Save {
				if value, found := vars[name]; found {
					result.SavedVars[name] = value
				}
			}
		}
	}
	result.Duration = time.Since(started)
	if step.Timing.Max != "" {
		result.Assertions = append(result.Assertions, evaluateTiming(result.Duration, step.Timing.Max))
	}
	return result
}
