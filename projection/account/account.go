// DependsOn: runtime/rehydrate/snapshot
package account

import (
	"strconv"
	"strings"

	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/snapshot"
)

type AccountBalance struct {
	ID      string
	Balance uint64
}

func (AccountBalance) Kind() string { return "account_balance" }

type cacheEntry struct {
	balance uint64
}

type AccountProjection struct {
	cache map[string]cacheEntry
}

func BuildAccountProjection(snap snapshot.ReplaySnapshot) AccountProjection {
	cache := make(map[string]cacheEntry)
	it := snap.Scan([]byte("acct:"))
	defer it.Close()
	for it.Next() {
		k := string(it.Key())
		v := it.Value()
		balance, err := strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			continue
		}
		id := strings.TrimPrefix(k, "acct:")
		cache[id] = cacheEntry{balance: balance}
	}
	return AccountProjection{cache: cache}
}

func (p AccountProjection) GetBalance(id string) (uint64, bool) {
	entry, ok := p.cache[id]
	return entry.balance, ok
}

func (p AccountProjection) AllBalances() []AccountBalance {
	var result []AccountBalance
	for id, entry := range p.cache {
		result = append(result, AccountBalance{ID: id, Balance: entry.balance})
	}
	return result
}

func (p AccountProjection) Count() int {
	return len(p.cache)
}
