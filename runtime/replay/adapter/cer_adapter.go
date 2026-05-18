package adapter

import (
	"fmt"

	"github.com/anomalyco/nexus-ccnf-ref/replay"
	ccnfreplay "github.com/anomalyco/nexus-ccnf-ref/runtime/replay"
)

func CEREventsToReplay(events []replay.CEREvent) []ccnfreplay.ReplayEvent {
	result := make([]ccnfreplay.ReplayEvent, 0, len(events))
	for _, e := range events {
		delta := ccnfreplay.StateDelta{
			Writes: make(map[ccnfreplay.StateKey]ccnfreplay.StateValue),
		}
		for k, v := range e.StateDelta {
			switch val := v.(type) {
			case string:
				delta.Writes[ccnfreplay.StateKey(k)] = ccnfreplay.StateValue(val)
			case []byte:
				delta.Writes[ccnfreplay.StateKey(k)] = val
			default:
				delta.Writes[ccnfreplay.StateKey(k)] = ccnfreplay.StateValue(fmt.Sprintf("%v", val))
			}
		}

		re := ccnfreplay.ReplayEvent{
			EventID: e.EventID,
			Delta:   delta,
		}

		if e.CausalChainID != "" {
			re.PrevEventID = e.CausalChainID
		}

		result = append(result, re)
	}
	return result
}

func CEREventsToReplayInput(
	events []replay.CEREvent,
	cerRootHash, traceRootHash, replayBindingHash string,
	ccnfVersion, semanticsVersion int,
) ccnfreplay.ReplayInput {
	replayEvents := CEREventsToReplay(events)
	return ccnfreplay.ReplayInput{
		Events:            replayEvents,
		CerRootHash:       cerRootHash,
		TraceRootHash:     traceRootHash,
		ReplayBindingHash: replayBindingHash,
		CCNFVersion:       ccnfVersion,
		SemanticsVersion:  semanticsVersion,
		EventCount:        uint64(len(replayEvents)),
	}
}
