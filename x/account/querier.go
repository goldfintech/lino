package account

import (
	"encoding/hex"

	wire "github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	linotypes "github.com/lino-network/lino/types"
	"github.com/lino-network/lino/x/account/model"
	"github.com/lino-network/lino/x/account/types"
	abci "github.com/tendermint/tendermint/abci/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

const (
	QueryAccountInfo            = "info"
	QueryAccountBank            = "bank"
	QueryAccountMeta            = "meta"
	QueryAccountReward          = "reward"
	QueryAccountPendingCoinDay  = "pendingCoinDay"
	QueryAccountGrantPubKeys    = "grantPubKey"
	QueryAccountAllGrantPubKeys = "allGrantPubKey"
	QueryTxAndAccountSequence   = "txAndSeq"
)

// creates a querier for account REST endpoints
func NewQuerier(am AccountKeeper) sdk.Querier {
	cdc := wire.New()
	wire.RegisterCrypto(cdc)
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QueryAccountInfo:
			return queryAccountInfo(ctx, cdc, path[1:], req, am)
		case QueryAccountBank:
			return queryAccountBank(ctx, cdc, path[1:], req, am)
		case QueryAccountMeta:
			return queryAccountMeta(ctx, cdc, path[1:], req, am)
		case QueryAccountReward:
			return queryAccountReward(ctx, cdc, path[1:], req, am)
		// case QueryAccountPendingCoinDay:
		// 	return queryAccountPendingCoinDay(ctx, cdc, path[1:], req, am)
		case QueryAccountGrantPubKeys:
			return queryAccountGrantPubKeys(ctx, cdc, path[1:], req, am)
		case QueryAccountAllGrantPubKeys:
			return queryAccountAllGrantPubKeys(ctx, cdc, path[1:], req, am)
		case QueryTxAndAccountSequence:
			return queryTxAndSequenceNumber(ctx, cdc, path[1:], req, am)
		default:
			return nil, sdk.ErrUnknownRequest("unknown account query endpoint")
		}
	}
}

func queryAccountInfo(ctx sdk.Context, cdc *wire.Codec, path []string, req abci.RequestQuery, am AccountKeeper) ([]byte, sdk.Error) {
	if err := linotypes.CheckPathContentAndMinLength(path, 1); err != nil {
		return nil, err
	}
	accountInfo, err := am.GetInfo(ctx, linotypes.AccountKey(path[0]))
	if err != nil {
		return nil, err
	}
	res, marshalErr := cdc.MarshalJSON(accountInfo)
	if marshalErr != nil {
		return nil, types.ErrQueryFailed()
	}
	return res, nil
}

func queryAccountBank(ctx sdk.Context, cdc *wire.Codec, path []string, req abci.RequestQuery, am AccountKeeper) ([]byte, sdk.Error) {
	if err := linotypes.CheckPathContentAndMinLength(path, 1); err != nil {
		return nil, err
	}
	bank, err := am.GetBank(ctx, linotypes.AccountKey(path[0]))
	if err != nil {
		return nil, err
	}
	res, marshalErr := cdc.MarshalJSON(bank)
	if marshalErr != nil {
		return nil, types.ErrQueryFailed()
	}
	return res, nil
}

func queryAccountMeta(ctx sdk.Context, cdc *wire.Codec, path []string, req abci.RequestQuery, am AccountKeeper) ([]byte, sdk.Error) {
	if err := linotypes.CheckPathContentAndMinLength(path, 1); err != nil {
		return nil, err
	}
	accountMeta, err := am.GetMeta(ctx, linotypes.AccountKey(path[0]))
	if err != nil {
		return nil, err
	}
	res, marshalErr := cdc.MarshalJSON(accountMeta)
	if marshalErr != nil {
		return nil, types.ErrQueryFailed()
	}
	return res, nil
}

func queryTxAndSequenceNumber(ctx sdk.Context, cdc *wire.Codec, path []string, req abci.RequestQuery, am AccountKeeper) ([]byte, sdk.Error) {
	if err := linotypes.CheckPathContentAndMinLength(path, 2); err != nil {
		return nil, err
	}
	bank, err := am.GetBank(ctx, linotypes.AccountKey(path[0]))
	if err != nil {
		return nil, err
	}

	txAndSeq := model.TxAndSequenceNumber{
		Username: path[0],
		Sequence: bank.Sequence,
	}

	txHash, decodeFail := hex.DecodeString(path[1])
	if decodeFail != nil {
		return nil, types.ErrQueryFailed()
	}

	rpc := rpcclient.NewHTTP("http://localhost:26657", "/websocket")
	tx, _ := rpc.Tx(txHash, false)
	if tx != nil {
		txAndSeq.Tx = &model.Transaction{
			Hash:   hex.EncodeToString(tx.Hash),
			Height: tx.Height,
			Tx:     tx.Tx,
			Code:   tx.TxResult.Code,
			Log:    tx.TxResult.Log,
		}
	}
	res, marshalErr := cdc.MarshalJSON(txAndSeq)
	if marshalErr != nil {
		return nil, types.ErrQueryFailed()
	}
	return res, nil
}

func queryAccountReward(ctx sdk.Context, cdc *wire.Codec, path []string, req abci.RequestQuery, am AccountKeeper) ([]byte, sdk.Error) {
	if err := linotypes.CheckPathContentAndMinLength(path, 1); err != nil {
		return nil, err
	}
	reward, err := am.GetReward(ctx, linotypes.AccountKey(path[0]))
	if err != nil {
		return nil, err
	}
	res, marshalErr := cdc.MarshalJSON(reward)
	if marshalErr != nil {
		return nil, types.ErrQueryFailed()
	}
	return res, nil
}

// func queryAccountPendingCoinDay(ctx sdk.Context, cdc *wire.Codec, path []string, req abci.RequestQuery, am AccountKeeper) ([]byte, sdk.Error) {
// 	if err := linotypes.CheckPathContentAndMinLength(path, 1); err != nil {
// 		return nil, err
// 	}
// 	pendingCoinDay, err := am.storage.GetPendingCoinDayQueue(ctx, linotypes.AccountKey(path[0]))
// 	if err != nil {
// 		return nil, err
// 	}
// 	res, marshalErr := cdc.MarshalJSON(pendingCoinDay)
// 	if marshalErr != nil {
// 		return nil, ErrQueryFailed()
// 	}
// 	return res, nil
// }

func queryAccountGrantPubKeys(ctx sdk.Context, cdc *wire.Codec, path []string, req abci.RequestQuery, am AccountKeeper) ([]byte, sdk.Error) {
	if err := linotypes.CheckPathContentAndMinLength(path, 2); err != nil {
		return nil, err
	}
	grantPubKeys, err := am.GetGrantPubKeys(ctx, linotypes.AccountKey(path[0]), linotypes.AccountKey(path[1]))
	if err != nil {
		return nil, types.ErrQueryFailed()
	}
	res, marshalErr := cdc.MarshalJSON(grantPubKeys)
	if marshalErr != nil {
		return nil, types.ErrQueryFailed()
	}
	return res, nil
}

func queryAccountAllGrantPubKeys(ctx sdk.Context, cdc *wire.Codec, path []string, req abci.RequestQuery, am AccountKeeper) ([]byte, sdk.Error) {
	if err := linotypes.CheckPathContentAndMinLength(path, 1); err != nil {
		return nil, err
	}
	pubKeys, err := am.GetAllGrantPubKeys(ctx, linotypes.AccountKey(path[0]))
	if err != nil {
		return nil, types.ErrQueryFailed()
	}
	res, marshalErr := cdc.MarshalJSON(pubKeys)
	if marshalErr != nil {
		return nil, types.ErrQueryFailed()
	}
	return res, nil
}
