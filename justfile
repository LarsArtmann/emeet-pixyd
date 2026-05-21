default:
    @just --list

# Bump version, commit, push (GitHub Actions auto-tags on push to master)
# Usage: just release 0.2.0
release VERSION:
    #!/usr/bin/env bash
    set -euo pipefail
    new_version="{{ VERSION }}"
    
    if ! echo "$new_version" | grep -qP '^\d+\.\d+(\.\d+)?$'; then
        echo "ERROR: Version must be semver (X.Y.Z), got: $new_version"
        exit 1
    fi
    
    version_file=""
    for f in flake.nix nix/packages/default.nix package.nix; do
        if [ -f "$f" ] && grep -qP 'version\s*=\s*"[^"]' "$f"; then
            version_file="$f"
            break
        fi
    done
    
    if [ -z "$version_file" ]; then
        echo "ERROR: No version found in flake.nix, nix/packages/default.nix, or package.nix"
        exit 1
    fi
    
    old_version=$(grep -oP 'version\s*=\s*"\K[^"]+' "$version_file" | head -1)
    echo "Bumping $old_version -> $new_version in $version_file"
    
    sed -i "s|version = \"$old_version\"|version = \"$new_version\"|" "$version_file"
    
    git add "$version_file"
    git commit -m "release: v$new_version"
    git push
    echo "Pushed. GitHub Actions will auto-tag v$new_version."
