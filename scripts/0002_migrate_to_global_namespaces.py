#!/bin/env/python3

# The script is intended to migrate data
# from these
# `item:global:qwer-asdf-1234asdf` - global keys
# `item:user:my:zcxv-asdf-qwer` - user specific namespaces
# to these
# `item:g:default:qwer-asdf-1234asdf` - global keys in default namespace
# `item:my:user:work:zcxv-asdf-qwer` - user specific keys in work namespace
# REDIS_PORT=6379 ./0002_migrate_to_global_namespaces.py

import json
import os
import subprocess


redis_port = os.getenv("REDIS_PORT", "6379")

keys_proc = subprocess.run(["redis-cli", "-p", redis_port, "-c", "keys", "*"], capture_output=True)
for key in keys_proc.stdout.decode('utf-8').split('\n'):
    if not key:
        continue
    if key.startswith('item:global:'):
        continue
    uid = key.split(':')[-1]
    new_key = f'item:g:default:{uid}'
    data_proc = subprocess.run(["redis-cli", "-p", redis_port, "-c", "get", key], capture_output=True)
    for raw_data in data_proc.stdout.decode('utf-8').split('\n'):
        if not raw_data:
            continue
        subprocess.run(["redis-cli", "-p", redis_port, "-c", "set", new_key, raw_data])
