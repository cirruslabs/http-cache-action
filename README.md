# Action to run an HTTP Caching server

Run a local server with an API compatible with build systems like Gradle, Bazel, Buck, Pants, etc.

`/<KEY>` endpoint supports `GET`, `POST`/`PUT` and `HEAD` methods for downloading, uploading and existence checking of a cache entry with `KEY` cache key.

## Inputs

### `port`

**Optional** Port number to start the proxy on. By default, `12321` is used.

## Example usage

```yaml
uses: cirruslabs/http-cache-action@master
```

After that you can reach the HTTP Caching Proxy via `http://localhost:12321/`

### Gradle Example

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test-gradle:
    runs-on: ubuntu-latest
    name: Gradle Check
    steps:
      - uses: actions/checkout@v2
      - uses: cirruslabs/http-cache-action@master
      - uses: actions/setup-java@v1
        with:
          java-version: 13
      - run: ./gradlew check
```

Don't forget to add the following to your `settings.gradle`:

```groovy
ext.isCiServer = System.getenv().containsKey("CI")

buildCache {
  local {
    enabled = !isCiServer
  }
  remote(HttpBuildCache) {
    url = 'http://' + System.getenv().getOrDefault("CIRRUS_HTTP_CACHE_HOST", "localhost:12321") + "/"
    enabled = isCiServer
    push = true
  }
}
```

Or the following to your `settings.gradle.kts` if you are using Kotlin Script:


```kotlin
val isCiServer = System.getenv().containsKey("CI")

buildCache {
  local {
    isEnabled = !isCiServer
  }
  remote<HttpBuildCache> {
    val cacheHost = System.getenv().getOrDefault("CIRRUS_HTTP_CACHE_HOST", "localhost:12321")
    url = uri("http://$cacheHost/")
    isEnabled = isCiServer
    isPush = true
  }
}
```
