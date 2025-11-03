#!/bin/sh

dry_run=false
size_limit=0

# Show usage
show_usage() {
    echo "Usage: $0 [--namespace NAMESPACE] [--pod-name PODNAME] [--container-name CONTAINERNAME] [--image-url IMAGE_URL] [--size-limit SIZE_LIMIT_IN_GiB] [--dry-run]"
    exit 1
}

# Delete the committed image
delete_image() {
    echo "[Cleanup] Deleting image $imageurl"
    nerdctl --namespace=k8s.io rmi "$imageurl" 2>&1
    if [ $? -eq 0 ]; then
        echo "Image $imageurl deleted successfully."
    else
        echo "Failed to delete image $imageurl."
    fi
}

# Trap for cleanup on exit or interruption
trap delete_image EXIT

# Parse command line arguments
while [ $# -gt 0 ]; do
    case "$1" in
        --namespace)
            namespace="$2"
            shift 2
            ;;
        --pod-name)
            podname="$2"
            shift 2
            ;;
        --container-name)
            containername="$2"
            shift 2
            ;;
        --image-url)
            imageurl="$2"
            shift 2
            ;;
        --size-limit)
            size_limit="$2"
            shift 2
            ;;
        --dry-run)
            dry_run=true
            shift
            ;;
        *)
            echo "Unknown parameter: $1"
            show_usage
            ;;
    esac
done

# Check required parameters
if [ -z "$namespace" ] || [ -z "$podname" ] || [ -z "$containername" ] || [ -z "$imageurl" ]; then
    echo "Missing required parameters."
    show_usage
fi

# Generate nerctl name
name="k8s://$namespace/$podname/$containername"

echo "[Step 1] Getting container ID for $name"
id=$(nerdctl --namespace=k8s.io ps -f "name=$name" -q 2>&1)
if [ $? -ne 0 ]; then
    echo "Error running first command: $id"
    exit 1
fi

if [ -z "$id" ]; then
    echo "Container with name $name not found."
    exit 1
fi

echo "Container ID: $id"

echo "[Step 2] Checking the size of the container image..."
size_info=$(nerdctl --namespace=k8s.io ps -s -f "id=$id" 2>&1)
if [ $? -ne 0 ]; then
    echo "Error running size check command: $size_info"
    exit 1
fi

size_line=$(echo "$size_info" | grep "$id")
size_value=$(echo "$size_line" | awk -F '[()]' '{print $2}' | awk '{print $2}')
size_unit=$(echo "$size_line" | awk -F '[()]' '{print $2}' | awk '{print $3}')

echo "Container Size: $size_value $size_unit"

# Convert size to GiB if it is in MiB
if [ "$size_limit" != "0" ] && { [ "$size_unit" = "MiB" ] || [ "$size_unit" = "GiB" ]; }; then
    if [ "$size_unit" = "MiB" ]; then
        size_in_gib=$(echo "scale=2; $size_value / 1024" | bc)
    else
        size_in_gib=$size_value
    fi
    # Compare the size with the limit
    comparison=$(echo "$size_in_gib > $size_limit" | bc -l)
    if [ "$comparison" -eq 1 ]; then
        echo "Error: Container size $size_in_gib GiB exceeds the limit of $size_limit GiB."
        exit 1
    fi
fi

if [ "$dry_run" = true ]; then
    echo "Dry-run mode enabled. Skipping commit and push steps."
    exit 0
fi

echo "[Step 3] Commit and push the container to the registry..."
commit_result=$(nerdctl --namespace=k8s.io commit "$id" "$imageurl" 2>&1)
if [ $? -ne 0 ]; then
    echo "Error running commit command: $commit_result"
    exit 1
fi

echo "Commit result: $commit_result"

echo "[Step 4] Pushing the image to the registry..."
max_retries=5
retry_delay=2
attempt=1

while [ $attempt -le $max_retries ]; do
    push_result=$(nerdctl -n k8s.io push "$imageurl" 2>&1)
    if [ $? -eq 0 ]; then
        echo "Push successful."
        break
    fi
    echo "Push attempt $attempt failed: $push_result"
    if [ $attempt -eq $max_retries ]; then
        echo "Maximum retries reached. Exiting with failure."
        exit 1
    fi
    echo "Retrying in $retry_delay seconds..."
    sleep $retry_delay
    retry_delay=$((retry_delay * 2))
    attempt=$((attempt + 1))
done

echo "All commands executed successfully."
