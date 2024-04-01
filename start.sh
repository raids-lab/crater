#!/bin/bash

git pull
pnpm install
pnpm serve --build --port 8888 --host 0.0.0.0