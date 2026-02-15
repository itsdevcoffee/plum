#!/usr/bin/env python3
"""Detect the tech stack of a project directory.

Zero external dependencies - stdlib only.
Outputs JSON dict with languages, tags, frameworks, config_files_found, has_claude_code.
"""

import json
import os
import sys
from collections import Counter
from pathlib import Path

SKIP_DIRS = {
    ".git", "node_modules", "vendor", "venv", ".venv", "dist", "build",
    "__pycache__", "target", ".next", ".nuxt", "coverage", ".tox",
    "env", ".env", ".mypy_cache", ".pytest_cache", "eggs", "*.egg-info",
}

MAX_DEPTH = 3

# Maps indicator files to (language, [tags])
INDICATOR_MAP = {
    "go.mod":              ("go",         ["go", "backend"]),
    "go.sum":              ("go",         ["go"]),
    "package.json":        ("javascript", ["javascript", "nodejs"]),
    "tsconfig.json":       ("typescript", ["typescript"]),
    "Cargo.toml":          ("rust",       ["rust"]),
    "pyproject.toml":      ("python",     ["python"]),
    "requirements.txt":    ("python",     ["python"]),
    "setup.py":            ("python",     ["python"]),
    "Pipfile":             ("python",     ["python", "pipenv"]),
    "Gemfile":             ("ruby",       ["ruby"]),
    "pom.xml":             ("java",       ["java", "maven"]),
    "build.gradle":        ("java",       ["java", "gradle"]),
    "build.gradle.kts":    ("kotlin",     ["kotlin", "gradle"]),
    "CMakeLists.txt":      ("cpp",        ["cpp", "cmake"]),
    "Makefile":            (None,         ["make"]),
    "Dockerfile":          (None,         ["docker", "containerization"]),
    "docker-compose.yml":  (None,         ["docker", "containerization"]),
    "docker-compose.yaml": (None,         ["docker", "containerization"]),
    ".dockerignore":       (None,         ["docker"]),
    "terraform.tf":        (None,         ["terraform", "infrastructure"]),
    "main.tf":             (None,         ["terraform", "infrastructure"]),
    "serverless.yml":      (None,         ["serverless", "cloud"]),
    "fly.toml":            (None,         ["fly-io", "deployment"]),
    "vercel.json":         (None,         ["vercel", "deployment"]),
    "netlify.toml":        (None,         ["netlify", "deployment"]),
    "CLAUDE.md":           (None,         ["claude-code"]),
    ".claude":             (None,         ["claude-code"]),
}

# Check for directories as indicators
DIR_INDICATORS = {
    ".github/workflows": ["ci-cd", "github-actions"],
    ".github":           ["github"],
    ".circleci":         ["ci-cd", "circleci"],
}

# Extension to language mapping
EXT_LANG_MAP = {
    ".go":   "go",
    ".py":   "python",
    ".js":   "javascript",
    ".ts":   "typescript",
    ".tsx":  "typescript",
    ".jsx":  "javascript",
    ".rs":   "rust",
    ".rb":   "ruby",
    ".java": "java",
    ".kt":   "kotlin",
    ".cpp":  "cpp",
    ".c":    "c",
    ".cs":   "csharp",
    ".swift":"swift",
    ".lua":  "lua",
    ".php":  "php",
    ".ex":   "elixir",
    ".exs":  "elixir",
    ".zig":  "zig",
}

# Dependency patterns: (dep_substring, [tags])
GO_DEP_PATTERNS = [
    ("github.com/charmbracelet/bubbletea",  ["bubbletea", "tui", "terminal"]),
    ("github.com/charmbracelet/lipgloss",   ["lipgloss", "tui", "terminal"]),
    ("github.com/charmbracelet/bubbles",    ["bubbles", "tui", "terminal"]),
    ("github.com/charmbracelet/glamour",    ["glamour", "tui", "markdown"]),
    ("github.com/charmbracelet/huh",        ["huh", "tui", "forms"]),
    ("github.com/spf13/cobra",              ["cobra", "cli"]),
    ("github.com/spf13/viper",              ["viper", "config"]),
    ("github.com/urfave/cli",               ["urfave-cli", "cli"]),
    ("github.com/gin-gonic/gin",            ["gin", "web", "api"]),
    ("github.com/gofiber/fiber",            ["fiber", "web", "api"]),
    ("github.com/labstack/echo",            ["echo", "web", "api"]),
    ("github.com/gorilla/mux",              ["gorilla", "web", "api"]),
    ("gorm.io/gorm",                        ["gorm", "database", "orm"]),
    ("github.com/jmoiron/sqlx",             ["sqlx", "database"]),
    ("google.golang.org/grpc",              ["grpc", "api"]),
    ("google.golang.org/protobuf",          ["protobuf", "api"]),
]

NPM_DEP_PATTERNS = [
    ("react",           ["react", "frontend"]),
    ("next",            ["nextjs", "react", "frontend", "fullstack"]),
    ("vue",             ["vue", "frontend"]),
    ("nuxt",            ["nuxt", "vue", "frontend", "fullstack"]),
    ("svelte",          ["svelte", "frontend"]),
    ("angular",         ["angular", "frontend"]),
    ("remotion",        ["remotion", "video", "creative"]),
    ("express",         ["express", "backend", "api"]),
    ("fastify",         ["fastify", "backend", "api"]),
    ("tailwindcss",     ["tailwind", "css", "design"]),
    ("prisma",          ["prisma", "database", "orm"]),
    ("drizzle-orm",     ["drizzle", "database", "orm"]),
    ("supabase",        ["supabase", "database", "backend"]),
    ("firebase",        ["firebase", "backend"]),
    ("electron",        ["electron", "desktop"]),
    ("expo",            ["expo", "react-native", "mobile"]),
    ("react-native",    ["react-native", "mobile"]),
    ("three",           ["threejs", "3d", "creative"]),
    ("typescript",      ["typescript"]),
    ("jest",            ["jest", "testing"]),
    ("vitest",          ["vitest", "testing"]),
    ("playwright",      ["playwright", "testing", "e2e"]),
    ("cypress",         ["cypress", "testing", "e2e"]),
    ("storybook",       ["storybook", "design", "components"]),
]

PYTHON_DEP_PATTERNS = [
    ("django",          ["django", "web", "fullstack"]),
    ("flask",           ["flask", "web", "api"]),
    ("fastapi",         ["fastapi", "api", "backend"]),
    ("sqlalchemy",      ["sqlalchemy", "database", "orm"]),
    ("pytest",          ["pytest", "testing"]),
    ("celery",          ["celery", "async", "tasks"]),
    ("pydantic",        ["pydantic", "validation"]),
    ("torch",           ["pytorch", "ml", "ai"]),
    ("tensorflow",      ["tensorflow", "ml", "ai"]),
    ("transformers",    ["huggingface", "ml", "ai", "nlp"]),
    ("langchain",       ["langchain", "ai", "llm"]),
    ("anthropic",       ["anthropic", "ai", "claude"]),
    ("openai",          ["openai", "ai", "llm"]),
    ("scrapy",          ["scrapy", "scraping"]),
    ("streamlit",       ["streamlit", "dashboard", "data"]),
    ("pandas",          ["pandas", "data"]),
    ("numpy",           ["numpy", "data", "scientific"]),
]


def _should_skip(name):
    return name in SKIP_DIRS or name.startswith(".")


def _walk_limited(root, max_depth):
    """Walk directory tree up to max_depth levels."""
    root = Path(root)
    for item in root.iterdir():
        try:
            if item.is_file():
                yield item, 0
            elif item.is_dir() and not _should_skip(item.name):
                if max_depth > 0:
                    for sub_item, depth in _walk_limited(item, max_depth - 1):
                        yield sub_item, depth + 1
        except PermissionError:
            continue


def _parse_go_mod(path):
    """Extract tags from go.mod require block."""
    tags = set()
    frameworks = []
    try:
        text = Path(path).read_text(errors="ignore")
        for dep_str, dep_tags in GO_DEP_PATTERNS:
            if dep_str in text:
                tags.update(dep_tags)
                frameworks.append(dep_str.split("/")[-1])
    except (OSError, IOError):
        pass
    return tags, frameworks


def _parse_package_json(path):
    """Extract tags from package.json dependencies."""
    tags = set()
    frameworks = []
    try:
        data = json.loads(Path(path).read_text(errors="ignore"))
        all_deps = set()
        for key in ("dependencies", "devDependencies", "peerDependencies"):
            all_deps.update(data.get(key, {}).keys())

        for dep_name, dep_tags in NPM_DEP_PATTERNS:
            # Check for exact match or scoped package
            if dep_name in all_deps or any(d.endswith("/" + dep_name) for d in all_deps):
                tags.update(dep_tags)
                frameworks.append(dep_name)
    except (OSError, IOError, json.JSONDecodeError):
        pass
    return tags, frameworks


def _parse_python_deps(path):
    """Extract tags from requirements.txt or pyproject.toml."""
    tags = set()
    frameworks = []
    try:
        text = Path(path).read_text(errors="ignore").lower()
        for dep_name, dep_tags in PYTHON_DEP_PATTERNS:
            if dep_name in text:
                tags.update(dep_tags)
                frameworks.append(dep_name)
    except (OSError, IOError):
        pass
    return tags, frameworks


def detect_stack(target_dir):
    """Detect the tech stack of the given directory.

    Returns dict with languages, tags, frameworks, config_files_found, has_claude_code.
    """
    target = Path(target_dir).resolve()
    if not target.is_dir():
        return {"error": f"Not a directory: {target}"}

    languages = set()
    tags = set()
    frameworks = []
    config_files = []
    ext_counts = Counter()

    # Phase 1: Indicator files (root level)
    for name, (lang, file_tags) in INDICATOR_MAP.items():
        indicator_path = target / name
        if indicator_path.exists():
            config_files.append(name)
            if lang:
                languages.add(lang)
            tags.update(file_tags)

    # Check directory indicators
    for dir_path, dir_tags in DIR_INDICATORS.items():
        if (target / dir_path).is_dir():
            config_files.append(dir_path)
            tags.update(dir_tags)

    # Phase 2: Dependency parsing
    go_mod = target / "go.mod"
    if go_mod.exists():
        dep_tags, dep_frameworks = _parse_go_mod(go_mod)
        tags.update(dep_tags)
        frameworks.extend(dep_frameworks)

    pkg_json = target / "package.json"
    if pkg_json.exists():
        dep_tags, dep_frameworks = _parse_package_json(pkg_json)
        tags.update(dep_tags)
        frameworks.extend(dep_frameworks)

    for pyfile in ("requirements.txt", "pyproject.toml", "Pipfile"):
        pypath = target / pyfile
        if pypath.exists():
            dep_tags, dep_frameworks = _parse_python_deps(pypath)
            tags.update(dep_tags)
            frameworks.extend(dep_frameworks)

    # Phase 3: Extension counting (fallback language signal)
    for item, _depth in _walk_limited(target, MAX_DEPTH):
        ext = item.suffix.lower()
        if ext in EXT_LANG_MAP:
            ext_counts[ext] += 1

    for ext, count in ext_counts.most_common():
        if count >= 2:
            languages.add(EXT_LANG_MAP[ext])

    # Derive additional tags from languages
    has_claude = "claude-code" in tags or (target / ".claude").is_dir() or (target / "CLAUDE.md").exists()

    return {
        "languages": sorted(languages),
        "tags": sorted(tags),
        "frameworks": sorted(set(frameworks)),
        "config_files_found": sorted(config_files),
        "has_claude_code": has_claude,
    }


if __name__ == "__main__":
    target = sys.argv[1] if len(sys.argv) > 1 else "."
    result = detect_stack(target)
    print(json.dumps(result, indent=2))
