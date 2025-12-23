package pricing

import (
	"math"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
)

// BlackScholesDigitalOption calculates the probability of a digital option
// finishing in the money using the Black-Scholes model
func BlackScholesDigitalOption(
	spot float64,
	barrier float64,
	timeToExpiry float64, // in years
	volatility float64,
	interestRate float64,
	quantoDrift float64,
	contractType pb.ContractType,
) float64 {
	// If already expired, probability is deterministic
	if timeToExpiry <= 0 {
		if contractType == pb.ContractType_CONTRACT_TYPE_CALL {
			if spot > barrier {
				return 1.0
			}
			return 0.0
		} else {
			if spot < barrier {
				return 1.0
			}
			return 0.0
		}
	}

	// Calculate d2 from Black-Scholes formula
	// d2 = [ln(S/K) + (r - q - σ²/2) * T] / (σ * √T)
	d2 := calculateD2(spot, barrier, timeToExpiry, volatility, interestRate, quantoDrift)

	// For a digital call: probability = N(d2)
	// For a digital put: probability = N(-d2)
	if contractType == pb.ContractType_CONTRACT_TYPE_CALL {
		return cumulativeNormalDistribution(d2)
	}

	return cumulativeNormalDistribution(-d2)
}

// calculateD2 computes the d2 parameter from the Black-Scholes formula
func calculateD2(spot, barrier, timeToExpiry, volatility, interestRate, quantoDrift float64) float64 {
	// d2 = [ln(S/K) + (r - q - σ²/2) * T] / (σ * √T)
	volatilitySquared := volatility * volatility
	sqrtTime := math.Sqrt(timeToExpiry)

	numerator := math.Log(spot/barrier) + (interestRate-quantoDrift-volatilitySquared/2.0)*timeToExpiry
	denominator := volatility * sqrtTime

	return numerator / denominator
}

// cumulativeNormalDistribution calculates the cumulative normal distribution N(x)
// This is the probability that a standard normal random variable is less than or equal to x
func cumulativeNormalDistribution(x float64) float64 {
	// Using the approximation from Abramowitz and Stegun
	// This provides accuracy to about 7 decimal places

	// Handle extreme values
	if x < -10 {
		return 0.0
	}
	if x > 10 {
		return 1.0
	}

	// Constants for the approximation
	const (
		a1 = 0.254829592
		a2 = -0.284496736
		a3 = 1.421413741
		a4 = -1.453152027
		a5 = 1.061405429
		p  = 0.3275911
	)

	sign := 1.0
	if x < 0 {
		sign = -1.0
		x = -x
	}

	// A&S formula 7.1.26
	t := 1.0 / (1.0 + p*x)
	y := 1.0 - (((((a5*t+a4)*t)+a3)*t+a2)*t+a1)*t*math.Exp(-x*x/2.0)

	return 0.5 * (1.0 + sign*y)
}

// CalculateAskPayout calculates the payout for an ask price
// Payout is calculated as: stake / probability, then commission is applied
func CalculateAskPayout(stake float64, probability float64, commission float64) float64 {
	if probability <= 0 || probability >= 1 {
		// Edge case: if probability is 0 or 1, return a reasonable payout
		if probability >= 1 {
			return stake // No risk, minimal payout
		}
		return stake * 100 // Very low probability, high payout
	}

	// Raw payout before commission
	rawPayout := stake / probability

	// Apply commission (hidden from user)
	finalPayout := rawPayout * (1.0 - commission)

	return finalPayout
}

// CalculateBidPrice calculates the bid price for an active contract
// Bid price represents the current market value based on remaining time
func CalculateBidPrice(
	payout float64,
	probability float64,
	isExpired bool,
	hasWon bool,
) float64 {
	if isExpired {
		if hasWon {
			return payout
		}
		return 0.0
	}

	// For active contracts, bid price is payout * probability
	return payout * probability
}

// TimeToExpiryInYears converts a duration in seconds to years (for Black-Scholes)
func TimeToExpiryInYears(seconds int64) float64 {
	const secondsPerYear = 365.25 * 24 * 60 * 60
	return float64(seconds) / secondsPerYear
}
