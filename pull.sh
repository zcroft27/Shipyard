#!/bin/bash

# Step 1: Get a bearer token
TOKEN=$(curl -s "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/ubuntu:pull" | jq -r .token)

# Step 2: Get the image manifest. In this example, an image manifest list is returned.
curl -s -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.docker.distribution.manifest.list.v2+json" \
  https://registry-1.docker.io/v2/library/ubuntu/manifests/latest \
  -o manifest-list.json

# Step 3a: Parse the `manifests[]` array to locate the digest for your target platform (e.g., `linux/amd64`).
IMAGE_MANIFEST_DIGEST=$(jq -r '.manifests[] | select(.platform.architecture == "amd64" and .platform.os == "linux") | .digest' manifest-list.json)

# Step 3b: Get the platform-specific image manifest
curl -s -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
  https://registry-1.docker.io/v2/library/ubuntu/manifests/$IMAGE_MANIFEST_DIGEST \
  -o manifest.json

# Step 4: Send a HEAD request to check if the layer blob exists
DIGEST=$(jq -r '.layers[0].digest' manifest.json)
curl -I -H "Authorization: Bearer $TOKEN" \
  https://registry-1.docker.io/v2/library/ubuntu/blobs/$DIGEST

# Step 5: Download the layer blob
curl -L -H "Authorization: Bearer $TOKEN" \
  https://registry-1.docker.io/v2/library/ubuntu/blobs/$DIGEST
