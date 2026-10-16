#!/usr/bin/env bash
# Copyright 2026 The Crater Project Team, RAIDS-Lab
# Run complete rendered startup scripts with isolated users, paths and services.
set -euo pipefail

scratch="$(mktemp -d "${TMPDIR:-/tmp}/crater-startup-compatibility.XXXXXX")"
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/real"

for tool in mkdir mktemp chmod mv; do
    ln -s "$(command -v "$tool")" "$scratch/real/$tool"
    cat > "$scratch/bin/$tool" <<'EOF'
#!/usr/bin/env bash
set -eu
tool=${0##*/}
if [ "$CRATER_TEST_FAIL" = write ] && [ "$tool" = mktemp ]; then
    mkdir -p "$CRATER_TEST_ROOT/session/unwritable"
    printf '%s\n' "$CRATER_TEST_ROOT/session/unwritable"
    exit 0
fi
if [ "$CRATER_TEST_FAIL" = "$tool" ] || { [ "$CRATER_TEST_FAIL" = chmod-private ] && [ "$tool" = chmod ] && [ "$1" = 600 ]; }; then
    for arg do
        case "$arg" in
            "$CRATER_TEST_ROOT/session"|"$CRATER_TEST_ROOT/session/"*)
                echo "injected $tool failure" >&2
                exit 1;;
        esac
    done
fi
exec "$CRATER_TEST_REAL/$tool" "$@"
EOF
done

cat > "$scratch/bin/getent" <<'EOF'
#!/bin/sh
case "$1" in
    passwd) printf 'alice:x:1234:5678::%s:/bin/sh\n' "$CRATER_TEST_HOME";;
    group) printf 'users:x:5678:alice\n';;
    *) exit 1;;
esac
EOF
cat > "$scratch/bin/id" <<'EOF'
#!/bin/sh
case "$1" in -u) echo 1234;; -g) echo 5678;; *) echo 'uid=1234(alice) gid=5678(users)';; esac
EOF
cat > "$scratch/bin/chown" <<'EOF'
#!/bin/sh
test "$1" = 1234:5678 || exit 1
case "$2" in "$CRATER_TEST_ROOT/"*) ;; *) exit 1;; esac
printf '%s\n' "$2" >> "$CRATER_TEST_ROOT/ownership"
EOF
cat > "$scratch/bin/sudo" <<'EOF'
#!/bin/sh
while [ "$#" -gt 0 ]; do
    case "$1" in
        --user) test "$2" = alice || exit 1; shift 2;;
        --preserve-env*|--set-home) shift;;
        *) break;;
    esac
done
exec env HOME="$CRATER_TEST_HOME" USER=alice "$@"
EOF
cat > "$scratch/bin/service-main" <<'EOF'
#!/bin/sh
set -eu
test "$HOME" = "$CRATER_TEST_HOME"
test "$#" = 2
test "$1" = '--fixture-argument=two words'
test -z "$2"
test -d "$CRATER_TEST_ROOT/home/alice"
test -d "$HOME"
printf 'notebook\n' > "$CRATER_TEST_ROOT/home/alice/notebook"
printf 'configuration\n' > "$HOME/configuration"
printf 'main-service-started\n'
EOF
ln -s service-main "$scratch/bin/jupyter"
ln -s service-main "$scratch/bin/code-server"
chmod +x "$scratch/bin/"*

custom_script=$1
unified_script=$2
for kind in custom unified; do
    source_script="$custom_script"
    if [ "$kind" = unified ]; then source_script="$unified_script"; fi
    for service in jupyter code-server; do
        for failure in none mkdir mktemp chmod-private write mv symlink; do
            case_root="$scratch/$kind-$service-$failure"
            mkdir -p "$case_root/etc/sudoers.d"
            if [ "$failure" = symlink ]; then
                mkdir "$case_root/target"
                ln -s "$case_root/target" "$case_root/session"
            fi
            sed -e "s|/home/|$case_root/home/|g" \
                -e "s|/etc/sudoers.d|$case_root/etc/sudoers.d|g" \
                -e "s|/run/crater|$case_root/session|g" \
                "$source_script" > "$case_root/start.sh"
            if ! PATH="$scratch/bin:$PATH" NB_USER=alice NB_UID=1001 NB_GID=1001 \
                SHELL=/bin/sh PYTHONUSERBASE=/fixture/python \
                CRATER_TEST_ROOT="$case_root" CRATER_TEST_HOME="$case_root/image-home" \
                CRATER_TEST_REAL="$scratch/real" CRATER_TEST_FAIL="$failure" \
                bash "$case_root/start.sh" "$service" '--fixture-argument=two words' '' > "$case_root/output" 2> "$case_root/error"; then
                cat "$case_root/error" >&2
                echo "$kind/$service failed to start after $failure" >&2
                exit 1
            fi
            grep -q main-service-started "$case_root/output"
            grep -Fxq "$case_root/home/alice" "$case_root/ownership"
            grep -Fxq "$case_root/image-home" "$case_root/ownership"
            test -f "$case_root/home/alice/notebook"
            test -f "$case_root/image-home/configuration"
            if [ "$failure" = none ]; then
                test -f "$case_root/session/environment.sh"
            else
                grep -q 'could not save startup environment' "$case_root/error"
                test ! -f "$case_root/session/environment.sh"
                test ! -f "$case_root/target/environment.sh"
                if compgen -G "$case_root/session/.environment.*" > /dev/null; then
                    echo "private temporary environment file leaked after $failure" >&2
                    exit 1
                fi
            fi
        done
    done
done
echo "Startup compatibility: both HOME paths, original service argv and snapshot-failure fallback passed."
