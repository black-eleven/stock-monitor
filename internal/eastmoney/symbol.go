package eastmoney

import (
	"fmt"
	"strings"
)

// toSinaSymbol converts market:code symbol to Sina Finance format (sh600519, sz000001, hk00700).
func toSinaSymbol(qosSymbol string) (string, error) {
	parts := strings.SplitN(qosSymbol, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid symbol format: %s", qosSymbol)
	}
	market := strings.ToUpper(parts[0])
	code := strings.ToUpper(parts[1])

	switch market {
	case "SH":
		return "sh" + code, nil
	case "SZ":
		return "sz" + code, nil
	case "HK":
		return "hk" + code, nil
	default:
		return "", fmt.Errorf("unsupported market: %s", market)
	}
}

// ktToSinaScale converts K-line type to Sina Finance scale parameter.
func ktToSinaScale(kt int) int {
	switch kt {
	case 1:
		return 1
	case 5:
		return 5
	case 15:
		return 15
	case 30:
		return 30
	case 60:
		return 60
	case 120:
		return 120
	case 240:
		return 240
	case 1001:
		return 240 // daily
	case 1007:
		return 240 // weekly
	case 1030:
		return 240 // monthly
	default:
		return 240
	}
}

// toYahooSymbol converts market:code symbol to Yahoo Finance format (0700.HK, 600519.SS, 000001.SZ).
func toYahooSymbol(qosSymbol string) (string, error) {
	parts := strings.SplitN(qosSymbol, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid symbol format: %s", qosSymbol)
	}
	market := strings.ToUpper(parts[0])
	code := strings.ToUpper(parts[1])

	switch market {
	case "SH":
		return code + ".SS", nil
	case "SZ":
		return code + ".SZ", nil
	case "HK":
		return code + ".HK", nil
	default:
		return "", fmt.Errorf("unsupported market: %s", market)
	}
}
