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

import argparse
import re

GROUP_LABELS = {
    400: 'Bad Request',
    401: 'Auth',
    403: 'Forbidden',
    404: 'Not Found',
    405: 'Method Not Allowed',
    409: 'Conflict',
    413: 'Payload Too Large',
    429: 'Rate Limit',
    500: 'Internal',
    502: 'Bad Gateway',
    503: 'Service Unavailable',
    504: 'Gateway Timeout',
}

GROUP_CONSTANT_NAMES = {
    400: 'BAD_REQUEST_ERROR_GROUP',
    401: 'AUTH_ERROR_GROUP',
    403: 'FORBIDDEN_ERROR_GROUP',
    404: 'NOT_FOUND_ERROR_GROUP',
    405: 'METHOD_NOT_ALLOWED_ERROR_GROUP',
    409: 'CONFLICT_ERROR_GROUP',
    413: 'PAYLOAD_TOO_LARGE_ERROR_GROUP',
    429: 'RATE_LIMIT_ERROR_GROUP',
    500: 'INTERNAL_ERROR_GROUP',
    502: 'BAD_GATEWAY_ERROR_GROUP',
    503: 'SERVICE_UNAVAILABLE_ERROR_GROUP',
    504: 'GATEWAY_TIMEOUT_ERROR_GROUP',
}


def parse_go_error_codes(go_file_content: str, deprecated_only: bool = False):
    """
    Parse error code definitions from Go source.
    Supported forms:
    1) Legacy constants: Name ErrorCode = 40001
    2) New constants:    Name bizerr.BizCode = 40001
    3) Struct tags:      Name BizCode `code:"40001"`
    """
    matches = []
    previous_comment_is_deprecated = False

    const_pattern = re.compile(r'^\s*(\w+)\s+(?:ErrorCode|bizerr\.BizCode)\s*=\s*(\d+)\s*$')
    tag_pattern = re.compile(r'^\s*(\w+)\s+BizCode\s+`code:"(\d+)"`')

    for line in go_file_content.splitlines():
        stripped = line.strip()
        if stripped.startswith('//'):
            previous_comment_is_deprecated = 'Deprecated:' in stripped
            continue

        const_match = const_pattern.match(line)
        if const_match:
            if not deprecated_only or previous_comment_is_deprecated:
                matches.append((const_match.group(1), const_match.group(2)))
            previous_comment_is_deprecated = False
            continue

        tag_match = tag_pattern.match(line)
        if tag_match:
            if not deprecated_only or previous_comment_is_deprecated:
                matches.append((tag_match.group(1), tag_match.group(2)))
            previous_comment_is_deprecated = False
            continue

        previous_comment_is_deprecated = False

    # de-duplicate by symbol name while preserving first-seen order
    deduplicated = []
    seen = set()
    for name, value in matches:
        if name in seen:
            continue
        seen.add(name)
        deduplicated.append((name, value))

    return deduplicated


def camel_to_screaming_snake(name: str):
    result = []
    for index, char in enumerate(name):
        if index > 0 and char.isupper():
            prev_char = name[index - 1]
            next_char = name[index + 1] if index + 1 < len(name) else ''
            if prev_char.islower() or prev_char.isdigit():
                result.append('_')
            elif prev_char.isupper() and next_char.islower():
                result.append('_')
        result.append(char)
    return ''.join(result).upper()


def to_ts_error_code_name(go_error_code_name: str, prefix: str):
    if go_error_code_name == 'OK':
        return f'{prefix}OK'
    return prefix + 'ERROR_' + camel_to_screaming_snake(go_error_code_name)


def group_header(code_value: int):
    if code_value == 0:
        return None

    group = code_value // 100
    label = GROUP_LABELS.get(group)
    if label is None:
        return f'// {group}xx'
    return f'// {group}xx - {label}'


def emit_group_constants(ts_error_code_file, go_error_code_matches, prefix: str):
    group_codes = []
    seen_groups = set()

    for _, value in go_error_code_matches:
        code_value = int(value)
        if code_value == 0:
            continue

        group_code = code_value // 100
        if group_code in seen_groups:
            continue

        seen_groups.add(group_code)
        group_codes.append(group_code)

    for group_code in group_codes:
        group_name = GROUP_CONSTANT_NAMES.get(group_code)
        if group_name is None:
            continue
        ts_error_code_file.write(f'export const {prefix}{group_name}: ErrorCode = {group_code}\n')


def generate_ts_error_code_file(
    go_error_code_file_path: str,
    ts_error_code_file_path: str,
    prefix: str = '',
    include_ok: bool = True,
    deprecated_only: bool = False,
):
    # Read Go ErrorCode File
    go_error_code_file = open(go_error_code_file_path, 'r')
    go_error_code_file_content = go_error_code_file.read()
    go_error_code_file.close()

    # Parse Go ErrorCode File
    go_error_code_matches = parse_go_error_codes(go_error_code_file_content, deprecated_only)
    if include_ok and not any(name == 'OK' for name, _ in go_error_code_matches):
        go_error_code_matches = [('OK', '0'), *go_error_code_matches]

    # Generate TypeScript ErrorCode File
    ts_error_code_file = open(ts_error_code_file_path, 'w')
    ts_error_code_file.write('// This file is generated by generator.py\n')
    ts_error_code_file.write('// Please do not modify this file manually\n\n')
    ts_error_code_file.write('export type ErrorCode = number\n\n')
    emit_group_constants(ts_error_code_file, go_error_code_matches, prefix)
    ts_error_code_file.write('\n')

    current_group_header = None
    for go_error_code_match in go_error_code_matches:
        go_error_code_name = go_error_code_match[0]
        go_error_code_value = int(go_error_code_match[1])

        next_group_header = group_header(go_error_code_value)
        if next_group_header != current_group_header:
            if current_group_header is not None:
                ts_error_code_file.write('\n')
            if next_group_header is not None:
                ts_error_code_file.write(f'{next_group_header}\n')
            current_group_header = next_group_header

        ts_error_code_name = to_ts_error_code_name(go_error_code_name, prefix)
        ts_error_code_file.write(f'export const {ts_error_code_name}: ErrorCode = {go_error_code_value}\n')

    ts_error_code_file.close()

if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Generate TypeScript error code constants from Go source.'
    )
    parser.add_argument('go_error_code_file_path')
    parser.add_argument('ts_error_code_file_path')
    parser.add_argument('--prefix', default='')
    parser.add_argument('--deprecated-only', action='store_true')
    parser.add_argument('--no-ok', action='store_true')
    args = parser.parse_args()

    generate_ts_error_code_file(
        args.go_error_code_file_path,
        args.ts_error_code_file_path,
        prefix=args.prefix,
        include_ok=not args.no_ok,
        deprecated_only=args.deprecated_only,
    )
    print('TypeScript ErrorCode File Generated Successfully')
