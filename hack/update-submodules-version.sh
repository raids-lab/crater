#!/bin/bash
echo "Updating submodules to the latest version..."

echo "Updating crater-backend submodule..."
cd crater-backend
git fetch upstream
git rebase upstream/main
cd ..

echo "Updating crater-frontend submodule..."
cd crater-frontend
git fetch upstream
git rebase upstream/main
cd ..

echo "Updating storage-server submodule..."
cd storage-server
git fetch upstream
git rebase upstream/main
cd ..