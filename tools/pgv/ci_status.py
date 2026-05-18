#!/usr/bin/env python3
"""Query pdtd-phase-b-verify job status from recent workflow runs."""

import json
import subprocess
import sys

REPO = "markpippins/nexus-go-ccnf-ref"
WORKFLOW = "conformance.yml"
JOB_NAME = "PDTD: Phase B verify"
LIMIT = 20


def gh(*args):
    result = subprocess.run(
        ["gh", "api"] + list(args),
        capture_output=True, text=True, timeout=30
    )
    if result.returncode != 0:
        return None
    return json.loads(result.stdout)


def main():
    branch = subprocess.run(
        ["git", "rev-parse", "--abbrev-ref", "HEAD"],
        capture_output=True, text=True
    ).stdout.strip()

    # Get recent workflow runs across all branches
    runs = gh(
        f"/repos/{REPO}/actions/workflows/{WORKFLOW}/runs?per_page=100"
    )
    if runs is None or "workflow_runs" not in runs:
        print(f"  Branch: {branch}")
        print(f"  No conformance.yml runs found")
        print()
        print(f"  Counter: 0 / 7")
        return

    workflow_runs = runs["workflow_runs"]
    # Only count runs on main/master branches
    workflow_runs = [r for r in workflow_runs
                     if r.get("head_branch") in ("main", "master")]
    print(f"  Branch: {branch}")
    print(f"  Runs on main/master: {len(workflow_runs)}")

    if not workflow_runs:
        print()
        print(f"  Counter: 0 / 7")
        return

    # Calculate consecutive full-matrix passes (6 jobs = 3 OS × 2 Go)
    full_passes = 0
    MATRIX_SIZE = 6
    seen_run_ids = set()
    for run in workflow_runs:
        run_id = run["id"]
        if run_id in seen_run_ids:
            continue
        seen_run_ids.add(run_id)

        jobs_data = gh(f"/repos/{REPO}/actions/runs/{run_id}/jobs?per_page=100")
        if jobs_data is None:
            break

        phase_b_jobs = [j for j in jobs_data.get("jobs", [])
                        if JOB_NAME in j.get("name", "")]

        MATRIX_SIZE = 6  # 3 OS × 2 Go versions
        if len(phase_b_jobs) == MATRIX_SIZE and all(
            j.get("conclusion") == "success" and j.get("status") == "completed"
            for j in phase_b_jobs
        ):
            full_passes += 1
        else:
            break

    print()
    print(f"  Full matrix passes ({MATRIX_SIZE}/{MATRIX_SIZE}): {full_passes}")
    print(f"  Counter: {full_passes} / 7")

    if full_passes >= 7:
        print()
        print(f"  ✓✓✓ WINDOW COMPLETE — ready for PDTD_PHASE_B_ACTIVATE")
    elif full_passes > 0:
        print(f"  Need {7 - full_passes} more consecutive full-matrix passes")
        print(f"  Window resets on frozen component changes")


if __name__ == "__main__":
    main()
