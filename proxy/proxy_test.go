package main

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

func Test_API(t *testing.T) {
	log.Printf("Cache URL for debugging: %s\n", os.Getenv("ACTIONS_CACHE_URL"))
	encodedToken := base32.StdEncoding.EncodeToString([]byte(os.Getenv("ACTIONS_RUNTIME_TOKEN")))
	log.Printf("Token for debugging: %s\n", encodedToken)

	cacheKey := fmt.Sprintf("key%d", time.Now().Unix())
	location, err := findCacheLocation(cacheKey)
	if err != nil {
		t.Error(err)
	}
	if location != "" {
		t.Error("cache should not exist")
	}

	cacheId, err := reserveCache(cacheKey)
	if err != nil {
		t.Error(err)
		return
	}

	var cacheEntrySize int64 = 100 * 1024 * 1024
	err = uploadCacheFromReader(cacheId, io.LimitReader(rand.Reader, cacheEntrySize))
	if err != nil {
		t.Error(err)
		return
	}

	location, err = findCacheLocation(cacheKey)
	if err != nil {
		t.Error(err)
		return
	}
	if location == "" {
		t.Error("cache should exist now!")
		return
	}

	resp, err := http.Get(location)
	if err != nil {
		t.Error(err)
		return
	}
	if resp.StatusCode != 200 {
		t.Errorf("Failed to download cache entry: %d %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()

	file, err := ioutil.TempFile("proxy", "testing")
	if err != nil {
		t.Error(err)
		return
	}

	writtenBytes, err := io.Copy(file, resp.Body)

	if err != nil {
		t.Error(err)
		return
	}

	if writtenBytes != cacheEntrySize {
		t.Errorf("Downloaded only %d bytes!", writtenBytes)
		return
	}
}
