package main

import (
	"fee-bot/pkg/base"
	"fee-bot/pkg/fee-bot"
	"fmt"
	"github.com/Logarithm-Labs/go-hyperliquid/hyperliquid"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/imroc/req/v3"
	"log"
	"strings"
)

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

func main() {
	config := base.GetBotConfig()
	accountAddress := base.PkToAddress(config.Hyper.AccountPk)
	agentAddress := base.PkToAddress(config.Hyper.AgentPk)

	privateKeyHex := config.Hyper.AccountPk
	agentAddress = strings.ToLower(agentAddress)
	agentName := "test1" // 可选代理名称

	hyperliquidClient := hyperliquid.NewHyperliquid(&hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: accountAddress,
		PrivateKey:     privateKeyHex,
	})
	nonce := fee_bot.GetNonce()

	action := map[string]interface{}{
		"type":             "approveAgent",
		"hyperliquidChain": "Mainnet",
		"signatureChainId": "0xa4b1",
		"agentAddress":     agentAddress,
		"agentName":        agentName,
		"nonce":            nonce, // 使用uint64类型
	}

	types := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "agentAddress", Type: "address"},
		{Name: "agentName", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	v, r, s, err := hyperliquidClient.SignUserSignableAction(action, types, "HyperliquidTransaction:ApproveAgent")
	if err != nil {
		log.Fatal(err)
	}
	payload := map[string]interface{}{
		"action":    action,
		"signature": ToTypedSig(r, s, v),
		"nonce":     nonce,
	}

	resp, err := req.DevMode().R().SetBodyJsonMarshal(payload).Post("https://api.hyperliquid.xyz/exchange")
	if err != nil {
		log.Fatal("HTTP error:", err)
	}

	fmt.Println("Response Status:", resp.StatusCode)
	fmt.Println("Response Body:", resp.String())
}
