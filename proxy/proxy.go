package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dimchansky/utfbom"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var token = os.Getenv("ACTIONS_RUNTIME_TOKEN")
var httpClient = &http.Client{}

type GetCacheResponse struct {
	ArchiveLocation string `json:"archiveLocation"`
}

type ReserveCacheResponse struct {
	CacheId int `json:"cacheId"`
}

func main() {
	http.HandleFunc("/", handler)

	log.Println("Starting http cache server on port 12321")
	err := http.ListenAndServe(":12321", nil)
	if err != nil {
		log.Printf("Failed to start server: %v\n", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	key := r.URL.Path
	if key[0] == '/' {
		key = key[1:]
	}
	if key == "" {
		_, _ = w.Write([]byte("HTTP Cache is running!"))
		w.WriteHeader(200)
	} else if r.Method == "GET" {
		downloadCache(w, r, key)
	} else if r.Method == "HEAD" {
		checkCacheExists(w, key)
	} else if r.Method == "POST" {
		uploadCache(w, r, key)
	} else if r.Method == "PUT" {
		uploadCache(w, r, key)
	}
	duration := time.Since(startTime)
	log.Printf("Served %s request for %s key in %dms\n", r.Method, key, duration.Milliseconds())
}

func downloadCache(w http.ResponseWriter, r *http.Request, key string) {
	location, err := findCacheLocation(key)
	if err != nil {
		log.Printf("Failed to download key %s: %v\n", key, err)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	if location == "" {
		log.Printf("Cache %s not found\n", key)
		w.WriteHeader(404)
		return
	}
	proxyDownloadFromURL(w, location)
}

func proxyDownloadFromURL(w http.ResponseWriter, url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Proxying cache %s failed: %v\n", url, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	successfulStatus := 100 <= resp.StatusCode && resp.StatusCode < 300
	if !successfulStatus {
		log.Printf("Proxying cache %s failed with %d status\n", url, resp.StatusCode)
		w.WriteHeader(resp.StatusCode)
		return
	}
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func checkCacheExists(w http.ResponseWriter, key string) {
	location, err := findCacheLocation(key)
	if location == "" || err != nil {
		log.Printf("Cache %s not found\n", key)
		w.WriteHeader(404)
		return
	}
	w.WriteHeader(200)
}

func findCacheLocation(key string) (string, error) {
	resource := fmt.Sprintf("cache?keys=%s&version=%s", key, calculateSHA256(key))
	requestUrl := getCacheApiUrl(resource)
	request, _ := http.NewRequest("GET", requestUrl, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "actions/cache")
	request.Header.Set("Accept", "application/json;api-version=6.0-preview.1")
	request.Header.Set("Accept-Charset", "utf-8")

	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode == 404 {
		return "", nil
	}
	if response.StatusCode == 204 {
		// no content
		return "", nil
	}
	defer response.Body.Close()
	bodyBytes, err := ioutil.ReadAll(utfbom.SkipOnly(response.Body))
	if response.StatusCode >= 400 {
		log.Printf("Failed to download key %s: %d %s\n", key, response.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("failed to get location: %d", response.StatusCode)
	}

	cacheResponse := GetCacheResponse{}
	err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&cacheResponse)
	if err != nil {
		log.Println(string(bodyBytes))
		return "", err
	}
	if cacheResponse.ArchiveLocation == "" {
		log.Println(string(bodyBytes))
	}
	return cacheResponse.ArchiveLocation, nil
}

func uploadCache(w http.ResponseWriter, r *http.Request, key string) {
	cacheId, err := reserveCache(key)
	if err != nil {
		log.Printf("Failed to reserve upload for cache key %s: %v\n", key, err)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	err = uploadCacheFromReader(cacheId, r.Body)
	if err != nil {
		log.Printf("Failed to upload cache %s: %v\n", key, err)
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func uploadCacheFromReader(cacheId int, body io.Reader) error {
	resourceUrl := getCacheApiUrl(fmt.Sprintf("caches/%d", cacheId))
	readBufferSize := int(1024 * 1024)
	readBuffer := make([]byte, readBufferSize)
	bufferedBodyReader := bufio.NewReaderSize(body, readBufferSize)
	bytesUploaded := 0
	for {
		n, err := bufferedBodyReader.Read(readBuffer)

		if n > 0 {
			uploadCacheChunk(resourceUrl, readBuffer[:n], bytesUploaded)
			bytesUploaded += n
		}

		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return err
		}
	}
	return commitCache(cacheId, bytesUploaded)
}

func uploadCacheChunk(url string, data []byte, position int) error {
	request, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "actions/cache")
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/*", position, position+len(data)-1))
	request.Header.Set("Accept", "application/json;api-version=6.0-preview.1")
	request.Header.Set("Accept-Charset", "utf-8")

	response, _ := httpClient.Do(request)
	if response.StatusCode != 204 {
		defer response.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(response.Body)
		log.Printf("Failed to upload cache chunk: %s\n", string(bodyBytes))
		log.Println(string(bodyBytes))
		return fmt.Errorf("failed to upload chunk with status %d: %s", response.StatusCode, string(bodyBytes))
	}
	return nil
}

func commitCache(cacheId int, size int) error {
	url := getCacheApiUrl(fmt.Sprintf("caches/%d", cacheId))
	requestBody := fmt.Sprintf("{ \"size\": \"%d\" }", size)
	request, _ := http.NewRequest("POST", url, bytes.NewBufferString(requestBody))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "actions/cache")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json;api-version=6.0-preview.1")
	request.Header.Set("Accept-Charset", "utf-8")
	response, _ := httpClient.Do(request)
	if response.StatusCode != 204 {
		defer response.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(response.Body)
		log.Printf("Failed to commit cache %d: %s\n", cacheId, string(bodyBytes))
		return fmt.Errorf("failed to commit cache %d with status %d: %s", cacheId, response.StatusCode, string(bodyBytes))
	}
	return nil
}

func reserveCache(key string) (int, error) {
	requestUrl := getCacheApiUrl("caches")
	requestBody := fmt.Sprintf("{ \"key\": \"%s\", \"version\": \"%s\" }", key, calculateSHA256(key))
	request, _ := http.NewRequest("POST", requestUrl, bytes.NewBufferString(requestBody))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "actions/cache")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json;api-version=6.0-preview.1")
	request.Header.Set("Accept-Charset", "utf-8")

	response, err := httpClient.Do(request)
	if err != nil {
		return -1, err
	}
	defer response.Body.Close()
	bodyBytes, err := ioutil.ReadAll(utfbom.SkipOnly(response.Body))
	if response.StatusCode >= 400 {
		return -1, fmt.Errorf("failed to reserve cache: %d", response.StatusCode)
	}

	var cacheResponse ReserveCacheResponse
	err = json.Unmarshal(bodyBytes, &cacheResponse)
	if err != nil {
		return -1, err
	}
	return cacheResponse.CacheId, nil
}

func calculateSHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func getCacheApiUrl(resource string) string {
	baseUrl := strings.ReplaceAll(os.Getenv("ACTIONS_CACHE_URL"), "pipelines", "artifactcache")
	return baseUrl + "_apis/artifactcache/" + resource
}
