package btcapi

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

type BTCAPIClient interface {
	GetRawTransaction(txHash *chainhash.Hash) (*wire.MsgTx, error)
	BroadcastTx(tx *wire.MsgTx) (*chainhash.Hash, error)
	ListUnspent(address btcutil.Address) ([]UnspentOutput, error)
}

type blockchainClient struct {
	rpcClient    *rpcclient.Client
	btcApiClient BTCAPIClient
}

type MempoolClient struct {
	baseURL string
}

var _ BTCAPIClient = (*MempoolClient)(nil)
