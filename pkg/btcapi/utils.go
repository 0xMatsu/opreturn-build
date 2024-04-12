package btcapi

import (
	"fmt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
	"io"
	"net/http"
)

func Request(method, baseURL, subPath string, requestBody io.Reader) ([]byte, error) {
	url := fmt.Sprintf("%s%s", baseURL, subPath)
	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	return body, nil
}

func SelectUTxO(utxos []UnspentOutput, amount int64) (int64, []UnspentOutput,
	error) {
	var value int64 = 0
	result := make([]UnspentOutput, 0)

	for _, output := range utxos {
		value += output.Output.Value
		result = append(result, output)

		if value > amount {
			return value, result, nil
		}
	}

	return 0, nil, fmt.Errorf("insufficient wallet %d < %d", value, amount)
}

func FillTxOutByOutPoint(client *MempoolClient,
	fetcher *txscript.MultiPrevOutFetcher,
	outPoint *wire.OutPoint) {
	var txOut *wire.TxOut
	tx, err := client.GetRawTransaction(&outPoint.Hash)
	if err != nil {
		return
	}
	if int(outPoint.Index) >= len(tx.TxOut) {
		return
	}
	txOut = tx.TxOut[outPoint.Index]
	fetcher.AddPrevOut(*outPoint, txOut)
}
