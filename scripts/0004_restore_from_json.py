#!/bin/env/python3

# This script restores data from items.json produced by
# 0003_backup_as_json.py

import json
import os
import subprocess


redis_port = os.getenv("REDIS_PORT", "6379")
f = open('items.json')
items = json.load(f)

# TODO: clear the storage before restoration?

for item in items:
    subprocess.run(["redis-cli", "-p", redis_port, "-c", "set", item["key"], item["value"]])

