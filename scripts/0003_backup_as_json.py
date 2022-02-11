#!/bin/env/python3

# This script makes a backup of all keys 
# and their data in json format.
# It creates items.json with the following format:
# [{"key": "item:g:default:123-asdf-3546-zxcv", "value": "some-utf-encoded-data"]

import json
import os
import subprocess


redis_port = os.getenv("REDIS_PORT", "6379")
result = []

keys_proc = subprocess.run(["redis-cli", "-p", redis_port, "-c", "keys", "*"], capture_output=True)
for key in keys_proc.stdout.decode("utf-8").split("\n"):
    data_proc = subprocess.run(["redis-cli", "-p", redis_port, "-c", "get", key], capture_output=True)
    for raw_data in data_proc.stdout.decode("utf-8").split("\n"):
        if not raw_data:
            continue
        result.append({"key": key, "value": raw_data})

with open("items.json", "w") as f:
    json.dump(result, f, ensure_ascii=False)

