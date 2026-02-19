#!/usr/bin/env python3
"""
Fix SARIF artifact locations for GitHub Code Scanning.

GitHub Code Scanning requires artifact locations to have full paths
from repository root, but gosec outputs only filenames.
This script uses gosec JSON output to get accurate file paths and
filters out invalid entries (e.g., build cache files).
"""

import json
import os
import sys

def main():
    sarif_file = "gosec-results-raw.sarif"
    json_file = "gosec-results.json"
    output_file = "gosec-results.sarif"

    repo_root = os.getcwd()

    if not os.path.exists(sarif_file):
        print(f"Error: {sarif_file} not found")
        sys.exit(1)

    with open(sarif_file, "r") as f:
        sarif = json.load(f)

    # Build file mapping from gosec JSON output (has full paths)
    file_line_map = {}
    valid_files = set()  # Track files within repo

    if os.path.exists(json_file):
        with open(json_file, "r") as f:
            gosec_json = json.load(f)
            for issue in gosec_json.get("Issues", []):
                file_path = issue.get("file", "")
                line = issue.get("line", "")
                filename = os.path.basename(file_path)

                # Skip files outside repository (e.g., build cache)
                if not file_path.startswith(repo_root + "/"):
                    continue

                # Convert to relative path
                file_path = file_path[len(repo_root) + 1:]
                valid_files.add(file_path)

                # Map (filename, line) -> full_path for accurate matching
                key = (filename, str(line))
                file_line_map[key] = file_path

                # Also create filename-only fallback
                if filename not in file_line_map or "/test" in file_line_map.get(filename, ""):
                    file_line_map[filename] = file_path

        print(f"Loaded {len(file_line_map)} file mappings from JSON")
        print(f"Valid files in repository: {len(valid_files)}")

    # Sanitize invalid relationships in tool driver rules
    # gosec outputs null entries in relationships arrays which violates SARIF schema
    sanitized_rules = 0
    for run in sarif.get("runs", []):
        driver = run.get("tool", {}).get("driver", {})
        for rule in driver.get("rules", []):
            if "relationships" in rule:
                original = rule["relationships"]
                if isinstance(original, list):
                    cleaned = [r for r in original if r is not None and isinstance(r, dict)]
                    if len(cleaned) != len(original):
                        sanitized_rules += 1
                    if cleaned:
                        rule["relationships"] = cleaned
                    else:
                        del rule["relationships"]
                elif original is None:
                    del rule["relationships"]
                    sanitized_rules += 1

    if sanitized_rules > 0:
        print(f"Sanitized {sanitized_rules} rules with invalid relationships entries")

    # Fix artifact locations in SARIF and filter invalid results
    fixed_count = 0
    removed_count = 0
    examples = []

    for run in sarif.get("runs", []):
        original_results = run.get("results", [])
        filtered_results = []

        for result in original_results:
            valid_result = True

            # Skip results with no locations array
            locations = result.get("locations", [])
            if not locations:
                removed_count += 1
                continue

            for location in locations:
                # Skip locations with no physicalLocation
                if "physicalLocation" not in location:
                    valid_result = False
                    removed_count += 1
                    break

                phys_loc = location.get("physicalLocation", {})

                # Skip if no artifactLocation
                if "artifactLocation" not in phys_loc:
                    valid_result = False
                    removed_count += 1
                    break

                artifact_loc = phys_loc.get("artifactLocation", {})
                uri = artifact_loc.get("uri", "")

                # Get line number for precise matching
                region = phys_loc.get("region", {})
                line = str(region.get("startLine", ""))

                # Skip results with empty or missing URI
                if not uri:
                    valid_result = False
                    removed_count += 1
                    break

                # If uri is just a filename, look up full path
                if "/" not in uri:
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
                    else:
                        # Cannot map to repository file
                        valid_result = False
                        removed_count += 1
                        break
                elif uri not in valid_files:
                    # URI is full path but not in repository
                    valid_result = False
                    removed_count += 1
                    break

            # Only include valid results
            if valid_result:
                filtered_results.append(result)

        # Replace results with filtered list
        run["results"] = filtered_results

    for example in examples:
        print(example)

    if fixed_count > 5:
        print(f"  ... and {fixed_count - 5} more")

    print(f"\nTotal artifact locations fixed: {fixed_count}")
    print(f"Invalid results removed: {removed_count}")
    print(f"Final result count: {len(sarif['runs'][0]['results'])}")

    with open(output_file, "w") as f:
        json.dump(sarif, f, indent=2)

    print(f"✓ Fixed SARIF saved to {output_file}")

if __name__ == "__main__":
    main()
