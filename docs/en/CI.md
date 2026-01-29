[English](CI.md) | [简体中文](../zh-CN/CI.md)

# CI Documentation

This document describes the design and implementation of continuous integration (CI) for the Crater project.

Before open-sourcing to GitHub, this project was hosted on a GitLab instance deployed internally by the lab, using GitLab Pipeline for CI/CD. After migrating to GitHub, we removed the continuous deployment (CD) part, keeping only continuous integration (CI), and implemented it using GitHub Actions workflows.

---

## Overview

Crater's CI process is built on GitHub Actions, primarily serving code quality assurance and artifact publishing. Unlike traditional CI/CD processes, we only retain the continuous integration (CI) part, leaving continuous deployment (CD) to users to handle according to their own environments. This design ensures code quality and standardized build artifacts while giving users flexibility in deployment.

The inputs of the CI process mainly include source code, Dockerfiles, documentation files, and Helm Chart configurations in the repository; the outputs are build artifacts, including multi-platform Docker images, static websites, and Helm Chart packages. All Docker images and Helm Charts are stored in GitHub Container Registry (GHCR), and the documentation website is deployed on GitHub Pages. Users can access these artifacts through the corresponding addresses.

### Goals

Crater's CI process aims to ensure code quality, standardize build artifacts, and provide users with ready-to-use images and Charts through automation. It ensures code quality through PR checks, reduces manual intervention through automated builds and publishing, meets different environment needs through multi-platform support, and ensures artifact traceability and storage efficiency through version management and cleanup strategies.

### Technology Stack

Crater's CI process is built on GitHub Actions, Docker Buildx, and GitHub Container Registry (GHCR). GitHub Actions provides deeply integrated CI capabilities with the repository without requiring additional third-party service configuration; Docker Buildx enables cross-platform builds through QEMU emulation, building both amd64 and arm64 architecture images simultaneously to meet different hardware environment needs; GHCR serves as a unified storage for container images and Helm Charts, integrates with GitHub's permission system, supports OCI standards, and enables automated authentication through `GITHUB_TOKEN`.

### CI Process Categories

Crater's CI process is divided into four main categories based on different build targets, each with independent trigger conditions and build processes:

- **Frontend & Backend** is the core of the CI process, responsible for code quality checks and image build publishing for application services (Backend, Frontend, Storage). It adopts a two-stage design: the PR check stage performs code style checks (Lint) and build verification to ensure code quality; the build and publish stage builds multi-platform images and pushes them to GHCR after code merge, while managing storage space through automatic cleanup strategies.

- **Dependency Images** are responsible for building and pushing Docker images related to build tools (buildx-client, envd-client, nerdctl-client), providing necessary runtime environments for application builds. These images also support multi-platform builds and are managed uniformly through GHCR.

- **Documentation Website** handles documentation building, quality checks, and automated deployment. The PR check stage verifies successful documentation builds and checks image format standards; the deployment stage automatically builds Next.js websites and deploys them to GitHub Pages; meanwhile, it ensures documentation quality and multi-language synchronization through automatic correction and translation mechanisms.

- **Helm Chart** is responsible for Chart validation and publishing. The PR check stage validates Chart syntax, templates, and version number updates; the publish stage packages Charts and pushes them to the GHCR OCI repository, providing users with standardized deployment solutions.

---

## Frontend & Backend

This section introduces the CI configuration for Crater's frontend and backend.

It should be noted that the storage service (storage-server) is located in the `storage` directory under the main repository, and its CI configuration is also included in this section. The storage service adopts the same CI pattern as the backend and is planned to be merged into the backend in the future.

### Overview

The CI process for frontend and backend adopts a two-stage design: PR check stage and build & publish stage. Inputs are source code (Go code or frontend resources), outputs are multi-platform Docker images (linux/amd64 and linux/arm64), and artifacts are stored in GHCR repositories `ghcr.io/raids-lab/crater-backend`, `ghcr.io/raids-lab/crater-frontend`, and `ghcr.io/raids-lab/storage-server`.

The PR check stage executes before code merge, performing code style checks (Lint) and build verification, building only a single platform to save time, without building and pushing images. The build & publish stage executes after code merge or when creating version tags, building multi-platform images and pushing them to GHCR, while automatically cleaning up old images to control storage space.

The three components (Backend, Frontend, and Storage) adopt the same two-stage CI pattern, but the build processes differ: Backend and Storage compile to generate binary files and then package them into images, while Frontend builds static resources and provides service images through web servers.

The following sections mainly introduce the detailed processes and mechanisms of the build & publish stage, and the PR check stage will be briefly explained at the end.

### Trigger & Version Management

The build & publish stage has two trigger methods: code push to the main branch or creating version tags. Taking Backend's workflow configuration as an example:

```yaml
on:
  push:
    branches: [main]
    paths:
      - "backend/**"
      - ".github/workflows/backend-build.yml"
  tags:
    - "v*.*.*"
```

In the `push` event, `branches: [main]` specifies listening only to pushes to the main branch, and the `paths` parameter further filters paths, triggering builds only when files under the `backend/**` directory or the workflow file itself change. In other words, if a commit only modifies frontend code without modifying backend code, only the frontend image will be rebuilt.

The `tags` event is configured as `v*.*.*`, matching all semantic version format tags (e.g., v1.2.3). Tag triggers do not use path filtering, and will trigger builds for all components regardless of paths. This is because version releases need to ensure all components are built based on the same code version, guaranteeing version consistency and completeness. Even if a component has no code changes in this release, it will be rebuilt and tagged with the corresponding version tag.

### Version Injection

During the build process, version information is injected into build artifacts for runtime querying and issue tracking. Version information includes version number (AppVersion), commit SHA (CommitSHA), build type (BuildType), and build time (BuildTime).

The version information generation logic is implemented through scripts in the workflow. Taking Backend as an example:

```yaml
- name: Set version variables
  id: set-version
  run: |
    COMMIT_SHA="${{ github.sha }}"
    SHORT_SHA="${COMMIT_SHA:0:7}"

    # Check if triggered by tag
    if [[ "${{ github.ref_type }}" == "tag" ]]; then
      APP_VERSION="${{ github.ref_name }}"
      BUILD_TYPE="release"
    else
      APP_VERSION="$SHORT_SHA"
      BUILD_TYPE="development"
    fi

    echo "app_version=$APP_VERSION" >> $GITHUB_OUTPUT
    echo "commit_sha=$COMMIT_SHA" >> $GITHUB_OUTPUT
    echo "build_type=$BUILD_TYPE" >> $GITHUB_OUTPUT
    echo "build_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $GITHUB_OUTPUT
```

The script checks `github.ref_type` to determine the trigger type: when triggered by a tag, it uses the tag name as the version number and sets BUILD_TYPE to "release"; when triggered by a branch, it uses the first 7 characters of the commit SHA as the version number and sets BUILD_TYPE to "development".

For Go projects like Backend and Storage, version information is injected into binary files through `ldflags`:

```yaml
go build -ldflags="-X main.AppVersion=${{ steps.set-version.outputs.app_version }} \
  -X main.CommitSHA=${{ steps.set-version.outputs.commit_sha }} \
  -X main.BuildType=${{ steps.set-version.outputs.build_type }} \
  -X main.BuildTime=${{ steps.set-version.outputs.build_time }} -w -s" \
  -o bin/linux_amd64/controller cmd/crater/main.go
```

The `-X` parameter is used to set package variable values, compiling version information into binary files, which can be queried through program interfaces at runtime.

For Frontend projects, version information is injected into the build process through environment variables:

```yaml
echo "VITE_APP_VERSION=$APP_VERSION" >> $GITHUB_ENV
echo "VITE_APP_COMMIT_SHA=$COMMIT_SHA" >> $GITHUB_ENV
echo "VITE_APP_BUILD_TYPE=$BUILD_TYPE" >> $GITHUB_ENV
echo "VITE_APP_BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $GITHUB_ENV
```

These environment variables are replaced into frontend code by Vite during build, and users can view version information in the frontend interface.

Generally speaking, the latest versions of frontend and backend should be consistent, but if a modification only changes one of them, it will lead to inconsistent latest versions between frontend and backend.

When deploying with images, users are advised to synchronize frontend and backend versions, as the project does not currently guarantee compatibility between different frontend and backend versions.

### Cross-Platform Build

The build & publish stage supports building images for both linux/amd64 and linux/arm64 platforms simultaneously to meet different hardware architecture needs. Cross-platform building is divided into two stages: first compiling binary files for different platforms, then using Docker Buildx to build multi-platform images.

For projects that require compilation like Backend and Storage, GitHub Actions' matrix strategy is used to build binary files for different platforms in parallel:

```yaml
build_backend:
  strategy:
    matrix:
      platform:
        - goos: linux
          goarch: amd64
          image_platform: linux/amd64
        - goos: linux
          goarch: arm64
          image_platform: linux/arm64
  steps:
    - name: Build backend binaries
      run: |
        go build -ldflags="..." -o bin/${{ matrix.platform.image_platform }}/controller cmd/crater/main.go
      env:
        GOOS: ${{ matrix.platform.goos }}
        GOARCH: ${{ matrix.platform.goarch }}
```

By setting `GOOS` and `GOARCH` environment variables, the Go compiler generates corresponding binary files for the target platform. After building, binary files for each platform are uploaded as build artifacts for subsequent image building.

The image building stage uses Docker Buildx and QEMU to achieve cross-platform image building:

```yaml
- name: Set up QEMU
  uses: docker/setup-qemu-action@v3

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Build and push multi-platform image
  uses: docker/build-push-action@v6
  with:
    context: ./backend
    file: ./backend/Dockerfile
    platforms: linux/amd64,linux/arm64
    push: true
```

QEMU enables cross-platform building through CPU emulation, allowing building arm64 images on amd64 architecture build machines. Docker Buildx is Docker's extended build tool that supports multi-platform builds and image manifest management. After building, Docker Buildx creates a manifest containing multiple platform images, and users will automatically select the matching platform version when pulling images.

For projects like Frontend that don't require compilation, build artifacts are static resource files (HTML, CSS, JavaScript, etc.), not binary executables, so cross-compilation is not needed. The frontend build process doesn't depend on the target platform architecture; it only needs to package build artifacts into different platform base images. Docker Buildx will select appropriate base images (such as Nginx) for the target platform to serve static files.

### Image Tagging Strategy

After building, images are tagged with multiple labels to help users select appropriate versions for different use cases. Tag generation uses `docker/metadata-action` with the following configuration:

```yaml
- name: Docker meta
  id: meta
  uses: docker/metadata-action@v5
  with:
    images: ${{ env.REGISTRY }}/${{ env.REPOSITORY }}/${{ env.IMAGE_NAME }}
    tags: |
      type=ref,event=branch
      type=semver,pattern={{version}}
      type=semver,pattern={{major}}.{{minor}}
      type=semver,pattern={{major}}
      type=raw,value=latest,enable={{is_default_branch}}
      type=sha
```

The `images` parameter specifies the base name of the image, composed of `${{ env.REGISTRY }}`, `${{ env.REPOSITORY }}`, and `${{ env.IMAGE_NAME }}`. The complete image address will be detailed in the next section "Image Push & Cleanup".

Each line under the `tags` parameter is an independent tag generation rule, and these rules generate corresponding tags in parallel based on trigger conditions. For example, when creating version tag `v1.2.3`, multiple tags are generated simultaneously: `v1.2.3`, `1.2`, `1`, and SHA tag; when pushing to the main branch, `main`, `latest`, and SHA tags are generated.

Tag rule parameter descriptions:

- **`type=ref,event=branch`**: `type=ref` means generating tags based on Git references, `event=branch` specifies that it only takes effect on branch push events. When pushing to a branch, the branch name is used as the tag.
- **Semantic version tags** (`type=semver`): Generate tags based on semantic versions, only taking effect when creating version tags. Includes three patterns:
  - `pattern={{version}}`: Uses the full version number (e.g., `v1.2.3`)
  - `pattern={{major}}.{{minor}}`: Uses major and minor version numbers (e.g., `1.2`). When creating `v1.2.3`, it generates the `1.2` tag, which points to the latest version in that version series
  - `pattern={{major}}`: Uses the major version number (e.g., `1`). When creating `v1.2.3`, it generates the `1` tag, which points to the latest version in that major version series
- **`type=raw,value=latest,enable={{is_default_branch}}`**: `type=raw` means using raw values as tags, `value=latest` specifies the tag value as `latest`, `enable={{is_default_branch}}` means this tag is only generated on the default branch (main).
- **`type=sha`**: Generates tags based on commit SHA, format is `sha-<first 7 chars of SHA>`. All builds generate this tag for precise build source tracking.

### Image Push & Cleanup

After image building is complete, images need to be pushed to the image registry and old images cleaned up to control storage space. Image pushing uses GHCR (GitHub Container Registry) as the repository. The complete image address format is `${{ env.REGISTRY }}/${{ env.REPOSITORY }}/${{ env.IMAGE_NAME }}`, i.e., `ghcr.io/raids-lab/crater-backend`, `ghcr.io/raids-lab/crater-frontend`, and `ghcr.io/raids-lab/storage-server`.

Before pushing, you need to log in to GHCR with the following configuration:

```yaml
- name: Login to GHCR
  uses: docker/login-action@v3
  with:
    registry: ${{ env.REGISTRY }}
    username: ${{ github.repository_owner }}
    password: ${{ secrets.GITHUB_TOKEN }}
```

`GITHUB_TOKEN` is used as the authentication credential. This token is automatically provided by GitHub Actions and requires no additional configuration. `github.repository_owner` is the repository owner (organization or username), which is `raids-lab` for this project.

Image building and pushing configuration:

```yaml
- name: Build and push multi-platform image
  uses: docker/build-push-action@v6
  with:
    context: ./backend
    file: ./backend/Dockerfile
    platforms: linux/amd64,linux/arm64
    push: true
    tags: ${{ steps.meta.outputs.tags }}
```

The `tags` parameter uses the tag list generated by `docker/metadata-action` in the previous section. After building is complete, images are pushed to GHCR with all generated tags. Users can pull images via `docker pull ghcr.io/raids-lab/crater-backend:<tag>`. Pushed images can be viewed in the GitHub repository's Packages page, including all tags and version information.

After building, old images are automatically cleaned up with the following configuration:

```yaml
- uses: quartx-analytics/ghcr-cleaner@v1
  with:
    owner-type: org
    token: ${{ secrets.PAT_TOKEN }}
    repository-owner: ${{ github.repository_owner }}
    package-name: crater-backend
    delete-untagged: true
    keep-at-most: 2
    skip-tags: v*
```

Cleanup rule descriptions:

- `delete-untagged: true`: Deletes untagged image layers (dangling images), which are intermediate layers generated during the build process and no longer referenced by any tags.
- `keep-at-most: 2`: Keeps at most 2 untagged images per package. Older images beyond this count are deleted.
- `skip-tags: v*`: Skips tags starting with `v` (version tags), protecting all version images from deletion, ensuring users can access historical versions.

### PR Check

In addition to the build & publish stage, CI also sets up a check process for Pull Requests to prevent bad code from entering the main branch. PR Check's trigger mechanism is consistent with the build & publish stage, using the same path filtering. Only when relevant code or workflow files change will the corresponding component's check be triggered.

PR Check includes two stages: Lint Check and Build Check, using the same process as the build workflow for checking. Unlike the build & publish stage, PR Check only builds a single platform (linux/amd64) to save build time and does not push images, only verifying that the build succeeds.

However, it should be noted that under the current workflow configuration, we cannot require these PR Checks to pass in GitHub branch protection rules *Require status checks to pass*. Because workflows not triggered by paths are not considered passed, they remain in Pending status, which will permanently block PR merges.

---

## Dependency Images

Dependency images include three Docker images related to build tools: `buildx-client`, `envd-client`, and `nerdctl-client`. `buildx-client` and `envd-client` are used to support the platform's image building functionality, while `nerdctl-client` is used to snapshot (commit) running containers into images, supporting container snapshotting for Jupyter and Custom job types.

Although these images serve as dependencies for backend functionality, they are explained separately due to significantly different build processes from the backend.

### Overview

The CI process for dependency images adopts a change detection mechanism, triggering builds only when a Dockerfile or related files for a specific image change. Inputs are Dockerfiles and related files (located in the `hack/depend-image-dockerfile/` directory), outputs are multi-platform Docker images, and artifacts are stored in GHCR repositories `ghcr.io/raids-lab/buildx-client`, `ghcr.io/raids-lab/envd-client`, and `ghcr.io/raids-lab/nerdctl-client`. After building, old images are automatically cleaned up to control storage space. Each image's build process is independent, including multi-platform image building, tag generation, and pushing.

### Trigger

Dependency image builds are only triggered when code is pushed to the main branch, listening only to changes in the `hack/depend-image-dockerfile/**` directory. Unlike frontend and backend, dependency images do not support tag triggers because dependency image versions are fixed by environment variables (such as `BUILDX_VERSION`, `ENVD_VERSION`, `NERDCTL_VERSION`) and do not need version management through tags.

The workflow trigger configuration:

```yaml
on:
  push:
    branches: [main]
    paths:
      - ".github/workflows/depend-build.yml"
      - "hack/depend-image-dockerfile/**"
```

The workflow uses the `detect-changes` job to detect which images need to be built, checking changes in each image directory through `dorny/paths-filter`. Only when a Dockerfile or related files for a specific image change will the corresponding build job execute, avoiding unnecessary build overhead. Change detection configuration:

```yaml
detect-changes:
  runs-on: ubuntu-latest
  outputs:
    buildx-client: ${{ steps.changes.outputs.buildx-client }}
    envd-client: ${{ steps.changes.outputs.envd-client }}
    nerdctl-client: ${{ steps.changes.outputs.nerdctl-client }}
  steps:
    - name: Detect changes
      uses: dorny/paths-filter@v3
      id: changes
      with:
        filters: |
          buildx-client:
            - 'hack/depend-image-dockerfile/buildx-client/**'
          envd-client:
            - 'hack/depend-image-dockerfile/envd-client/**'
          nerdctl-client:
            - 'hack/depend-image-dockerfile/nerdctl-client/**'
```

Each build job determines whether to execute through `needs: detect-changes` and conditional judgment `if: needs.detect-changes.outputs.buildx-client == 'true'`.

### Image Tagging Strategy

The image tagging strategy is similar to frontend and backend, but version tags use fixed version numbers defined in environment variables rather than parsing from Git tags. Tag generation configuration:

```yaml
- name: Docker meta for buildx-client
  id: meta
  uses: docker/metadata-action@v5
  with:
    images: ${{ env.REGISTRY }}/${{ env.REPOSITORY }}/${{ env.BUILDX_CLIENT }}
    tags: |
      type=ref,event=branch
      type=sha
      type=raw,value=latest,enable={{is_default_branch}}
      type=raw,value=${{ env.BUILDX_VERSION }}
```

The `images` parameter is composed of `${{ env.REGISTRY }}` (`ghcr.io`), `${{ env.REPOSITORY }}` (`raids-lab`), and the image name (e.g., `buildx-client`). The complete image address will be explained in the next section.

Tag rule descriptions:

- **`type=ref,event=branch`**: Generates tags based on branch names. When pushing to the main branch, generates the `main` tag.
- **`type=sha`**: Generates tags based on commit SHA, format is `sha-<first 7 chars of SHA>`. All builds generate this tag.
- **`type=raw,value=latest,enable={{is_default_branch}}`**: Only generates the `latest` tag on the default branch (main).
- **`type=raw,value=${{ env.BUILDX_VERSION }}`**: Uses the version number defined in environment variables (e.g., `v0.25.0`) as the tag. This is a tag type unique to dependency images, used to mark the fixed version of dependency tools.

Each image has corresponding version environment variables: `BUILDX_VERSION`, `ENVD_VERSION`, and `NERDCTL_VERSION`. These version numbers are defined in the workflow's environment variables and match the release versions of the dependency tools.

### Image Build

Dependency image builds use Docker Buildx for multi-platform building, similar to frontend and backend, achieving cross-platform support through QEMU emulation. Different images have different platform support: `buildx-client` and `nerdctl-client` support both `linux/amd64` and `linux/arm64` platforms, while `envd-client` only supports the `linux/amd64` platform.

Unlike frontend and backend, dependency image builds are simpler: directly building using Dockerfiles without first compiling source code. Dependency image Dockerfiles are located in the `hack/depend-image-dockerfile/` directory, with each image having its own Dockerfile for installing and configuring corresponding tools (such as `buildx`, `envd`, `nerdctl`).

Image build configuration (using `buildx-client` as an example):

```yaml
- name: Set up QEMU
  uses: docker/setup-qemu-action@v3

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Build and push buildx-client image
  uses: docker/build-push-action@v6
  with:
    context: hack/depend-image-dockerfile/buildx-client
    file: hack/depend-image-dockerfile/buildx-client/Dockerfile
    platforms: linux/amd64,linux/arm64
    push: true
    tags: ${{ steps.meta.outputs.tags }}
```

The build context is set to each image's Dockerfile directory, and the target platforms are specified through the `platforms` parameter.

The `tags` parameter uses the tag list generated by `docker/metadata-action` in the previous section (`${{ steps.meta.outputs.tags }}`) to tag the built images.

Although building and pushing are completed in the same step (`docker/build-push-action@v6`), they are logically separate. This section mainly introduces the build process, and push-related content will be explained in the next section.

### Image Push & Cleanup

The push mechanism for dependency images is the same as frontend and backend, using GHCR as the repository. Image addresses are `ghcr.io/raids-lab/buildx-client`, `ghcr.io/raids-lab/envd-client`, and `ghcr.io/raids-lab/nerdctl-client`. Image pushing is completed in the `docker/build-push-action@v6` step from the previous section (enabled through the `push: true` parameter). After building, images are automatically pushed to GHCR with all generated tags. Pushed images can be viewed in the GitHub repository's Packages page.

Image cleanup uses `ghcr-cleaner` with configuration similar to frontend and backend, but with different parameters:

```yaml
- uses: quartx-analytics/ghcr-cleaner@v1
  with:
    package-name: ${{ env.BUILDX_CLIENT }}
    delete-untagged: true
    keep-at-most: 5
    skip-tags: latest
```

Main differences:

- `keep-at-most: 5`: Compared to frontend and backend (keeping 2), dependency images keep more untagged images because dependency images have relatively lower update frequency.
- `skip-tags: latest`: Skips the `latest` tag, protecting the latest version image, while frontend and backend use `skip-tags: v*` to protect all version tags.

All three dependency images execute the same cleanup steps, cleaning their respective image repositories.

---

## Documentation Website

The documentation website is built with Next.js and deployed on GitHub Pages, containing tutorials for deploying and using Crater.

### Overview

The documentation website's CI/CD process includes both CI (Continuous Integration) and CD (Continuous Deployment) parts. The CI part includes PR checks, automatic correction, and automatic translation: the PR check stage executes before code merge, performing build verification and image format checks; the automatic correction stage automatically corrects documentation format during PRs; the automatic translation stage automatically translates documentation and creates translation PRs after code merge. The CD part includes documentation deployment: automatically building Next.js websites and deploying them to GitHub Pages after code merge. Inputs are documentation source code (Markdown, MDX, configuration files, etc.), and outputs are static websites deployed on GitHub Pages.

Unlike frontend/backend and dependency images, the documentation website includes not only CI (build and verification) but also CD (automatic deployment), achieving a fully automated process from code changes to production environment.

Taking creating a PR that updates the documentation website as an example, the execution flow is as follows:

1. **When PR is created**: When the PR modifies files under `website/src/**` or `website/content/**` directories, two workflows are triggered simultaneously:
   - **PR Check** (`docs-build.yml`): Builds the Next.js website to verify successful build and checks whether newly added or modified images are in WebP format
   - **Automatic Correction** (`docs-autocorrect.yml`): Automatically corrects documentation format. For internal PRs, it directly commits corrections; for Fork PRs, it reports issues in PR comments

2. **After PR merge**: When the PR is merged to the main branch, two workflows are triggered:
   - **Documentation Deployment** (`docs-deploy.yml`): Builds the Next.js website including Pagefind search index and deploys it to GitHub Pages, automatically updating the documentation website
   - **Automatic Translation** (`docs-autotranslate.yml`): Intelligently filters files that need translation based on file change types and PR labels, runs translation scripts, and creates PRs containing translation results (branch name: `feature/auto-translate-{run_id}`)

3. **After translation PR is created**: The PR created by automatic translation triggers PR check and automatic correction workflows, but since the translation PR's commit message starts with `chore(i18n):`, the automatic translation workflow's loop prevention mechanism skips this PR to avoid infinite loops.

4. **After translation PR merge**: After the translation PR is merged to the main branch, the documentation deployment workflow is triggered to update the documentation website. Since the commit message starts with `chore(i18n):`, it will not trigger the automatic translation workflow again.

It should be noted that the current automatic translation process is still not very reliable and needs improvement.

### PR Check

PR check is triggered when creating a Pull Request, listening to changes in `website/src/**`, `website/content/**`, `website/package.json`, and `website/pnpm-lock.yaml`. The check process includes two steps: build verification and image format check.

Build verification uses Next.js to build the website, ensuring the documentation can be built normally:

```yaml
- name: Build website
  run: pnpm exec next build
```

Image format check ensures that newly added or modified images in the PR use WebP format. The check scope includes `src/` and `content/` directories. If non-WebP format images (such as PNG, JPG, JPEG, GIF, BMP) are found, the PR check fails, prompting users to convert images to WebP format.

### Documentation Build & Deployment

Documentation deployment is triggered when code is pushed to the main branch, listening to changes in `website/src/**`, `website/content/**`, and `website/package.json`. The deployment process includes two stages: build and deployment.

The build stage uses Next.js to build the website,  and creates a `.nojekyll` file to prevent GitHub Pages from using Jekyll processing:

```yaml
- name: Build website
  run: pnpm build

- name: Create .nojekyll file
  run: touch ./out/.nojekyll
```

The deployment stage uses `actions/deploy-pages@v4` to deploy build artifacts to GitHub Pages, requiring `pages: write` and `id-token: write` permissions, and uses the `github-pages` environment:

```yaml
- name: Deploy to GitHub Pages
  uses: actions/deploy-pages@v4
```

After deployment, the documentation website is automatically updated, and users can access the latest version of the documentation through the GitHub Pages URL.

### Automatic Correction

Automatic correction uses the `autocorrect` tool to automatically correct documentation format, triggered when creating Pull Requests, listening to changes in `website/src/**` and `website/content/**`. Different processing strategies are adopted based on PR source.

For internal PRs (from the same repository), automatic correction directly fixes files and commits changes:

```yaml
- name: AutoCorrect and Fix (for internal PRs)
  uses: huacnlee/autocorrect-action@v2
  with:
    args: --fix ${{ steps.internal_files.outputs.files }}

- name: Commit changes (for internal PRs)
  uses: stefanzweifel/git-auto-commit-action@v5
```

For Fork PRs (from external repositories), due to permission limitations, changes cannot be directly committed. Therefore, Reviewdog is used to report format issues in PR comments, and contributors fix them themselves.

Automatic correction excludes `*.*.mdx` files (multi-language files) to avoid affecting translation files.

### Automatic Translation

Automatic translation is triggered when code is pushed to the main branch, listening to changes in `website/content/docs/**`, `website/messages/**`, and `website/src/i18n/config.ts`. The translation process uses GitHub App for authentication, determines files that need translation through intelligent filtering mechanisms, then runs Python scripts for translation, and finally creates PRs containing translation results.

Loop prevention mechanism avoids infinite loops by checking commit messages: skipping commits starting with `chore(i18n):` or containing `from feature/auto-translate-`.

Intelligent filtering mechanism is based on file change types and PR labels:

- **New files**: Always translated
- **Modified files**: Check the source PR's labels. Only files modified by PRs with the `run-translation` label are translated
- **Skip condition**: If the PR has the `no-translation` label, the entire workflow is skipped

After translation is complete, translation PRs are automatically created with branch name `feature/auto-translate-{run_id}`, PR title `🌐 [Auto-Translate] 同步多语言文件`, and labels `i18n, automated`. After PR creation, branches are automatically deleted to avoid branch accumulation.

---

## Helm Chart

Helm Chart is used to deploy the Crater platform to Kubernetes clusters, providing one-click deployment and configuration management capabilities. Helm Chart's CI process includes two stages: Chart validation and Chart publishing, ensuring Chart quality and automated publishing.

### Overview

Helm Chart's CI process adopts a two-stage design: the Chart validation stage executes during PRs, performing syntax validation, template validation, and version number checks; the Chart publishing stage executes when code is merged to the main branch or when Releases are created, packaging Charts and pushing them to the GHCR OCI repository. Inputs are Chart source code (located in the `charts/crater/` directory), outputs are packaged Helm Charts (`.tgz` files), and artifacts are stored in GHCR's `ghcr.io/raids-lab/crater` OCI repository.

Chart validation ensures Chart correctness and completeness, including syntax checks, template rendering validation, and version number update checks. Chart publishing packages validated Charts and pushes them to GHCR. Users can install Charts via `helm install crater oci://ghcr.io/raids-lab/crater --version <version>`.

### Chart Validation

Chart validation is triggered when creating Pull Requests, listening to changes in the `charts/**` directory. The validation process includes four steps: syntax validation, template validation, version number check, and packaging test.

Syntax validation uses `helm lint` to check Chart syntax, dependencies, templates, etc.:

```yaml
- name: Validate Chart Syntax
  run: |
    cd charts
    helm lint crater/
    helm template crater crater/ --dry-run
```

`helm lint` checks Chart syntax errors, dependencies, and best practices; `helm template --dry-run` verifies whether templates can render correctly, ensuring template syntax is correct and all required values are provided.

Version number check ensures that each PR updates the Chart version number by comparing version numbers between the current branch and the base branch:

```bash
CURRENT_VERSION=$(helm show chart charts/crater/ | grep '^version:' | awk '{print $2}')
BASE_VERSION=$(git show "origin/$BASE_BRANCH:charts/crater/Chart.yaml" | grep '^version:' | awk '{print $2}')

if [ "$CURRENT_VERSION" = "$BASE_VERSION" ]; then
  echo "⚠️  Chart version has not been updated"
  exit 1
fi
```

If the version number is not updated, the PR check fails, prompting users to update the version number and follow semantic versioning standards.

Packaging test uses `helm package` to test whether Charts can be packaged normally. After packaging, packaged files are deleted, used only for verification:

```yaml
- name: Package Chart (Test)
  run: |
    cd charts
    helm lint crater/
    helm package crater/
    rm -f crater-*.tgz
```

### Chart Publishing

Chart publishing executes when code is pushed to the main branch, when Releases are created, or when manually triggered, listening to changes in the `charts/**` directory. The publishing process includes two steps: packaging and pushing.

The packaging stage uses `helm package` to package Charts into `.tgz` files and reads version numbers from `Chart.yaml`:

```yaml
- name: Package and Push Helm Chart
  run: |
    cd charts
    helm package crater/
    CHART_VERSION=$(helm show chart crater/ | grep '^version:' | awk '{print $2}')
```

The pushing stage uses `helm push` to push packaged Charts to the GHCR OCI repository:

```yaml
- name: Login to GHCR
  uses: docker/login-action@v3
  with:
    registry: ${{ env.REGISTRY }}
    username: ${{ github.repository_owner }}
    password: ${{ secrets.GITHUB_TOKEN }}

- name: Package and Push Helm Chart
  run: |
    helm push crater-${CHART_VERSION}.tgz oci://${{ env.REGISTRY }}/${{ env.REPOSITORY }}
```

After Charts are pushed to the `ghcr.io/raids-lab/crater` OCI repository, users can install them with the following commands:

```bash
helm registry login ghcr.io
helm install crater oci://ghcr.io/raids-lab/crater --version <version>
```

After publishing, old Chart versions are automatically cleaned up using `ghcr-cleaner` to clean untagged Charts, keeping at most 10 versions and skipping the `latest` tag:

```yaml
- uses: quartx-analytics/ghcr-cleaner@v1
  with:
    package-name: crater
    delete-untagged: true
    keep-at-most: 10
    skip-tags: latest
```

### Version Management

Helm Chart uses Semantic Versioning to manage version numbers, with version numbers defined in the `version` field of `charts/crater/Chart.yaml`. Version number format is `MAJOR.MINOR.PATCH` (e.g., `0.1.1`), following these rules:

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible feature additions
- **PATCH**: Backward-compatible bug fixes

The PR check stage enforces version number updates, ensuring each Chart change has a corresponding version number update. This helps users track Chart change history and select appropriate versions when upgrading.

When Charts are published, version numbers are pushed to GHCR as tags. Users can install specific Chart versions through version numbers. The cleanup mechanism keeps at most 10 Chart versions, ensuring users can access historical versions while controlling storage space.

