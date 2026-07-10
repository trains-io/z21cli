package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/trains-io/z21.go/protocol"
)

type subscription struct {
	Name string
	Flag uint32
}

var subscriptionNamesByFlag = map[uint32]string{
	protocol.BroadcastFlagXpressNet:      "XPRESSNET_UPDATES",
	protocol.BroadcastFlagRBus:           "RBUS_UPDATES",
	protocol.BroadcastFlagRailComLegacy:  "RAILCOM_LEGACY_UPDATES",
	protocol.BroadcastFlagFastClock:      "FAST_CLOCK_UPDATES",
	protocol.BroadcastFlagSystemState:    "SYSTEM_UPDATES",
	protocol.BroadcastFlagAllLocoInfo:    "ALL_LOCO_INFO_UPDATES",
	protocol.BroadcastFlagCANBooster:     "CAN_BOOSTER_UPDATES",
	protocol.BroadcastFlagRailCom:        "RAILCOM_UPDATES",
	protocol.BroadcastFlagCANDetector:    "CAN_DETECTOR_UPDATES",
	protocol.BroadcastFlagLocoNetGeneral: "LOCONET_GENERAL_UPDATES",
	protocol.BroadcastFlagLocoNetLoco:    "LOCONET_LOCO_UPDATES",
	protocol.BroadcastFlagLocoNetTurnout: "LOCONET_TURNOUT_UPDATES",
	protocol.BroadcastFlagLocoNetOcc:      "LOCONET_OCC_UPDATES",
}

var knownSubscriptions = initKnownSubscriptions()

func initKnownSubscriptions() []subscription {
	subs := make([]subscription, 0, len(protocol.KnownBroadcastFlags))
	for _, flag := range protocol.KnownBroadcastFlags {
		name, ok := subscriptionNamesByFlag[flag.Value]
		if !ok {
			panic(fmt.Sprintf("missing subscription name for broadcast flag %#x", flag.Value))
		}
		subs = append(subs, subscription{Name: name, Flag: flag.Value})
	}
	return subs
}

func lookupSubscription(name string) (uint32, error) {
	key := strings.ToUpper(strings.TrimSpace(name))
	for _, sub := range knownSubscriptions {
		if sub.Name == key {
			return sub.Flag, nil
		}
	}
	return 0, fmt.Errorf("unknown subscription %q (known: %s)", name, strings.Join(subscriptionNames(), ", "))
}

func subscriptionNames() []string {
	names := make([]string, len(knownSubscriptions))
	for i, sub := range knownSubscriptions {
		names[i] = sub.Name
	}
	return names
}

func subscriptionDescription(flag uint32) string {
	for _, known := range protocol.KnownBroadcastFlags {
		if known.Value == flag {
			return known.Description
		}
	}
	return ""
}

func subscriptionHelpText() string {
	var b strings.Builder
	for _, sub := range knownSubscriptions {
		fmt.Fprintf(&b, "  %-24s %s\n", sub.Name, subscriptionDescription(sub.Flag))
	}
	return strings.TrimRight(b.String(), "\n")
}

func activeSubscriptions(mask uint32) []string {
	var active []string
	for _, sub := range knownSubscriptions {
		if protocol.HasBroadcastFlag(mask, sub.Flag) {
			active = append(active, sub.Name)
		}
	}
	sort.Strings(active)
	return active
}
