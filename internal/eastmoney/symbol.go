package eastmoney

import (
	"fmt"
	"strings"
)

// toSecID converts market:code symbol (SH:600519, SZ:000001, HK:00700) to EastMoney secid.
func toSecID(qosSymbol string) (string, error) {
	parts := strings.SplitN(qosSymbol, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid symbol format: %s", qosSymbol)
	}
	market := strings.ToUpper(parts[0])
	code := strings.ToUpper(parts[1])

	switch market {
	case "SH":
		return "1." + code, nil
	case "SZ":
		return "0." + code, nil
	case "HK":
		return "116." + code, nil
	default:
		return "", fmt.Errorf("unsupported market: %s", market)
	}
}

// fromSecID converts EastMoney secid back to market:code symbol.
func fromSecID(secID string) string {
	parts := strings.SplitN(secID, ".", 2)
	if len(parts) != 2 {
		return secID
	}
	switch parts[0] {
	case "1":
		return "SH:" + parts[1]
	case "0":
		return "SZ:" + parts[1]
	case "116":
		return "HK:" + parts[1]
	default:
		return secID
	}
}

// ktToKlt converts K-line type to EastMoney klt parameter.
func ktToKlt(kt int) int {
	switch kt {
	case 1:
		return 1   // 1m
	case 5:
		return 5   // 5m
	case 15:
		return 15  // 15m
	case 30:
		return 30  // 30m
	case 60:
		return 60  // 1h
	case 120:
		return 120 // 2h
	case 240:
		return 240 // 4h
	case 1001:
		return 101 // daily
	case 1007:
		return 102 // weekly
	case 1030:
		return 103 // monthly
	default:
		return 101 // default to daily
	}
}
