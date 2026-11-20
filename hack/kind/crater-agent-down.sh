#!/usr/bin/env bash

# Copyright 2026 The Crater Project Team, RAIDS-Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

set -euo pipefail

export PATH="${HOME}/.local/bin:${PATH}"

kind delete cluster --name crater-agent-dev
