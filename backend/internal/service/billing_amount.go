package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// BillingPointScale stores 1 point as 1,000,000 internal units.
	BillingPointScale int64 = 1_000_000
	// LegacyBillingMilliPointScale stores the previous milli-point precision.
	LegacyBillingMilliPointScale int64 = 1_000
	billingAmountDecimals              = 2
	displayPointsDivisor               = 100
	billingCentScale                   = BillingPointScale / displayPointsDivisor
)

type BillingAmountInput int64

func (a *BillingAmountInput) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	var raw string
	if trimmed[0] == '"' {
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return err
		}
	} else {
		raw = string(trimmed)
	}

	parsed, err := ParseBillingAmountString(raw)
	if err != nil {
		return err
	}
	*a = BillingAmountInput(parsed)
	return nil
}

func (a BillingAmountInput) MicroPoints() int64 {
	return int64(a)
}

func BillingWholePoints(points int64) int64 {
	return points * BillingPointScale
}

func ParseBillingAmountString(raw string) (int64, error) {
	trimmed, sign, err := trimSignedBillingAmount(raw)
	if err != nil {
		return 0, err
	}

	wholePart, fractionPart, err := splitBillingAmountParts(raw, trimmed)
	if err != nil {
		return 0, err
	}

	whole, err := parseBillingWholePart(raw, wholePart)
	if err != nil {
		return 0, err
	}
	fraction, err := parseBillingFraction(raw, fractionPart)
	if err != nil {
		return 0, err
	}

	return combineBillingAmount(raw, sign, whole, fraction)
}

func trimSignedBillingAmount(raw string) (trimmed string, sign int64, err error) {
	trimmed = strings.TrimSpace(raw)
	if trimmed == "" {
		return "", 0, errors.New("billing amount is empty")
	}

	sign = 1
	if trimmed[0] == '+' || trimmed[0] == '-' {
		if trimmed[0] == '-' {
			sign = -1
		}
		trimmed = trimmed[1:]
	}
	if trimmed == "" {
		return "", 0, errors.New("billing amount is empty")
	}

	return trimmed, sign, nil
}

func splitBillingAmountParts(raw, trimmed string) (wholePart, fractionPart string, err error) {
	wholePart, fractionPart, hasDot := strings.Cut(trimmed, ".")
	if hasDot && strings.Contains(fractionPart, ".") {
		return "", "", fmt.Errorf("invalid billing amount %q", raw)
	}
	if wholePart == "" {
		wholePart = "0"
	}
	return wholePart, fractionPart, nil
}

func parseBillingWholePart(raw, wholePart string) (int64, error) {
	whole, err := strconv.ParseInt(wholePart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid billing amount %q", raw)
	}
	return whole, nil
}

func parseBillingFraction(raw, fractionPart string) (int64, error) {
	if len(fractionPart) > billingAmountDecimals {
		return 0, fmt.Errorf("billing amount %q supports at most %d decimal places", raw, billingAmountDecimals)
	}
	for _, ch := range fractionPart {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid billing amount %q", raw)
		}
	}
	for len(fractionPart) < billingAmountDecimals {
		fractionPart += "0"
	}
	if fractionPart == "" {
		return 0, nil
	}

	fraction, err := strconv.ParseInt(fractionPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid billing amount %q", raw)
	}
	return fraction, nil
}

func combineBillingAmount(raw string, sign, whole, fraction int64) (int64, error) {
	total := new(big.Int).Mul(big.NewInt(whole), big.NewInt(BillingPointScale))
	total.Add(total, big.NewInt(fraction*billingCentScale))
	if sign < 0 {
		total.Neg(total)
	}
	if !total.IsInt64() {
		return 0, fmt.Errorf("billing amount %q overflows int64", raw)
	}
	return total.Int64(), nil
}

func FormatBillingAmountConfigValue(amount int64) string {
	return strconv.FormatInt(amount, 10)
}

func ParseBillingAmountConfigValue(raw string, def int64) int64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return def
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return def
	}
	return parsed
}

func roundBillingMicroToCents(v int64) int64 {
	if v >= 0 {
		return (v + billingCentScale/2) / billingCentScale
	}
	return (v - billingCentScale/2) / billingCentScale
}

func ToDisplayPoints(totalMicro int64) float64 {
	if totalMicro == 0 {
		return 0
	}
	return float64(roundBillingMicroToCents(totalMicro)) / displayPointsDivisor
}

func quantityToRat(q resource.Quantity) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(q.AsDec().String())
	if !ok {
		return nil, fmt.Errorf("failed to parse quantity %q", q.String())
	}
	return r, nil
}

func ratFloorToInt64(r *big.Rat) int64 {
	if r == nil || r.Sign() <= 0 {
		return 0
	}
	n := new(big.Int).Quo(r.Num(), r.Denom())
	if !n.IsInt64() {
		return math.MaxInt64
	}
	return n.Int64()
}
