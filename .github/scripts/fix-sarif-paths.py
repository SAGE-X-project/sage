#!/usr/bin/env python3
"""
Fix SARIF artifact locations for GitHub Code Scanning.

GitHub Code Scanning requires artifact locations to have full paths
from repository root, but gosec outputs only filenames.
This script uses gosec JSON output to get accurate file paths.
"""

import json
import os
import sys

def main():
    sarif_file = "gosec-results-raw.sarif"
    json_file = "gosec-results.json"
    output_file = "gosec-results.sarif"

    if not os.path.exists(sarif_file):
        print(f"Error: {sarif_file} not found")
        sys.exit(1)

    with open(sarif_file, "r") as f:
        sarif = json.load(f)

    # Build file mapping from gosec JSON output (has full paths)
    file_line_map = {}
    if os.path.exists(json_file):
        with open(json_file, "r") as f:
            gosec_json = json.load(f)
            for issue in gosec_json.get("Issues", []):
                file_path = issue.get("file", "")
                line = issue.get("line", "")
                filename = os.path.basename(file_path)

                # Remove absolute path prefix if present
                if file_path.startswith(os.getcwd() + "/"):
                    file_path = file_path[len(os.getcwd()) + 1:]

                # Map (filename, line) -> full_path for accurate matching
                key = (filename, str(line))
                file_line_map[key] = file_path

                # Also create filename-only fallback
                if filename not in file_line_map or "/test" in file_line_map.get(filename, ""):
                    file_line_map[filename] = file_path

        print(f"Loaded {len(file_line_map)} file mappings from JSON")

    # Fix artifact locations in SARIF
    fixed_count = 0
    examples = []

    for run in sarif.get("runs", []):
        for result in run.get("results", []):
            for location in result.get("locations", []):
                phys_loc = location.get("physicalLocation", {})
                artifact_loc = phys_loc.get("artifactLocation", {})
                uri = artifact_loc.get("uri", "")

                # Get line number for precise matching
                region = phys_loc.get("region", {})
                line = str(region.get("startLine", ""))

                # If uri is just a filename, look up full path
                if uri and "/" not in uri:
                    old_uri = uri
                    new_uri = None

                    # Try exact match with line number first
                    if line:
                        key = (uri, line)
                        new_uri = file_line_map.get(key)

                    # Fall back to filename-only match
                    if not new_uri:
                        new_uri = file_line_map.get(uri)

                    if new_uri:
                        artifact_loc["uri"] = new_uri
                        fixed_count += 1
                        if len(examples) < 5:
                            examples.append(f"  {old_uri}:{line} -> {new_uri}")

    for example in examples:
        print(example)

    if fixed_count > 5:
        print(f"  ... and {fixed_count - 5} more")

    print(f"\nTotal artifact locations fixed: {fixed_count}")

    with open(output_file, "w") as f:
        json.dump(sarif, f, indent=2)

    print(f"✓ Fixed SARIF saved to {output_file}")

if __name__ == "__main__":
    main()
