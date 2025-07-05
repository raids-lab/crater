#!/bin/bash

# Copyright 2025 RAIDS Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Function to print usage
usage() {
    echo "Usage: bash create-secret.sh -tls {xxxxx.zip} -n {namespace} {secret_name}"
    exit 1
}

# Parse command line arguments
while [ $# -gt 0 ]; do
    case "$1" in
    -tls)
        zip_file="$2"
        shift 2
        ;;
    -n)
        namespace="$2"
        shift 2
        ;;
    *)
        secret_name="$1"
        shift
        ;;
    esac
done

# Check if mandatory arguments are provided
if [ -z "$zip_file" ] || [ -z "$namespace" ] || [ -z "$secret_name" ]; then
    usage
fi

# Unzip or extract archive
if [[ "$zip_file" == *.zip ]]; then
    unzip -o "$zip_file"
elif [[ "$zip_file" == *.tgz || "$zip_file" == *.tar.gz ]]; then
    tar -xzf "$zip_file"
else
    echo "Error: Invalid file format. Please provide a zip, tgz, or tar.gz file."
    exit 1
fi

# Find the extracted directory (assumes only one new top-level directory is added)
extracted_dir=$(unzip -l "$zip_file" | awk '/\/$/ {print $4}' | head -n 1  | sed 's|/$||')

if [ -z "$extracted_dir" ] || [ ! -d "$extracted_dir" ]; then
    echo "Error: Unable to determine the extracted directory."
    exit 1
fi

# Validate required files exist
key_file="$extracted_dir/$extracted_dir.key"
cert_file="$extracted_dir/fullchain.cer"

if [ ! -f "$key_file" ] || [ ! -f "$cert_file" ]; then
    echo "Error: Required files are missing in the zip file."
    echo "Expected:"
    echo "  $key_file"
    echo "  $cert_file"
    exit 1
fi

# Create the Kubernetes TLS secret
kubectl delete secret -n "$namespace" "$secret_name" --ignore-not-found
kubectl create secret tls "$secret_name" --key "$key_file" --cert "$cert_file" -n "$namespace"

# Check if secret creation was successful
if [ $? -eq 0 ]; then
    echo "Secret '$secret_name' created successfully in namespace '$namespace'."
else
    echo "Error: Failed to create secret."
    exit 1
fi

# Clean up
rm -rf "$extracted_dir"
