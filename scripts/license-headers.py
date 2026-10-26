#!/usr/bin/env python3
"""Проверяет (или добавляет с --fix) заголовок лицензии в исходниках Kubron.

Использование:  python3 scripts/license-headers.py [--fix]
Проверяются отслеживаемые git файлы *.go, *.ts, *.vue (кроме сгенерированных).
"""
import subprocess, sys

SPDX = "SPDX-License-Identifier: Apache-2.0"
COPYRIGHT = "Copyright 2026 The Kubron Authors"
SKIP_PREFIXES = ("frontend/wailsjs/", "frontend/node_modules/", "build/")


def header(path: str) -> str:
    if path.endswith(".vue"):
        return f"<!-- {COPYRIGHT} -->\n<!-- {SPDX} -->\n"
    return f"// {COPYRIGHT}\n// {SPDX}\n\n"


def main() -> int:
    fix = "--fix" in sys.argv
    files = subprocess.check_output(["git", "ls-files", "*.go", "*.ts", "*.vue"], text=True).split()
    missing = []
    for path in files:
        if path.startswith(SKIP_PREFIXES):
            continue
        with open(path, encoding="utf-8") as f:
            text = f.read()
        if "Code generated" in text[:400] and path.endswith(".go"):
            continue
        if SPDX in text[:300]:
            continue
        missing.append(path)
        if fix:
            with open(path, "w", encoding="utf-8", newline="") as f:
                f.write(header(path) + text)
    if missing and not fix:
        print("Нет заголовка лицензии (python3 scripts/license-headers.py --fix):")
        print("\n".join("  " + p for p in missing))
        return 1
    if missing:
        print(f"Добавлен заголовок: {len(missing)} файлов")
    return 0


sys.exit(main())
