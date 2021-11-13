#!/bin/env/python3

# The script is intended to migrate data from keys `item:1`, `item:2`
# to `item:global:1234-asdf-qwer-3455`
# REDIS_PORT=6379 ./0001_migrate_to_namespaces.py

import json
import os
import subprocess
import uuid


redis_port = os.getenv("REDIS_PORT", "6379")

keys_proc = subprocess.run(["redis-cli", "-p", redis_port, "-c", "keys", "*"], capture_output=True)
for key in keys_proc.stdout.decode('utf-8').split('\n'):
    if not key:
        continue
    if not key.startswith('item'):
        continue
    new_uid = str(uuid.uuid4())
    new_key = f'item:global:{new_uid}'
    data_proc = subprocess.run(["redis-cli", "-p", redis_port, "-c", "get", key], capture_output=True)
    for raw_data in data_proc.stdout.decode('utf-8').split('\n'):
        if not raw_data:
            continue
        data = json.loads(raw_data)
        del data['id']
        data['uid'] = new_uid
        subprocess.run(["redis-cli", "-p", redis_port, "-c", "set", new_key, json.dumps(data)])
