#!/bin/bash
set -e

# Check rancher/rancher releases on github
# vk3s() { curl -sH "Accept: application/vnd.github.v3+json" 'https://api.github.com/repos/k3s-io/k3s/releases' | jq -r '.[] | select (.assets[].name == "k3s") | .name' | grep -v '\-rc' | sort -r; }

# curl -sH "Accept: application/vnd.github.v3+json" 'https://api.github.com/repos/rancher/rancher/releases' | jq -r '.[] | .tag_name' | grep -v '\-rc' | sort -r #> rancher-releases.txt

# install hauler
# https://docs.hauler.dev/docs/guides-references/cluster-images
# curl -sfL https://get.hauler.dev | sudo bash # this will install hauler in /usr/local/bin/hauler

# NOTE in action one can use secrets.GITHUB_TOKEN which has 1000 requests per hour
# export GH_TOKEN=${{ secrets.GITHUB_TOKEN }}
GH_TOKEN=${GH_TOKEN}

# Get latest
LATEST_RANCHER_VERSION=$(curl -sH "Accept: application/vnd.github.v3+json" -H "Authorization: Bearer ${GH_TOKEN}" 'https://api.github.com/repos/rancher/rancher/releases' | jq -r '.[] | .tag_name'  | sort -r | head -n1)
# Get the Rancher images for airgap registry
rancher_version=$LATEST_RANCHER_VERSION
curl -L https://prime.ribs.rancher.io/rancher/${rancher_version}/rancher-images.txt -o rancher-images-${rancher_version}.txt
RANCHER_IMAGES_LIST=$(cat rancher-images-${rancher_version}.txt)

# NOTE if it is Prime alpha or rc version we have to prepend the stgregistry.suse.com in front of the images
if [[ "${rancher_version}" =~ alpha|rc ]]; then
  RANCHER_IMAGES_LIST=$(echo "${RANCHER_IMAGES_LIST}" | sed 's/^/stgregistry.suse.com\//')
else
  RANCHER_IMAGES_LIST=$(echo "${RANCHER_IMAGES_LIST}" | sed 's/^/registry.suse.com\//')
fi

RANCHER_IMAGE_LIST_MODIFIED=$(echo "${RANCHER_IMAGES_LIST}" | sed 's/^/    - name: /')

# Create Hauler manifest with Rancher images
cat << EOF > hauler-manifest-${rancher_version}.yaml
apiVersion: content.hauler.cattle.io/v1
kind: Images
metadata:
  name: hauler-cluster-images
spec:
  images:
$RANCHER_IMAGE_LIST_MODIFIED
EOF

# once done we can sync the images - this takes approx 50minutes
time hauler store sync -f hauler-manifest-${rancher_version}.yaml --store rancher-${rancher_version}-store --platform linux/amd64
# with platform it takes 20minus and it is about 32GB

# to produce zst archive
time hauler store save --filename rancher-images-${rancher_version}.tar.zst --store rancher-${rancher_version}-store --platform linux/amd64
# this takes around 2minutes and has 32GB size

#but the previous step can be probably skipped as we will serve the registry directly from the store
time hauler store serve registry --port 5001 --store rancher-${rancher_version}-store --directory rancher-${rancher_version}-registry
# this took around 2minutes to start up - but it keeps running on BACKGROUND to serve the registry so maybe a service would be better
# this would be better to do over systemd service - but it seems to be pretty trick so we can leave it like this for now and put it on backgrand maybe with nohup



# IMPROVEMENT now we could use btrfs deduplication to reduce the size of the store

# Create Hauler manifest with Rancher chart

# Create Hauler manifest with files
