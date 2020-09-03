# Action to run an HTTP Caching server

Run a local server with an API compatioble with build systems like Gradle, Bazel, Buck, Pants, etc.

## Inputs

### `port`

**Optional** Port number to start the proxy on. By default, `12321` is used.

## Example usage

```yaml
uses: cirruslabs/http-cache-action@master
```