#!/usr/bin/env python3
"""
Generate docs/models.md from OpenRouter catalog API.

Usage:
    curl -s 'https://openrouter.ai/api/frontend/v1/catalog/models' > /tmp/or_models.json
    python3 docs/scripts/generate-openrouter-models.py

The script reads /tmp/or_models.json and writes docs/models.md.
"""

import json
import os
import sys
from collections import defaultdict
from datetime import datetime

INPUT_FILE = "/tmp/or_models.json"
OUTPUT_FILE = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "models.md")

# Priority order for major model families
PRIORITY_FAMILIES = [
    "claude", "gpt", "gemini", "gemma", "llama", "qwen", "deepseek",
    "mistral", "grok", "nova", "minimax", "glm", "ernie", "kimi",
    "nemotron", "cohere", "phi",
]

# Patterns to detect model family from slug
FAMILY_PATTERNS = [
    ("claude", ["claude"]),
    ("gpt", ["gpt-", "gpt-oss", "o1-", "o3-", "o4-"]),
    ("gemini", ["gemini"]),
    ("gemma", ["gemma"]),
    ("llama", ["llama"]),
    ("qwen", ["qwen"]),
    ("deepseek", ["deepseek"]),
    ("mistral", ["mistral", "mixtral", "codestral", "pixtral", "devstral"]),
    ("grok", ["grok"]),
    ("nova", ["nova"]),
    ("cohere", ["command", "north"]),
    ("phi", ["phi"]),
    ("minimax", ["minimax"]),
    ("glm", ["glm"]),
    ("ernie", ["ernie"]),
    ("kimi", ["kimi"]),
    ("nemotron", ["nemotron"]),
    ("yi", ["yi"]),
    ("rwkv", ["rwkv"]),
]


def fmt_price_per_m(p):
    try:
        v = float(p)
    except (ValueError, TypeError):
        return "0"
    v = v * 1_000_000
    if v == 0:
        return "0"
    if v < 0.01:
        return f"${v:.4f}"
    if v < 1:
        return f"${v:.3f}"
    return f"${v:.2f}"


def fmt_ctx(n):
    if n >= 1_000_000:
        return f"{n / 1_000_000:.1f}M" if n % 1_000_000 != 0 else f"{n // 1_000_000}M"
    if n >= 1000:
        return f"{n // 1000}K" if n % 1000 == 0 else f"{n / 1000:.1f}K"
    return str(n)


def fmt_input(inp):
    mapping = {"text": "T", "image": "I", "video": "V", "audio": "A"}
    return "/".join(mapping.get(i, i) for i in inp)


def get_family(slug):
    parts = slug.split("/")
    prefix = parts[0] if len(parts) >= 2 else ""
    model_id = parts[1] if len(parts) >= 2 else slug
    base = model_id.lower().split(":")[0]

    for family, patterns in FAMILY_PATTERNS:
        for pat in patterns:
            if pat in base:
                return family

    if prefix:
        return prefix
    return "other"


def parse_models(raw_models):
    results = []
    for m in raw_models:
        ep = m.get("endpoint") or {}
        pricing = ep.get("pricing", {}) or {}
        rc = m.get("reasoning_config") or {}

        results.append({
            "slug": m.get("slug", ""),
            "name": m.get("name", m.get("slug", "")),
            "author": m.get("author", ""),
            "context_length": m.get("context_length", 0),
            "max_tokens": ep.get("max_completion_tokens", 0) or 0,
            "reasoning": m.get("supports_reasoning", False),
            "mandatory_reasoning": rc.get("is_mandatory_reasoning", False),
            "default_reasoning": rc.get("default_reasoning_enabled", False),
            "input": m.get("input_modalities", []),
            "output": m.get("output_modalities", []),
            "is_free": ep.get("is_free", False),
            "prompt_price": pricing.get("prompt", "0"),
            "completion_price": pricing.get("completion", "0"),
            "cache_read_price": pricing.get("input_cache_read", "0"),
            "cache_write_price": pricing.get("input_cache_write", "0"),
        })
    return results


def generate_markdown(models):
    family_groups = defaultdict(list)
    for m in models:
        family_groups[get_family(m["slug"])].append(m)

    sorted_families = sorted(
        family_groups.keys(),
        key=lambda x: (0, PRIORITY_FAMILIES.index(x)) if x in PRIORITY_FAMILIES else (1, x),
    )

    lines = []
    lines.append("# OpenRouter Model Catalog")
    lines.append("")
    lines.append(f"> Auto-generated from OpenRouter API on {datetime.now().strftime('%Y-%m-%d')}")
    lines.append(">")
    total = len(models)
    reasoning_count = sum(1 for m in models if m["reasoning"])
    free_count = sum(1 for m in models if m["is_free"])
    lines.append(f"> Total models: **{total}** | Reasoning: **{reasoning_count}** | Free: **{free_count}**")
    lines.append("")
    lines.append("## How to Update")
    lines.append("")
    lines.append("```bash")
    lines.append("# 1. Fetch latest catalog")
    lines.append("curl -s 'https://openrouter.ai/api/frontend/v1/catalog/models' > /tmp/or_models.json")
    lines.append("")
    lines.append("# 2. Run this script")
    lines.append("python3 docs/scripts/generate-openrouter-models.py")
    lines.append("")
    lines.append("# 3. Review and commit")
    lines.append("# git diff docs/models.md")
    lines.append("```")
    lines.append("")
    lines.append("## Legend")
    lines.append("")
    lines.append("| Symbol | Meaning |")
    lines.append("|--------|---------|")
    lines.append("| 🧠 | Supports reasoning/thinking |")
    lines.append("| 🆓 | Free tier available |")
    lines.append("| ⚠️ | Mandatory reasoning (cannot disable) |")
    lines.append("| T | Text input |")
    lines.append("| I | Image input |")
    lines.append("| V | Video input |")
    lines.append("| A | Audio input |")
    lines.append("")
    lines.append("## Model Groups")
    lines.append("")
    lines.append("| Group | Count |")
    lines.append("|-------|-------|")
    for fam in sorted_families:
        label = fam.replace("-", " ").title()
        lines.append(f"| [{label}](#{fam}) | {len(family_groups[fam])} |")
    lines.append("")

    for fam in sorted_families:
        ms = sorted(family_groups[fam], key=lambda x: x["slug"])
        label = fam.replace("-", " ").title()
        lines.append("---")
        lines.append("")
        lines.append(f'## {label} <a id="{fam}"></a>')
        lines.append("")
        lines.append("| Model ID | Name | 🧠 | 🆓 | Context | Max Output | Input | Prompt $/M | Completion $/M |")
        lines.append("|----------|------|-----|-----|---------|-----------|-------|-----------|----------------|")

        for m in ms:
            name = m["name"]
            if len(name) > 50:
                name = name[:47] + "..."

            reasoning_icon = ""
            if m["reasoning"]:
                reasoning_icon = "⚠️" if m["mandatory_reasoning"] else "🧠"

            free_icon = "🆓" if m["is_free"] else ""
            ctx = fmt_ctx(m["context_length"])
            max_t = fmt_ctx(m["max_tokens"]) if m["max_tokens"] else "-"
            inp = fmt_input(m["input"])
            prompt_m = fmt_price_per_m(m["prompt_price"])
            comp_m = fmt_price_per_m(m["completion_price"])

            lines.append(
                f"| `{m['slug']}` | {name} | {reasoning_icon} | {free_icon} | {ctx} | {max_t} | {inp} | {prompt_m} | {comp_m} |"
            )
        lines.append("")

    # Summary
    lines.append("---")
    lines.append("")
    lines.append("## Summary Statistics")
    lines.append("")

    ctx_ranges = defaultdict(int)
    for m in models:
        c = m["context_length"]
        if c >= 1_000_000:
            ctx_ranges["1M+"] += 1
        elif c >= 500_000:
            ctx_ranges["500K-1M"] += 1
        elif c >= 200_000:
            ctx_ranges["200K-500K"] += 1
        elif c >= 100_000:
            ctx_ranges["100K-200K"] += 1
        elif c >= 32_000:
            ctx_ranges["32K-100K"] += 1
        else:
            ctx_ranges["<32K"] += 1

    lines.append("### Context Length Distribution")
    lines.append("")
    lines.append("| Range | Count |")
    lines.append("|-------|-------|")
    for r in ["1M+", "500K-1M", "200K-500K", "100K-200K", "32K-100K", "<32K"]:
        lines.append(f"| {r} | {ctx_ranges[r]} |")
    lines.append("")

    input_counts = defaultdict(int)
    for m in models:
        key = "/".join(sorted(m["input"]))
        input_counts[key] += 1

    lines.append("### Input Modality Distribution")
    lines.append("")
    lines.append("| Modalities | Count |")
    lines.append("|------------|-------|")
    for k, v in sorted(input_counts.items(), key=lambda x: -x[1]):
        lines.append(f"| {k} | {v} |")
    lines.append("")

    return "\n".join(lines)


def main():
    if not os.path.exists(INPUT_FILE):
        print(f"Error: {INPUT_FILE} not found.", file=sys.stderr)
        print("Run: curl -s 'https://openrouter.ai/api/frontend/v1/catalog/models' > /tmp/or_models.json", file=sys.stderr)
        sys.exit(1)

    with open(INPUT_FILE) as f:
        data = json.load(f)

    raw_models = data.get("data", [])
    if not raw_models:
        print("Error: no models found in input file.", file=sys.stderr)
        sys.exit(1)

    models = parse_models(raw_models)
    content = generate_markdown(models)

    with open(OUTPUT_FILE, "w") as f:
        f.write(content)

    print(f"Generated {OUTPUT_FILE}: {len(models)} models")


if __name__ == "__main__":
    main()
