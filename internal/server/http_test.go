package server_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	dbApi "github.com/tdx/rkv/db/api"

	"github.com/stretchr/testify/require"
	"github.com/tdx/rkv/internal/server"
)

func TestHTTPServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client *client,
		config *server.Config,
	){
		"put/get key-value pair over HTTP succeeds": testHTTPPutGet,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupHTTPTest(t)
			defer teardown()
			fn(t, client, config)
		})
	}
}

func setupHTTPTest(t *testing.T) (*client, *server.Config, func()) {

	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	raft, dir := getRaft(t, "1", true, "bolt")

	config := &server.Config{
		Db: raft,
	}

	server, err := server.NewHTTPServer(config)
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	client := &client{
		serverAddr: l.Addr().String(),
	}

	return client, config, func() {
		func() {
			ctx, cancel := context.WithTimeout(
				context.Background(), 5*time.Second)
			defer func() {
				cancel()
			}()
			server.Shutdown(ctx)
		}()
		l.Close()
		raft.Close()
		func() {
			os.RemoveAll(dir)
		}()
	}
}

func testHTTPPutGet(t *testing.T, client *client, config *server.Config) {

	var (
		tab = []byte{'t', 'a', 'b'}
		key = []byte{'k', 'e', 'y'}
		val = []byte{'v', 'a', 'l'}
	)

	err := client.Put(tab, key, val)
	require.NoError(t, err)

	v, err := client.Get(tab, key)
	t.Log("v:", string(v))
	require.NoError(t, err)
	require.Equal(t, val, v)

	err = client.Del(tab, key)
	require.NoError(t, err)

	v, err = client.Get(tab, key)
	t.Log("v:", string(v))
	require.Equal(t, true, dbApi.IsNoKeyError(err))
	require.Nil(t, v)

}

//
// wrapper for HTTP requests
//
type client struct {
	serverAddr string
}

func (c *client) Put(tab, key, val []byte) error {

	url := fmt.Sprintf("http://%s/db/put", c.serverAddr)

	keyHex := hex.EncodeToString(key)
	valHex := hex.EncodeToString(val)
	jsonStr := []byte(
		`{"tab":"tab", "key":"` + keyHex + `", "val":"` + valHex + `"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	return nil
}

func (c *client) Get(tab, key []byte) ([]byte, error) {

	keyHex := hex.EncodeToString(key)
	url := fmt.Sprintf("http://%s/db/get?tab=tab&key=%s", c.serverAddr, keyHex)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, dbApi.ErrNoKey(key)
		}
		return nil, fmt.Errorf("http status: %d", resp.StatusCode)
	}

	var r map[string]string

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	valHex, ok := r["val"]
	if !ok {
		return nil, dbApi.ErrNoKey(key)
	}

	return hex.DecodeString(valHex)
}

func (c *client) Del(tab, key []byte) error {

	url := fmt.Sprintf("http://%s/db/del", c.serverAddr)

	keyHex := hex.EncodeToString(key)
	jsonStr := []byte(`{"tab":"tab", "key":"` + keyHex + `"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return dbApi.ErrNoKey(key)
		}
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	return nil
}
