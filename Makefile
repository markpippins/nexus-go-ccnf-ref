.PHONY: test conformance cross-platform fuzz oracle ci clean \
        r2 r2-collisions r2-stress r3 r3-roundtrip r4 r5 r6 r8

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
	  "R6:Cross-platform build|go build -o /dev/null ./conformance && go build ./ccnf/... && go build ./replay/..." \
	  "R8:Rust verifier|cargo build --release --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml && cargo run --release --manifest-path ../../../rust/wrp/ccnf-verifier/Cargo.toml -- ../../../go/wrp/ccnf-ref/vectors/v1"; \
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

ci: test conformance cross-platform fuzz r2 r3 r4 r5 r8
	@echo "--- CI gate: all OK ---"

clean:
	rm -rf ./bin

.DEFAULT_GOAL := test
