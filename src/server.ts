import * as http from "http";
import {IncomingMessage, ServerResponse} from "http";
import {getCacheApiUrl, getCacheEntry} from "./vendor/internal/cacheHttpClient";

async function checkCache(cacheKey: string, res: ServerResponse) {
    let cacheEntry = await getCacheEntry([cacheKey], []);
    if (cacheEntry == null) {
        res.write(404)
    } else {
        res.write(200)
    }
    res.end()
}

async function downloadCache(cacheKey: string, res: ServerResponse) {
    let cacheEntry = await getCacheEntry([cacheKey], []);
    if (cacheEntry == null) {
        res.write(404)
    } else {
        res.writeHead(302, {
            'Location': cacheEntry.archiveLocation
        })
    }
    res.end()
}

async function uploadCache(cacheKey: string, res: ServerResponse) {
    let uploadURL = await getCacheApiUrl(`caches/${cacheKey}`)
    res.writeHead(302, {
        'Location': uploadURL
    })
    res.end()
}

export function createCachingServer() {
    const requestListener = function (req: IncomingMessage, res: ServerResponse) {
        let cacheKey = (req.url || "").substring(1); // remove trailing slash
        switch (req.method) {
            case "HEAD":
                return checkCache(cacheKey, res)
            case "POST":
                return uploadCache(cacheKey, res)
            default:
                return downloadCache(cacheKey, res)
        }
    }

    return http.createServer(requestListener);
}

createCachingServer().listen(12321);