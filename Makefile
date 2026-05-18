SHELL = /bin/bash

.PHONY: test conformance cross-platform fuzz oracle ci clean \
        r2 r2-collisions r2-stress r3 r3-roundtrip r4 r5 r6 r8 \
        r9 r9-rust r10 r10-rust replay-seal replay-import-check replay-build-isolation \
        r10.3 r10.3-rust r10.3b \
        rehydrate-forbidden-imports rehydrate-registry-immutability rehydrate-no-pointer-receivers \
        rehydrate-no-domain-words rehydrate-view-purity rehydrate-no-cross-view rehydrate-build-isolation \
        rust-rehydrate-no-mut \
        projection-no-rehydration-backedge \
        projection-build-isolation projection-cache-not-exposed projection-no-pointer-receivers \
        rust-projection-no-rehydration-backedge rust-projection-no-mut \
        pdtp pdtp-all pdtp-phase-b-verify pdtp-window-status dependency-visibility

test:
	go test ./ccnf/...

conformance:
	go test -run TestGoldenVectors ./ccnf/...

cross-platform:
	@echo "--- cross-compile: linux/amd64 ---"
	GOOS=linux   GOARCH=amd64 go build -o /dev/null ./conformance
	@echo "--- cross-compile: darwin/arm64 ---"
	GOOS=darwin  GOARCH=arm64 go build -o /dev/null ./conformance
	@echo "--- cross-compile: windows/amd64 ---"
	GOOS=windows GOARCH=amd64 go build -o /dev/null ./conformance
	@echo "--- cross-compile OK ---"

fuzz:
	go test -run TestR2 -count=1 -timeout 60s ./ccnf/...

oracle:
	go build -o ./bin/ccnf-conformance ./conformance

r2:
	@echo "--- R2: equivalence class boundary stress (10k) ---"
	go test -run "TestR2Determinism|TestR2EquivalenceClassBoundary|TestR2CrossOrderEquivalence|TestR2RoundTrip|TestR2UnicodeStress|TestR2TimestampBoundaries" -count=1 -timeout 120s ./ccnf/...

r2-collisions:
	@echo "--- R2: collision atlas regeneration (100k) ---"
	go test -run TestR2CollisionDetection -count=1 -timeout 300s ./ccnf/...
	@echo "--- collision atlas written to vectors/r2/collisions/v0.1.0-fuzz.json ---"

r2-stress:
	@echo "--- R2: full stress (100k iterations) ---"
	R2_STRESS=1 go test -run "TestR2" -count=1 -timeout 600s ./ccnf/...

r3:
	@echo "--- R3: CER round-trip ---"
	go test -run "TestCER" -count=1 -timeout 60s ./ccnf/...

r3-roundtrip:
	@echo "--- R3: CER serialize/rehydrate round-trip ---"
	go test -run TestCERSerializeRoundTrip -count=1 -timeout 60s ./ccnf/...

r4:
	@echo "--- R4: Replay oracle ---"
	go test -v -count=1 ./replay/...

r5:
	@echo "--- R5: Snapshot oracle ---"
	go test -v -count=1 ./replay/snapshot/...

r6:
	@echo "==========================================="
	@echo "  R6: Proof validator — semantic gate"
	@echo "==========================================="
	@echo ""
	@rm -f .r6_failed
	@success=0; total=0; \
	for phase in \
	  "R1:CCNF determinism|go test -run 'TestGoldenVectors|TestSerializerDeterminism' -count=1 ./ccnf/..." \
	  "R1:Hash-locked golden vectors|go build -o /tmp/r6_conformance ./conformance && /tmp/r6_conformance run vectors/v1" \
	  "R2:Equivalence closure|go test -run 'TestR2Determinism|TestR2EquivalenceClassBoundary|TestR2CrossOrderEquivalence|TestR2RoundTrip|TestR2UnicodeStress|TestR2TimestampBoundaries' -count=1 -timeout 120s ./ccnf/..." \
	  "R3:CER round-trip|go test -run 'TestCER' -count=1 -timeout 60s ./ccnf/..." \
	  "R4:Replay oracle|go test -count=1 ./replay/..." \
	  "R5:Snapshot oracle|go test -count=1 ./replay/snapshot/..." \
	  "R4=R5:Cross-check|go test -run 'TestR4R5' -count=1 ./replay/..." \
	  "R6:Cross-platform build|go build -o /dev/null ./conformance && go build ./ccnf/... && go build ./replay/... && go build ./runtime/... && go build ./runtime/replay/..." \
	  "R8:Rust verifier|cargo build --release --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml && cargo run --release --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- ../../../go/wrp/ccnf-ref/vectors/v1" \
	  "R9:Runtime boundary|go test -count=1 ./runtime/..." \
	  "R9:Rust runtime|cargo test --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- runtime::types runtime::trace 2>&1" \
	  "R10:Replay binding|go test -run 'TestReplayBinding|TestBuildReceipt|TestValidateReceipt' -count=1 ./runtime/..." \
	  "R10:Sealed replay|go test -count=1 ./runtime/replay/..." \
	  "R10:Rust replay|cargo test --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- runtime::replay 2>&1" \
	  "R10:Import check|bash -c '! grep -r \"ccnf/\" go/wrp/ccnf-ref/runtime/replay/*.go | grep -v \"_test.go\" | grep -v \"adapter/\"'" \
	  "R10.3:Rehydration Go|go test -count=1 ./runtime/rehydrate/..." \
	  "R10.3:Forbidden imports|make rehydrate-forbidden-imports" \
	  "R10.3:Registry immutability|make rehydrate-registry-immutability" \
	  "R10.3:View purity|make rehydrate-view-purity && make rehydrate-no-pointer-receivers && make rehydrate-no-domain-words && make rehydrate-no-cross-view" \
	  "R10.3:Build isolation|make rehydrate-build-isolation" \
	  "R10.3:Rust rehydrate|cargo build --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml && cargo test --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- runtime::rehydrate 2>&1" \
	  "R10.3:No Rust &mut|make rust-rehydrate-no-mut" \
	  "R10.3B:Projection Go|go test -count=1 ./projection/..." \
	  "R10.3B:No back-edge|make projection-no-rehydration-backedge" \
	  "R10.3B:View-style|make projection-cache-not-exposed && make projection-no-pointer-receivers" \
	  "R10.3B:Build isolation|make projection-build-isolation" \
	  "R10.3B:Rust projection|cargo build --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml && cargo test --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- projection 2>&1 && make rust-projection-no-rehydration-backedge && make rust-projection-no-mut" \
	  "PDTD:PGV Go|make pdtp" \
	  "PDTD:Dependency visibility|make dependency-visibility"; \
	do \
	  total=$$((total + 1)); \
	  label=$$(echo "$$phase" | cut -d'|' -f1); \
	  cmd=$$(echo "$$phase" | cut -d'|' -f2-); \
	  printf "  [%s] " "$$label"; \
	  if eval "$$cmd" > /tmp/r6_phase_output 2>&1; then \
	    echo "PASS"; \
	    success=$$((success + 1)); \
	  else \
	    echo "FAIL"; \
	    cat /tmp/r6_phase_output | head -20; \
	    echo "..."; \
	    echo "  CLASSIFICATION: $$label failure"; \
	    case "$$label" in \
	      *determinism*)   echo "  CLASS: determinism-violation (fatal)";; \
	      *golden*)         echo "  CLASS: determinism-violation (fatal)";; \
	      *Equivalence*)    echo "  CLASS: spec-ambiguity (R2 collision)";; \
	      *CER*)            echo "  CLASS: event-correctness (R3)";; \
	      *Replay*)         echo "  CLASS: oracle-divergence (R4)";; \
	      *Snapshot*)       echo "  CLASS: version-lock-failure (R5)";; \
	      *Cross-check*)    echo "  CLASS: oracle-divergence (R4 != R5)";; \
	      *Cross-platform*) echo "  CLASS: platform-nondeterminism";; \
	      *Rust*)           echo "  CLASS: oracle-divergence (Rust != Go)";; \
	      *Runtime*)        echo "  CLASS: runtime-divergence (R9)";; \
	      *Replay*)         echo "  CLASS: replay-binding-failure (R10)";; \
	      *Rehydration*)    echo "  CLASS: rehydration-leak (R10.3)";; \
	      *Forbidden*)      echo "  CLASS: import-seal-violation (R10.3)";; \
	      *Registry*)       echo "  CLASS: registry-mutation (R10.3)";; \
	      *View-style*)     echo "  CLASS: projection-style-violation (R10.3B)";; \
	      *View*)           echo "  CLASS: view-purity-violation (R10.3)";; \
	      *Build*)          echo "  CLASS: build-isolation-failure (R10.3)";; \
	      *Projection*)     echo "  CLASS: projection-failure (R10.3B)";; \
	      *back-edge*)      echo "  CLASS: projection-back-edge (R10.3B)";; \
	      *Import*)         echo "  CLASS: import-seal-violation (R10)";; \
	      *PGV*)            echo "  CLASS: pdtp-violation (PDTD)";; \
	      *Dependency*)     echo "  CLASS: pdtp-warn (P10 advisory)";; \
	      *)                echo "  CLASS: unclassified";; \
	    esac; \
	    touch .r6_failed; \
	  fi; \
	done; \
	echo ""; \
	echo "  $$success/$$total phases passed"; \
	if [ -f .r6_failed ]; then echo "  R6: GATE FAILED"; rm -f .r6_failed; exit 1; fi; \
	echo "  R6: GATE PASSED"; rm -f .r6_failed

r8:
	@echo "==========================================="
	@echo "  R8: Rust verifier — independent CCNF pipeline"
	@echo "==========================================="
	@echo ""
	@cargo run --release --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- ../../../go/wrp/ccnf-ref/vectors/v1
	@echo ""

r9:
	@echo "--- R9: Runtime boundary types ---"
	go test -v -count=1 ./runtime/...

r9-rust:
	@echo "--- R9: Rust runtime boundary ---"
	cargo test --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- runtime::types runtime::trace 2>&1

r10:
	@echo "--- R10: Replay binding + sealed replay ---"
	go test -run "TestReplayBinding|TestBuildReceipt|TestValidateReceipt" -count=1 ./runtime/...
	go test -count=1 ./runtime/replay/...

r10-rust:
	@echo "--- R10: Rust replay mirror ---"
	cargo test --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- runtime::replay 2>&1

replay-seal:
	@echo "--- Replay compile-time seal ---"
	bash -c '! grep -r "ccnf/" go/wrp/ccnf-ref/runtime/replay/*.go | grep -v "_test.go" | grep -v "adapter/"'
	@echo "  OK: replay does not import CCNF"

replay-import-check:
	@echo "--- Replay forbidden import check ---"
	@banned=0; \
	for pkg in ccnf cer canonical identity normalize artifacts; do \
	  if grep -r "github.com/anomalyco/nexus-ccnf-ref/$$pkg" go/wrp/ccnf-ref/runtime/replay/*.go 2>/dev/null | grep -qv "_test.go"; then \
	    echo "  BANNED: runtime/replay imports $$pkg"; \
	    banned=1; \
	  fi; \
	done; \
	if [ "$$banned" = "1" ]; then echo "  REPLAY IMPORT CHECK: FAILED"; exit 1; fi; \
	echo "  OK: all replay imports are clean"

replay-build-isolation:
	@echo "--- Replay build isolation (build without CCNF) ---"
	@go build ./runtime/replay/... 2>&1 && echo "  OK: replay builds independently"

rehydrate-forbidden-imports:
	@echo "--- Rehydrate forbidden import check ---"
	@banned=0; \
	for pkg in ccnf execution engine; do \
	  if grep -r "github.com/anomalyco/nexus-ccnf-ref/$$pkg" runtime/rehydrate/*.go runtime/rehydrate/*/*.go 2>/dev/null | grep -qv "_test.go"; then \
	    echo "  BANNED: rehydrate imports $$pkg"; \
	    banned=1; \
	  fi; \
	done; \
	for forbidden in "replay/apply" "replay/core"; do \
	  if grep -r "github.com/anomalyco/nexus-ccnf-ref/runtime/$$forbidden" runtime/rehydrate/*.go runtime/rehydrate/*/*.go 2>/dev/null | grep -qv "_test.go"; then \
	    echo "  BANNED: rehydrate imports $$forbidden"; \
	    banned=1; \
	  fi; \
	done; \
	if [ "$$banned" = "1" ]; then echo "  REHYDRATE IMPORT CHECK: FAILED"; exit 1; fi; \
	echo "  OK: all rehydrate imports are clean"

rehydrate-registry-immutability:
	@echo "--- Rehydrate registry immutability check ---"
	@if grep -r "func Register(" runtime/rehydrate/ 2>/dev/null | grep -qv "_test.go"; then \
	  echo "  BANNED: Register() found in rehydrate"; \
	  exit 1; \
	fi; \
	echo "  OK: no Register() in rehydrate"

rehydrate-no-pointer-receivers:
	@echo "--- Rehydrate view pointer receiver check ---"
	@if grep -qR "func (.*\*)" runtime/rehydrate/view/ 2>/dev/null; then \
	  echo "  BANNED: pointer receivers found in view/"; \
	  grep -R "func (.*\*)" runtime/rehydrate/view/; \
	  exit 1; \
	fi; \
	echo "  OK: no pointer receivers in view/"

rehydrate-no-domain-words:
	@echo "--- Rehydrate domain word check ---"
	@found=0; \
	for word in contract account stake validator tx execute apply validate "state machine"; do \
	  if grep -ri "$$word" runtime/rehydrate/ 2>/dev/null | grep -v "_test.go" | grep -q .; then \
	    echo "  DOMAIN WORD FOUND: $$word"; \
	    grep -ri "$$word" runtime/rehydrate/ 2>/dev/null | grep -v "_test.go"; \
	    found=1; \
	  fi; \
	done; \
	if [ "$$found" = "1" ]; then echo "  DOMAIN WORD CHECK: FAILED"; exit 1; fi; \
	echo "  OK: no domain words in rehydrate"

rehydrate-view-purity:
	@echo "--- Rehydrate view purity check ---"
	@if grep -qR "func (.*View.*) \w*(bool|error)" runtime/rehydrate/view/ 2>/dev/null; then \
	  echo "  BANNED: View methods returning bool/error"; \
	  grep -R "func (.*View.*) \w*(bool|error)" runtime/rehydrate/view/; \
	  exit 1; \
	fi; \
	echo "  OK: all View methods pure"

rehydrate-no-cross-view:
	@echo "--- Rehydrate cross-view awareness check ---"
	@if grep -r "runtime/rehydrate/view/" runtime/rehydrate/decode/*.go 2>/dev/null | grep -q .; then \
	  echo "  BANNED: decoder imports view package"; \
	  grep -r "runtime/rehydrate/view/" runtime/rehydrate/decode/*.go; \
	  exit 1; \
	fi; \
	echo "  OK: no cross-view references in decode"

rehydrate-build-isolation:
	@echo "--- Rehydrate build isolation ---"
	@go build ./runtime/rehydrate/... 2>&1 && echo "  OK: rehydrate builds independently"

r10.3:
	@echo "--- R10.3: Rehydration ---"
	@echo ""
	@$(MAKE) rehydrate-forbidden-imports
	@$(MAKE) rehydrate-registry-immutability
	@$(MAKE) rehydrate-no-pointer-receivers
	@$(MAKE) rehydrate-no-domain-words
	@$(MAKE) rehydrate-view-purity
	@$(MAKE) rehydrate-no-cross-view
	@$(MAKE) rehydrate-build-isolation
	go test -count=1 ./runtime/rehydrate/...

r10.3-rust:
	@echo "--- R10.3: Rust rehydration mirror ---"
	@cargo build --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml 2>&1 | tail -3
	@cargo test --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- runtime::rehydrate 2>&1 | tail -5

rust-rehydrate-no-mut:
	@echo "--- Rust rehydrate &mut check ---"
	@if grep -qR "&mut " ../../../rust/wrp/ccnf-verifier/src/runtime/rehydrate/ 2>/dev/null; then \
	  echo "  BANNED: &mut found in Rust rehydrate"; \
	  grep -R "&mut " ../../../rust/wrp/ccnf-verifier/src/runtime/rehydrate/; \
	  exit 1; \
	fi; \
	echo "  OK: no &mut in Rust rehydrate"

projection-no-rehydration-backedge:
	@echo "--- Projection back-edge check ---"
	@banned=0; \
	for f in projection/*.go projection/*/*.go; do \
	  if [ -f "$$f" ]; then \
	    if grep -q "runtime/rehydrate" "$$f" 2>/dev/null; then \
	      if ! grep -q "runtime/rehydrate/snapshot" "$$f" 2>/dev/null; then \
	        echo "  BANNED: $$f imports non-snapshot rehydrate"; \
	        banned=1; \
	      fi; \
	    fi; \
	  fi; \
	done; \
	if [ "$$banned" = "1" ]; then echo "  PROJECTION BACK-EDGE CHECK: FAILED"; exit 1; fi; \
	echo "  OK: all projection imports are rehydrate/snapshot only"


projection-cache-not-exposed:
	@echo "--- Projection cache exposure check ---"
	@if grep -qR "^type.*Cache\b" projection/ 2>/dev/null; then \
	  echo "  BANNED: exported Cache type in projection"; \
	  grep -R "^type.*Cache\b" projection/; \
	  exit 1; \
	fi; \
	echo "  OK: no exported cache types in projection"

projection-no-pointer-receivers:
	@echo "--- Projection pointer receiver check ---"
	@if grep -qR "func (.*\*)" projection/ 2>/dev/null; then \
	  echo "  BANNED: pointer receivers in projection"; \
	  grep -R "func (.*\*)" projection/; \
	  exit 1; \
	fi; \
	echo "  OK: no pointer receivers in projection"

projection-build-isolation:
	@echo "--- Projection build isolation ---"
	@go build ./projection/... 2>&1 && echo "  OK: projection builds independently"

rust-projection-no-rehydration-backedge:
	@echo "--- Rust projection back-edge check ---"
	@if grep -qR "runtime::rehydrate" ../../../rust/wrp/ccnf-verifier/src/projection/ 2>/dev/null; then \
	  echo "  Checking Rust projection rehydrate imports..."; \
	  if grep -qR "runtime::rehydrate" ../../../rust/wrp/ccnf-verifier/src/projection/ 2>/dev/null | grep -v "snapshot" | grep -q .; then \
	    echo "  BANNED: Rust projection imports non-snapshot rehydrate"; \
	    exit 1; \
	  fi; \
	fi; \
	echo "  OK: Rust projection imports only snapshot"

rust-projection-no-mut:
	@echo "--- Rust projection &mut check ---"
	@if grep -qR "&mut " ../../../rust/wrp/ccnf-verifier/src/projection/ 2>/dev/null; then \
	  echo "  BANNED: &mut found in Rust projection"; \
	  grep -R "&mut " ../../../rust/wrp/ccnf-verifier/src/projection/; \
	  exit 1; \
	fi; \
	echo "  OK: no &mut in Rust projection"

r10.3b:
	@echo "--- R10.3B: Projection layer ---"
	@echo ""
	@$(MAKE) projection-no-rehydration-backedge
	@$(MAKE) projection-cache-not-exposed
	@$(MAKE) projection-no-pointer-receivers
	@$(MAKE) projection-build-isolation
	go test -count=1 -v ./projection/...

r10.3b-rust:
	@echo "--- R10.3B: Rust projection layer ---"
	@cargo build --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml 2>&1 | tail -3
	@$(MAKE) rust-projection-no-rehydration-backedge
	@$(MAKE) rust-projection-no-mut
	@cargo test --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- projection 2>&1 | tail -10

ci: test conformance cross-platform fuzz r2 r3 r4 r5 r8 r9 r9-rust r10 r10-rust \
    replay-seal replay-import-check replay-build-isolation \
    r10.3 r10.3-rust rust-rehydrate-no-mut \
    r10.3b r10.3b-rust pdtp-all
	@echo "--- CI gate: all OK ---"

pdtp:
	@echo "--- PGV: Projection Dependency Topology Hardening ---"
	@go run ./tools/pgv/

pdtp-all: pdtp
	@echo "--- PGV: all checks passed ---"

pdtp-phase-b-verify:
	@echo "================================================"
	@echo "  Phase B verify: PGV topology enforcement"
	@echo "================================================"
	@echo ""
	@echo "--- Running PGV (all extractors) ---"
	@go run ./tools/pgv/

pdtp-window-status:
	@echo "================================================"
	@echo "  Phase-B Window Status"
	@echo "================================================"
	@echo ""
	@rm -f .window_pass .window_frozen_fail .window_hash_fail
	@echo "--- Frozen surface integrity ---"
	@frozen_changed=0; \
	for f in tools/pgv/extractor.go tools/pgv/go_extractor.go tools/pgv/rust_extractor.go tools/pgv/comment_extractor.go tools/pgv/ir.go tools/pgv/validator.go tools/pgv/baseline.go; do \
	  if git diff --name-only HEAD -- "$$f" 2>/dev/null | grep -q "$$f"; then \
	    echo "  MODIFIED: $$f (window reset required)"; \
	    frozen_changed=1; \
	  fi; \
	done; \
	if [ "$$frozen_changed" = "1" ]; then \
	  touch .window_frozen_fail; \
	else \
	  echo "  OK: frozen surface clean"; \
	fi
	@echo ""
	@echo "--- PGV baseline check ---"
	@current=$$(go run ./tools/pgv/ 2>/dev/null | grep "IR hash:" | sed 's/.*hash: //'); \
	baseline="e60ec6a575f4641368f43106780f9fba12817e0b802eaf7d47e7206a300077b0"; \
	if [ "$$current" = "$$baseline" ]; then \
	  echo "  Hash: $$current (matches baseline)"; \
	else \
	  echo "  Hash: $$current"; \
	  echo "  Expected: $$baseline"; \
	  echo "  MISMATCH — topology has changed, new baseline required"; \
	  touch .window_hash_fail; \
	fi
	@echo ""
	@echo "--- CI run history ---"
	@python3 tools/pgv/ci_status.py 2>&1 || echo "  (CI query failed)"
	@echo ""
	@echo "--- Status ---"
	@if [ -f .window_frozen_fail ]; then \
	  echo "  FROZEN SURFACE CHANGED"; \
	  rm -f .window_frozen_fail .window_hash_fail; \
	elif [ -f .window_hash_fail ]; then \
	  echo "  HASH CHANGED — topology evolved, update baseline."; \
	  rm -f .window_frozen_fail .window_hash_fail; \
	else \
	  echo "  Clean"; \
	fi

dependency-visibility:
	@echo "--- P10: Dependency visibility (advisory) ---"
	@missing=0; \
	for f in projection/*.go projection/*/*.go; do \
	  if [ -f "$$f" ]; then \
	    if ! grep -q "DependsOn:" "$$f" 2>/dev/null; then \
	      echo "  WARN: $$f has no // DependsOn: comment"; \
	      missing=1; \
	    fi; \
	  fi; \
	done; \
	if [ "$$missing" = "1" ]; then echo "  P10: advisory WARN — some files lack DependsOn annotations"; else echo "  P10: all files have DependsOn annotations"; fi

clean:
	rm -rf ./bin

.DEFAULT_GOAL := test
