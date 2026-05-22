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
		// Only zero-pad numeric codes (stock symbols); leave index tickers like HSI as-is.
		if isAllDigits(code) {
			for len(code) < 5 {
				code = "0" + code
			}
		}
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
		return 240  // daily
	case 1007:
		return 1200 // weekly
	case 1030:
		return 7200 // monthly
	default:
		return 240
	}
}

// toEastmoneySecID converts market:code symbol to Eastmoney secid format (116.00700 for HK).
func toEastmoneySecID(qosSymbol string) (string, error) {
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
		if isAllDigits(code) {
			for len(code) < 5 {
				code = "0" + code
			}
			return "116." + code, nil
		}
		return "100." + code, nil // HK index (e.g. HSI, HSCEI)
	case "US":
		return "100." + code, nil // global index (e.g. IXIC, DJI, SPX)
	default:
		return "", fmt.Errorf("eastmoney: unsupported market %s", market)
	}
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// ktToEastmoneyKlt converts K-line type to Eastmoney klt parameter.
// 1=1min, 5=5min, 15=15min, 30=30min, 60=60min, 101=daily, 102=weekly, 103=monthly.
func ktToEastmoneyKlt(kt int) int {
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
	case 120, 240:
		return 60 // no 120/240min in Eastmoney, fall back to 60min
	case 1001:
		return 101
	case 1007:
		return 102
	case 1030:
		return 103
	default:
		return 101
	}
}
