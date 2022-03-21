import * as core from '@actions/core';
import {exec} from "@actions/exec";

async function run() {
    const port = core.getInput('port');

    let runArguments = [
        "run", "-d", "-p", `${port}:12321`,
        "--name", "cache_proxy",
        "--env", "ACTIONS_CACHE_URL",
        "--env", "ACTIONS_RUNTIME_URL",
        "--env", "ACTIONS_RUNTIME_TOKEN",
        "ghcr.io/cirruslabs/actions-http-cache-proxy:latest"
    ];

    try {
        await exec(`"docker"`, runArguments);
    } catch (e: any) {
        core.error(e)
    }
}

run();