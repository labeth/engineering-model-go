#!/usr/bin/env bash
# Sourced by generate-examples.sh and validate-all.sh. Defines quint_models,
# which prints the engmod.yml manifest of every maintained model (the root model
# and each example) that has behavior machines, or that still has a committed
# generated/quint directory so that engquint generate removes modules of
# machines that were deleted. A model without machines has nothing to generate.

quint_models() {
  local root="${1:-.}" manifest dir
  while IFS= read -r manifest; do
    dir="$(dirname "$manifest")"
    if grep -Eqs '^[[:space:]]*-[[:space:]]+id:[[:space:]]+SM-' "$dir/model/behavior.yml" || [ -d "$dir/generated/quint" ]; then
      printf '%s\n' "$manifest"
    fi
  done < <({ [ -f "$root/engmod.yml" ] && echo "$root/engmod.yml"; find "$root/examples" -name engmod.yml -not -path '*/node_modules/*' 2>/dev/null | sort; })
}
