package fee_bot

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math"
	"sync/atomic"
	"time"
)

func TruncateFloat(val float64, precision int64) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Trunc(val*pow) / pow
}

func GetNonce() uint64 {
	var nonceCounter = time.Now().UnixMilli()
	now := time.Now().UnixMilli()
	for {
		// Load the current nonce value atomically.
		current := atomic.LoadInt64(&nonceCounter)

		// If the current time is greater than the stored nonce,
		// attempt to update the nonce to the current time.
		if current < now {
			if atomic.CompareAndSwapInt64(&nonceCounter, current, now) {
				return uint64(now)
			}
			// If the swap fails, retry.
			continue
		}

		// Otherwise, increment the nonce by one.
		newNonce := current + 1
		if atomic.CompareAndSwapInt64(&nonceCounter, current, newNonce) {
			return uint64(newNonce)
		}
	}
}

type RsvSignature struct {
	R string `json:"r"`
	S string `json:"s"`
	V byte   `json:"v"`
}

func ToTypedSig(r [32]byte, s [32]byte, v byte) RsvSignature {
	return RsvSignature{
		R: hexutil.Encode(r[:]),
		S: hexutil.Encode(s[:]),
		V: v,
	}
}

type FeeData struct {
	DailyUserVlm []struct {
		Date      string `json:"date"`
		UserCross string `json:"userCross"`
		UserAdd   string `json:"userAdd"`
		Exchange  string `json:"exchange"`
	} `json:"dailyUserVlm"`
	FeeSchedule struct {
		Cross     string `json:"cross"`
		Add       string `json:"add"`
		SpotCross string `json:"spotCross"`
		SpotAdd   string `json:"spotAdd"`
		Tiers     struct {
			Vip []struct {
				NtlCutoff string `json:"ntlCutoff"`
				Cross     string `json:"cross"`
				Add       string `json:"add"`
				SpotCross string `json:"spotCross"`
				SpotAdd   string `json:"spotAdd"`
			} `json:"vip"`
			Mm []struct {
				MakerFractionCutoff string `json:"makerFractionCutoff"`
				Add                 string `json:"add"`
			} `json:"mm"`
		} `json:"tiers"`
		ReferralDiscount     string `json:"referralDiscount"`
		StakingDiscountTiers []struct {
			BpsOfMaxSupply string `json:"bpsOfMaxSupply"`
			Discount       string `json:"discount"`
		} `json:"stakingDiscountTiers"`
	} `json:"feeSchedule"`
	UserCrossRate               string      `json:"userCrossRate"`
	UserAddRate                 string      `json:"userAddRate"`
	UserSpotCrossRate           string      `json:"userSpotCrossRate"`
	UserSpotAddRate             string      `json:"userSpotAddRate"`
	ActiveReferralDiscount      string      `json:"activeReferralDiscount"`
	Trial                       interface{} `json:"trial"`
	FeeTrialReward              string      `json:"feeTrialReward"`
	NextTrialAvailableTimestamp interface{} `json:"nextTrialAvailableTimestamp"`
	StakingLink                 interface{} `json:"stakingLink"`
	ActiveStakingDiscount       struct {
		BpsOfMaxSupply string `json:"bpsOfMaxSupply"`
		Discount       string `json:"discount"`
	} `json:"activeStakingDiscount"`
}
