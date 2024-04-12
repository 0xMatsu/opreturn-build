package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/0xMatsu/opreturn-build/pkg/btcapi"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"log"
)

func main() {
	// 钱包私钥
	wifKey := "L4s1sw38KpFN73hBG3a4o3oU6VDhG5VEGjbjiWpps8CST6zKaUs7"
	// Atomicals 接收捐款地址
	destAddress := "bc1p3f2t2dal9wpvw4u7wtjxsyf36hfsxf4kyfn55d6jqfyc4yd56ppqgsze2e"
	// OP_RETURN 后需要跟的消息
	message := ""
	// gas 费用
	var gasRate int64 = 9
	// 捐款的 sat 数量，1 BTC = 100000000 sat
	var amount int64 = 0
	// 测试网 or 主网，建议先在测试网正常运行再从主网运行
	cfg := &chaincfg.TestNet3Params

	rawTx, err := createTxWithNote(wifKey, message, destAddress, amount, gasRate, cfg)
	if err != nil {
		log.Fatalln(err)
	}
	println(rawTx)
}

func createTxWithNote(prvKey string, note string, destAddr string,
	amount int64, gasRate int64, cfg *chaincfg.Params) (string, error) {
	wif, err := btcutil.DecodeWIF(prvKey)
	if err != nil {
		log.Fatalln("Decode wif to key failed.")
		return "", err
	}

	selfAddr, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(txscript.
		ComputeTaprootKeyNoScript(wif.PrivKey.PubKey())), cfg)
	if err != nil {
		log.Fatalln("Get self taproot address failed.")
		return "", err
	}

	selfAddrByte, err := txscript.PayToAddrScript(selfAddr)
	if err != nil {
		log.Fatalln("Create pay to self address script failed.")
		return "", err
	}

	destAddress, err := btcutil.DecodeAddress(destAddr, cfg)
	if err != nil {
		log.Fatalln("Decode destination address failed.")
		return "", err
	}

	destAddrByte, err := txscript.PayToAddrScript(destAddress)
	if err != nil {
		log.Fatalln("Create pay to address script failed.")
		return "", err
	}

	client := btcapi.NewClient(cfg)
	utxos, err := client.ListUnspent(selfAddr)

	balance, spendable, err := btcapi.SelectUTxO(utxos, amount)
	fmt.Printf("Balance: %d\n", balance)
	if err != nil {
		log.Fatalln("Get UTxO failed.")
		return "", err
	}

	redeemTx := wire.NewMsgTx(wire.TxVersion)
	fetcher := txscript.NewMultiPrevOutFetcher(nil)
	for _, in := range spendable {
		txIn := wire.NewTxIn(in.Outpoint, nil, nil)
		redeemTx.AddTxIn(txIn)
		btcapi.FillTxOutByOutPoint(client, fetcher, in.Outpoint)
	}

	noteScript, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddData([]byte(note)).Script()
	if err != nil {
		log.Fatalln("Build note script failed.")
		return "", err
	}
	noteTxOut := wire.NewTxOut(0, noteScript)
	redeemTx.AddTxOut(noteTxOut)

	transferScript := wire.NewTxOut(amount, destAddrByte)
	redeemTx.AddTxOut(transferScript)
	fmt.Printf("Remain bitcoin: %d\n", balance-amount)

	redeemTx.AddTxOut(wire.NewTxOut(balance-amount, selfAddrByte))
	gasFee := btcutil.Amount(mempool.GetTxVirtualSize(btcutil.NewTx(
		redeemTx))) * btcutil.Amount(gasRate)
	fmt.Printf("Gas fee: %d\n", gasFee)

	redeemTx.TxOut[2].Value -= int64(gasFee)

	for idx := range redeemTx.TxIn {
		txOut := fetcher.FetchPrevOutput(redeemTx.TxIn[idx].PreviousOutPoint)
		witness, err := txscript.TaprootWitnessSignature(redeemTx,
			txscript.NewTxSigHashes(redeemTx, fetcher), idx, txOut.Value,
			txOut.PkScript, txscript.SigHashDefault, wif.PrivKey)
		if err != nil {
			return "", err
		}

		redeemTx.TxIn[idx].Witness = witness
	}

	var signedTx bytes.Buffer
	redeemTx.Serialize(&signedTx)
	finalRawTx := hex.EncodeToString(signedTx.Bytes())

	return finalRawTx, nil
}
