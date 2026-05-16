.PHONY: test conformance cross-platform fuzz oracle ci clean r2 r2-collisions r2-stress r3 r3-roundtrip r4

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

ci: test conformance cross-platform fuzz r2 r3 r4
	@echo "--- CI gate: all OK ---"

clean:
	rm -rf ./bin

.DEFAULT_GOAL := test
