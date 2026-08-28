package runner

import (
	"bufio"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/muhfaris/gherkio/internal/model"
)

type respExchange struct {
	want     []string
	response string
}

func serveRESP(t *testing.T, exchanges []respExchange) (string, <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		defer listener.Close()
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			done <- acceptErr
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		for _, exchange := range exchanges {
			raw, readErr := readRESP(r)
			if readErr != nil {
				done <- readErr
				return
			}
			items, ok := raw.([]interface{})
			if !ok {
				done <- fmt.Errorf("request is %T, want RESP array", raw)
				return
			}
			got := make([]string, len(items))
			for i, item := range items {
				got[i] = fmt.Sprint(item)
			}
			if !reflect.DeepEqual(got, exchange.want) {
				done <- fmt.Errorf("command = %v, want %v", got, exchange.want)
				return
			}
			if _, writeErr := fmt.Fprint(conn, exchange.response); writeErr != nil {
				done <- writeErr
				return
			}
		}
		done <- nil
	}()
	return listener.Addr().String(), done
}

func TestExecuteRedisGetDecodesJSON(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			done <- acceptErr
			return
		}
		defer conn.Close()
		request, readErr := readRESP(bufio.NewReader(conn))
		if readErr != nil {
			done <- readErr
			return
		}
		parts := request.([]interface{})
		if fmt.Sprint(parts[0]) != "GET" || fmt.Sprint(parts[1]) != "product:42" {
			done <- fmt.Errorf("unexpected command: %v", parts)
			return
		}
		payload := `{"id":42,"name":"tea"}`
		_, writeErr := fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(payload), payload)
		done <- writeErr
	}()

	resp, err := executeRedis(model.RedisConnection{Address: listener.Addr().String()}, "get", "product:42")
	if err != nil {
		t.Fatal(err)
	}
	if serverErr := <-done; serverErr != nil {
		t.Fatal(serverErr)
	}
	parsed := resp.Parsed.(map[string]interface{})
	if parsed["exists"] != true {
		t.Fatalf("exists = %v, want true", parsed["exists"])
	}
	value := parsed["value"].(map[string]interface{})
	if value["name"] != "tea" {
		t.Fatalf("name = %v, want tea", value["name"])
	}
}

func TestRedisAssertionAndSavePaths(t *testing.T) {
	resp := &ResponseInfo{Parsed: map[string]interface{}{
		"exists": true,
		"value":  map[string]interface{}{"id": 42, "name": "tea"},
	}}
	assertion := evaluateAssertion("redis.value.id", 42, resp, nil, "", nil)
	if !assertion.Passed {
		t.Fatalf("assertion failed: %+v", assertion)
	}
	vars := map[string]interface{}{}
	warnings := extractValues(vars, map[string]string{"cachedName": "redis.value.name"}, resp, nil, nil)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if vars["cachedName"] != "tea" {
		t.Fatalf("cachedName = %v, want tea", vars["cachedName"])
	}
}

func TestExecuteRedisRejectsArbitraryCommands(t *testing.T) {
	_, err := executeRedis(model.RedisConnection{Address: "127.0.0.1:6379"}, "del", "product:42")
	if err == nil {
		t.Fatal("expected unsupported command error")
	}
}

func TestRunRedisStepUsesMatchersAndSave(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_, _ = readRESP(bufio.NewReader(conn))
		payload := `{"id":42,"name":"tea"}`
		_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(payload), payload)
	}()

	step := model.Step{
		Redis:  &model.RedisStep{Connection: "cache", Command: "get", Key: "product:$id"},
		Expect: model.Expect{Extra: map[string]interface{}{"redis.exists": true, "redis.value.id": 42}},
		Save:   map[string]string{"cachedName": "redis.value.name"},
	}
	vars := map[string]interface{}{"id": 42}
	env := &model.Environment{Connections: map[string]model.RedisConnection{"cache": {Type: "redis", Address: listener.Addr().String()}}}
	result := runRedisStep(step, env, vars, false, 0)
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	for _, assertion := range result.Assertions {
		if !assertion.Passed {
			t.Fatalf("assertion failed: %+v", assertion)
		}
	}
	if vars["cachedName"] != "tea" {
		t.Fatalf("cachedName = %v, want tea", vars["cachedName"])
	}
}

func TestExecuteRedisDiscoversPrimaryThroughSentinel(t *testing.T) {
	payload := `{"id":42,"name":"tea"}`
	primaryAddress, primaryDone := serveRESP(t, []respExchange{
		{want: []string{"AUTH", "redis-user", "redis-password"}, response: "+OK\r\n"},
		{want: []string{"GET", "product:42"}, response: fmt.Sprintf("$%d\r\n%s\r\n", len(payload), payload)},
	})
	host, port, err := net.SplitHostPort(primaryAddress)
	if err != nil {
		t.Fatal(err)
	}
	sentinelResponse := fmt.Sprintf("*2\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(host), host, len(port), port)
	sentinelAddress, sentinelDone := serveRESP(t, []respExchange{
		{want: []string{"AUTH", "sentinel-user", "sentinel-password"}, response: "+OK\r\n"},
		{want: []string{"SENTINEL", "get-master-addr-by-name", "mymaster"}, response: sentinelResponse},
	})

	resp, err := executeRedis(model.RedisConnection{
		Sentinel: &model.RedisSentinel{
			Master:    "mymaster",
			Addresses: []string{"127.0.0.1:1", sentinelAddress},
			Username:  "sentinel-user",
			Password:  "sentinel-password",
		},
		Username: "redis-user",
		Password: "redis-password",
	}, "get", "product:42")
	if err != nil {
		t.Fatal(err)
	}
	if err := <-sentinelDone; err != nil {
		t.Fatal(err)
	}
	if err := <-primaryDone; err != nil {
		t.Fatal(err)
	}
	value := resp.Parsed.(map[string]interface{})["value"].(map[string]interface{})
	if value["name"] != "tea" {
		t.Fatalf("name = %v, want tea", value["name"])
	}
}

func TestDiscoverRedisPrimarySupportsIPv6AddressFormatting(t *testing.T) {
	response := "*2\r\n$3\r\n::1\r\n$4\r\n6379\r\n"
	sentinelAddress, done := serveRESP(t, []respExchange{{
		want:     []string{"SENTINEL", "get-master-addr-by-name", "mymaster"},
		response: response,
	}})
	address, err := discoverRedisPrimary(model.RedisSentinel{Master: "mymaster", Addresses: []string{sentinelAddress}}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if address != net.JoinHostPort("::1", "6379") {
		t.Fatalf("address = %q, want %q", address, net.JoinHostPort("::1", "6379"))
	}
}
