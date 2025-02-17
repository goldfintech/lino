package account

import (
	"reflect"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"

	"github.com/lino-network/lino/param"
	"github.com/lino-network/lino/types"
	"github.com/lino-network/lino/x/account/model"
)

// AccountManager - account manager
type AccountManager struct {
	storage     model.AccountStorage
	paramHolder param.ParamHolder
}

// NewLinoAccount - new account manager
func NewAccountManager(key sdk.StoreKey, holder param.ParamHolder) AccountManager {
	return AccountManager{
		storage:     model.NewAccountStorage(key),
		paramHolder: holder,
	}
}

// DoesAccountExist - check if account exists in KVStore or not
func (accManager AccountManager) DoesAccountExist(ctx sdk.Context, username types.AccountKey) bool {
	return accManager.storage.DoesAccountExist(ctx, username)
}

// CreateAccount - create account, caller should make sure the register fee is valid
func (accManager AccountManager) CreateAccount(
	ctx sdk.Context, referrer types.AccountKey, username types.AccountKey,
	resetKey, transactionKey, appKey crypto.PubKey, registerDeposit types.Coin) sdk.Error {
	if accManager.DoesAccountExist(ctx, username) {
		return ErrAccountAlreadyExists(username)
	}
	accParams, err := accManager.paramHolder.GetAccountParam(ctx)
	if err != nil {
		return err
	}

	depositWithFullCoinDay := registerDeposit
	if registerDeposit.IsGT(accParams.FirstDepositFullCoinDayLimit) {
		depositWithFullCoinDay = accParams.FirstDepositFullCoinDayLimit
	}
	if err := accManager.storage.SetPendingCoinDayQueue(
		ctx, username, &model.PendingCoinDayQueue{}); err != nil {
		return err
	}

	if err := accManager.storage.SetBankFromAccountKey(ctx, username, &model.AccountBank{}); err != nil {
		return err
	}

	accountInfo := &model.AccountInfo{
		Username:       username,
		CreatedAt:      ctx.BlockHeader().Time.Unix(),
		ResetKey:       resetKey,
		TransactionKey: transactionKey,
		AppKey:         appKey,
	}
	if err := accManager.storage.SetInfo(ctx, username, accountInfo); err != nil {
		return err
	}

	accountMeta := &model.AccountMeta{
		LastActivityAt:       ctx.BlockHeader().Time.Unix(),
		LastReportOrUpvoteAt: ctx.BlockHeader().Time.Unix(),
		TransactionCapacity:  depositWithFullCoinDay,
	}
	if err := accManager.storage.SetMeta(ctx, username, accountMeta); err != nil {
		return err
	}
	if err := accManager.storage.SetReward(ctx, username, &model.Reward{}); err != nil {
		return err
	}
	// when open account, blockchain will give a certain amount lino with full coin day.
	if err := accManager.AddSavingCoinWithFullCoinDay(
		ctx, username, depositWithFullCoinDay, referrer,
		types.InitAccountWithFullCoinDayMemo, types.TransferIn); err != nil {
		return ErrAddSavingCoinWithFullCoinDay()
	}
	if err := accManager.AddSavingCoin(
		ctx, username, registerDeposit.Minus(depositWithFullCoinDay), referrer,
		types.InitAccountRegisterDepositMemo, types.TransferIn); err != nil {
		return ErrAddSavingCoin()
	}
	return nil
}

// GetCoinDay - recalculate and get user current coin day
func (accManager AccountManager) GetCoinDay(
	ctx sdk.Context, username types.AccountKey) (types.Coin, sdk.Error) {
	bank, err := accManager.storage.GetBankFromAccountKey(ctx, username)
	if err != nil {
		return types.NewCoinFromInt64(0), err
	}
	pendingCoinDayQueue, err := accManager.storage.GetPendingCoinDayQueue(ctx, username)
	if err != nil {
		return types.NewCoinFromInt64(0), err
	}
	accManager.updateTXFromPendingCoinDayQueue(ctx, bank, pendingCoinDayQueue)

	if err := accManager.storage.SetPendingCoinDayQueue(
		ctx, username, pendingCoinDayQueue); err != nil {
		return types.NewCoinFromInt64(0), err
	}

	if err := accManager.storage.SetBankFromAccountKey(ctx, username, bank); err != nil {
		return types.NewCoinFromInt64(0), err
	}

	coinDay := bank.CoinDay
	coinDayInQueue := types.DecToCoin(pendingCoinDayQueue.TotalCoinDay)
	totalCoinDay := coinDay.Plus(coinDayInQueue)
	return totalCoinDay, nil
}

// AddSavingCoin - add coin to balance and pending coin day
func (accManager AccountManager) AddSavingCoin(
	ctx sdk.Context, username types.AccountKey, coin types.Coin, from types.AccountKey, memo string,
	detailType types.TransferDetailType) (err sdk.Error) {
	if !accManager.DoesAccountExist(ctx, username) {
		return ErrAccountNotFound(username)
	}
	if coin.IsZero() {
		return nil
	}
	bank, err := accManager.storage.GetBankFromAccountKey(ctx, username)
	if err != nil {
		return err
	}

	bank.Saving = bank.Saving.Plus(coin)

	coinDayParams, err := accManager.paramHolder.GetCoinDayParam(ctx)
	if err != nil {
		return err
	}
	startTime := ctx.BlockHeader().Time.Unix() / types.CoinDayRecordIntervalSec * types.CoinDayRecordIntervalSec
	pendingCoinDay := model.PendingCoinDay{
		StartTime: startTime,
		EndTime:   startTime + coinDayParams.SecondsToRecoverCoinDay,
		Coin:      coin,
	}
	if err := accManager.addPendingCoinDayToQueue(ctx, username, bank, pendingCoinDay); err != nil {
		return err
	}

	if err := accManager.storage.SetBankFromAccountKey(ctx, username, bank); err != nil {
		return err
	}
	return nil
}

// AddSavingCoinWithFullCoinDay - add coin to balance with full coin day
func (accManager AccountManager) AddSavingCoinWithFullCoinDay(
	ctx sdk.Context, username types.AccountKey, coin types.Coin, from types.AccountKey, memo string,
	detailType types.TransferDetailType) (err sdk.Error) {
	if !accManager.DoesAccountExist(ctx, username) {
		return ErrAccountNotFound(username)
	}
	if coin.IsZero() {
		return nil
	}
	bank, err := accManager.storage.GetBankFromAccountKey(ctx, username)
	if err != nil {
		return err
	}

	bank.Saving = bank.Saving.Plus(coin)
	bank.CoinDay = bank.CoinDay.Plus(coin)
	if err := accManager.storage.SetBankFromAccountKey(ctx, username, bank); err != nil {
		return err
	}
	return nil
}

// MinusSavingCoin - minus coin from balance, remove coin day in the tail
func (accManager AccountManager) MinusSavingCoin(
	ctx sdk.Context, username types.AccountKey, coin types.Coin, to types.AccountKey,
	memo string, detailType types.TransferDetailType) (err sdk.Error) {
	accountBank, err := accManager.storage.GetBankFromAccountKey(ctx, username)
	if err != nil {
		return err
	}

	accountParams, err := accManager.paramHolder.GetAccountParam(ctx)
	if err != nil {
		return err
	}
	remain := accountBank.Saving.Minus(coin)
	if !remain.IsGTE(accountParams.MinimumBalance) {
		return ErrAccountSavingCoinNotEnough()
	}

	if coin.IsZero() {
		return nil
	}
	accountBank.Saving = accountBank.Saving.Minus(coin)
	pendingCoinDayQueue, err :=
		accManager.storage.GetPendingCoinDayQueue(ctx, username)
	if err != nil {
		return err
	}
	// update pending coin day queue, remove expired transaction
	accManager.updateTXFromPendingCoinDayQueue(ctx, accountBank, pendingCoinDayQueue)

	coinDayParams, err := accManager.paramHolder.GetCoinDayParam(ctx)
	if err != nil {
		return err
	}

	for len(pendingCoinDayQueue.PendingCoinDays) > 0 {
		lengthOfQueue := len(pendingCoinDayQueue.PendingCoinDays)
		pendingCoinDay := pendingCoinDayQueue.PendingCoinDays[lengthOfQueue-1]
		recoverRatio := types.NewDecFromRat(
			pendingCoinDayQueue.LastUpdatedAt-pendingCoinDay.StartTime,
			coinDayParams.SecondsToRecoverCoinDay)
		if coin.IsGTE(pendingCoinDay.Coin) {
			// if withdraw money is much more than last pending transaction, remove last transaction
			coin = coin.Minus(pendingCoinDay.Coin)

			pendingCoinDayQueue.TotalCoinDay =
				pendingCoinDayQueue.TotalCoinDay.Sub((recoverRatio.Mul(pendingCoinDay.Coin.ToDec())))

			pendingCoinDayQueue.TotalCoin = pendingCoinDayQueue.TotalCoin.Minus(pendingCoinDay.Coin)
			pendingCoinDayQueue.PendingCoinDays = pendingCoinDayQueue.PendingCoinDays[:lengthOfQueue-1]
		} else {
			// otherwise try to cut last pending transaction
			pendingCoinDayQueue.TotalCoinDay =
				pendingCoinDayQueue.TotalCoinDay.Sub(recoverRatio.Mul(coin.ToDec()))

			pendingCoinDayQueue.TotalCoin = pendingCoinDayQueue.TotalCoin.Minus(coin)
			pendingCoinDayQueue.PendingCoinDays[lengthOfQueue-1].Coin =
				pendingCoinDayQueue.PendingCoinDays[lengthOfQueue-1].Coin.Minus(coin)
			coin = types.NewCoinFromInt64(0)
			break
		}
	}
	if coin.IsPositive() {
		accountBank.CoinDay = accountBank.Saving
	}
	if err := accManager.storage.SetPendingCoinDayQueue(
		ctx, username, pendingCoinDayQueue); err != nil {
		return err
	}

	if err := accManager.storage.SetBankFromAccountKey(
		ctx, username, accountBank); err != nil {
		return err
	}
	return nil
}

// MinusSavingCoin - minus coin from balance, remove most charged coin day coin
func (accManager AccountManager) MinusSavingCoinWithFullCoinDay(
	ctx sdk.Context, username types.AccountKey, coin types.Coin, to types.AccountKey,
	memo string, detailType types.TransferDetailType) (types.Coin, sdk.Error) {
	if coin.IsZero() {
		return types.NewCoinFromInt64(0), nil
	}
	coinDayLost := types.NewCoinFromInt64(0)
	accountBank, err := accManager.storage.GetBankFromAccountKey(ctx, username)
	if err != nil {
		return types.NewCoinFromInt64(0), err
	}

	accountParams, err := accManager.paramHolder.GetAccountParam(ctx)
	if err != nil {
		return types.NewCoinFromInt64(0), err
	}
	remain := accountBank.Saving.Minus(coin)
	if !remain.IsGTE(accountParams.MinimumBalance) {
		return types.NewCoinFromInt64(0), ErrAccountSavingCoinNotEnough()
	}
	accountBank.Saving = remain

	pendingCoinDayQueue, err :=
		accManager.storage.GetPendingCoinDayQueue(ctx, username)
	if err != nil {
		return types.NewCoinFromInt64(0), err
	}
	// update pending coin day queue, remove expired transaction
	accManager.updateTXFromPendingCoinDayQueue(ctx, accountBank, pendingCoinDayQueue)
	coinDayParams, err := accManager.paramHolder.GetCoinDayParam(ctx)
	if err != nil {
		return types.NewCoinFromInt64(0), err
	}
	if accountBank.CoinDay.IsGTE(coin) {
		accountBank.CoinDay = accountBank.CoinDay.Minus(coin)
		coinDayLost = coin
	} else {
		coin = coin.Minus(accountBank.CoinDay)
		coinDayLost = accountBank.CoinDay
		accountBank.CoinDay = types.NewCoinFromInt64(0)

		for len(pendingCoinDayQueue.PendingCoinDays) > 0 {
			pendingCoinDay := pendingCoinDayQueue.PendingCoinDays[0]
			recoverRatio := types.NewDecFromRat(
				pendingCoinDayQueue.LastUpdatedAt-pendingCoinDay.StartTime,
				coinDayParams.SecondsToRecoverCoinDay)
			if coin.IsGTE(pendingCoinDay.Coin) {
				// if withdraw money is much than first pending transaction, remove first transaction
				coin = coin.Minus(pendingCoinDay.Coin)

				pendingCoinDayQueue.TotalCoinDay =
					pendingCoinDayQueue.TotalCoinDay.Sub((recoverRatio.Mul(pendingCoinDay.Coin.ToDec())))

				pendingCoinDayQueue.TotalCoin = pendingCoinDayQueue.TotalCoin.Minus(pendingCoinDay.Coin)
				pendingCoinDayQueue.PendingCoinDays = pendingCoinDayQueue.PendingCoinDays[1:]
			} else {
				// otherwise try to cut first pending transaction
				pendingCoinDayQueue.TotalCoinDay =
					pendingCoinDayQueue.TotalCoinDay.Sub(recoverRatio.Mul(coin.ToDec()))

				pendingCoinDayQueue.TotalCoin = pendingCoinDayQueue.TotalCoin.Minus(coin)
				pendingCoinDayQueue.PendingCoinDays[0].Coin = pendingCoinDayQueue.PendingCoinDays[0].Coin.Minus(coin)
				coin = types.NewCoinFromInt64(0)
				break
			}
		}
	}

	if err := accManager.storage.SetPendingCoinDayQueue(
		ctx, username, pendingCoinDayQueue); err != nil {
		return types.NewCoinFromInt64(0), err
	}

	if err := accManager.storage.SetBankFromAccountKey(
		ctx, username, accountBank); err != nil {
		return types.NewCoinFromInt64(0), err
	}
	return coinDayLost, nil
}

// UpdateJSONMeta - update user JONS meta data
func (accManager AccountManager) UpdateJSONMeta(
	ctx sdk.Context, username types.AccountKey, JSONMeta string) sdk.Error {
	accountMeta, err := accManager.storage.GetMeta(ctx, username)
	if err != nil {
		return err
	}
	accountMeta.JSONMeta = JSONMeta

	return accManager.storage.SetMeta(ctx, username, accountMeta)
}

// GetResetKey - get reset public key
func (accManager AccountManager) GetResetKey(
	ctx sdk.Context, username types.AccountKey) (crypto.PubKey, sdk.Error) {
	accountInfo, err := accManager.storage.GetInfo(ctx, username)
	if err != nil {
		return nil, ErrGetResetKey(username)
	}
	return accountInfo.ResetKey, nil
}

// GetTransactionKey - get transaction public key
func (accManager AccountManager) GetTransactionKey(
	ctx sdk.Context, username types.AccountKey) (crypto.PubKey, sdk.Error) {
	accountInfo, err := accManager.storage.GetInfo(ctx, username)
	if err != nil {
		return nil, ErrGetTransactionKey(username)
	}
	return accountInfo.TransactionKey, nil
}

// GetAppKey - get app public key
func (accManager AccountManager) GetAppKey(
	ctx sdk.Context, username types.AccountKey) (crypto.PubKey, sdk.Error) {
	accountInfo, err := accManager.storage.GetInfo(ctx, username)
	if err != nil {
		return nil, ErrGetAppKey(username)
	}
	return accountInfo.AppKey, nil
}

// GetSavingFromBank - get user balance
func (accManager AccountManager) GetSavingFromBank(
	ctx sdk.Context, username types.AccountKey) (types.Coin, sdk.Error) {
	accountBank, err := accManager.storage.GetBankFromAccountKey(ctx, username)
	if err != nil {
		return types.Coin{}, ErrGetSavingFromBank(err)
	}
	return accountBank.Saving, nil
}

// GetSequence - get user sequence number
func (accManager AccountManager) GetSequence(
	ctx sdk.Context, username types.AccountKey) (uint64, sdk.Error) {
	accountMeta, err := accManager.storage.GetMeta(ctx, username)
	if err != nil {
		return 0, ErrGetSequence(err)
	}
	return accountMeta.Sequence, nil
}

// GetLastReportOrUpvoteAt - get user last report or upvote time
func (accManager AccountManager) GetLastReportOrUpvoteAt(
	ctx sdk.Context, username types.AccountKey) (int64, sdk.Error) {
	accountMeta, err := accManager.storage.GetMeta(ctx, username)
	if err != nil {
		return 0, ErrGetLastReportOrUpvoteAt(err)
	}
	return accountMeta.LastReportOrUpvoteAt, nil
}

// UpdateLastReportOrUpvoteAt - update user last report or upvote time to current block time
func (accManager AccountManager) UpdateLastReportOrUpvoteAt(
	ctx sdk.Context, username types.AccountKey) sdk.Error {
	accountMeta, err := accManager.storage.GetMeta(ctx, username)
	if err != nil {
		return ErrUpdateLastReportOrUpvoteAt(err)
	}
	accountMeta.LastReportOrUpvoteAt = ctx.BlockHeader().Time.Unix()
	return accManager.storage.SetMeta(ctx, username, accountMeta)
}

// GetLastPostAt - get user last post time
func (accManager AccountManager) GetLastPostAt(
	ctx sdk.Context, username types.AccountKey) (int64, sdk.Error) {
	accountMeta, err := accManager.storage.GetMeta(ctx, username)
	if err != nil {
		return 0, ErrGetLastPostAt(err)
	}
	return accountMeta.LastPostAt, nil
}

// UpdateLastPostAt - update user last post time to current block time
func (accManager AccountManager) UpdateLastPostAt(
	ctx sdk.Context, username types.AccountKey) sdk.Error {
	accountMeta, err := accManager.storage.GetMeta(ctx, username)
	if err != nil {
		return ErrUpdateLastPostAt(err)
	}
	accountMeta.LastPostAt = ctx.BlockHeader().Time.Unix()
	return accManager.storage.SetMeta(ctx, username, accountMeta)
}

// GetFrozenMoneyList - get user frozen money list
func (accManager AccountManager) GetFrozenMoneyList(
	ctx sdk.Context, username types.AccountKey) ([]model.FrozenMoney, sdk.Error) {
	accountBank, err := accManager.storage.GetBankFromAccountKey(ctx, username)
	if err != nil {
		return nil, ErrGetFrozenMoneyList(err)
	}
	return accountBank.FrozenMoneyList, nil
}

// IncreaseSequenceByOne - increase user sequence number by one
func (accManager AccountManager) IncreaseSequenceByOne(
	ctx sdk.Context, username types.AccountKey) sdk.Error {
	accountMeta, err := accManager.storage.GetMeta(ctx, username)
	if err != nil {
		return ErrIncreaseSequenceByOne(err)
	}
	accountMeta.Sequence++
	if err := accManager.storage.SetMeta(ctx, username, accountMeta); err != nil {
		return err
	}
	return nil
}

// AddDirectDeposit - when user received the donation, the donation except friction will be added to
// total income and original income
func (accManager AccountManager) AddDirectDeposit(
	ctx sdk.Context, username types.AccountKey, directDeposit types.Coin) sdk.Error {
	reward, err := accManager.storage.GetReward(ctx, username)
	if err != nil {
		return err
	}
	reward.TotalIncome = reward.TotalIncome.Plus(directDeposit)
	reward.OriginalIncome = reward.OriginalIncome.Plus(directDeposit)
	if err := accManager.storage.SetReward(ctx, username, reward); err != nil {
		return err
	}
	return nil
}

// CheckUserTPSCapacity - to prevent user spam the chain, every user has a TPS capacity
func (accManager AccountManager) CheckUserTPSCapacity(
	ctx sdk.Context, me types.AccountKey, tpsCapacityRatio sdk.Dec) sdk.Error {
	accountMeta, err := accManager.storage.GetMeta(ctx, me)
	if err != nil {
		return err
	}
	// get update to date user coin day
	coinDay, err := accManager.GetCoinDay(ctx, me)
	if err != nil {
		return err
	}

	// get bandwidth parameters
	bandwidthParams, err := accManager.paramHolder.GetBandwidthParam(ctx)
	if err != nil {
		return err
	}

	// increase upper limit for capacity
	coinDay = coinDay.Plus(bandwidthParams.VirtualCoin)

	// if coin day less than last update transaction capacity, set to coin day
	if accountMeta.TransactionCapacity.IsGTE(coinDay) {
		accountMeta.TransactionCapacity = coinDay
	} else {
		// otherwise try to increase user capacity
		incrementRatio := types.NewDecFromRat(
			ctx.BlockHeader().Time.Unix()-accountMeta.LastActivityAt,
			bandwidthParams.SecondsToRecoverBandwidth)
		if incrementRatio.GT(sdk.OneDec()) {
			incrementRatio = sdk.OneDec()
		}
		capacityTillCoinDay := coinDay.Minus(accountMeta.TransactionCapacity)
		increaseCapacity := types.DecToCoin(capacityTillCoinDay.ToDec().Mul(incrementRatio))
		accountMeta.TransactionCapacity =
			accountMeta.TransactionCapacity.Plus(increaseCapacity)
	}
	// based on current tps, calculate current transaction cost
	currentTxCost := types.DecToCoin(
		bandwidthParams.CapacityUsagePerTransaction.ToDec().Mul(tpsCapacityRatio))
	// check if user current capacity is enough or not
	if currentTxCost.IsGT(accountMeta.TransactionCapacity) {
		return ErrAccountTPSCapacityNotEnough(me)
	}
	accountMeta.TransactionCapacity = accountMeta.TransactionCapacity.Minus(currentTxCost)
	accountMeta.LastActivityAt = ctx.BlockHeader().Time.Unix()
	if err := accManager.storage.SetMeta(ctx, me, accountMeta); err != nil {
		return err
	}
	return nil
}

// AuthorizePermission - userA authorize permission to userB (currently only support auth to a developer)
func (accManager AccountManager) AuthorizePermission(
	ctx sdk.Context, me types.AccountKey, grantTo types.AccountKey,
	validityPeriod int64, grantLevel types.Permission, amount types.Coin) sdk.Error {
	if !accManager.DoesAccountExist(ctx, grantTo) {
		return ErrAccountNotFound(grantTo)
	}
	if grantLevel != types.PreAuthorizationPermission && grantLevel != types.AppPermission {
		return ErrUnsupportGrantLevel()
	}
	newGrantPubKey := model.GrantPermission{
		GrantTo:    grantTo,
		Permission: grantLevel,
		CreatedAt:  ctx.BlockHeader().Time.Unix(),
		ExpiresAt:  ctx.BlockHeader().Time.Add(time.Duration(validityPeriod) * time.Second).Unix(),
		Amount:     amount,
	}
	pubkeys, err := accManager.storage.GetGrantPermissions(ctx, me, grantTo)
	if err != nil {
		// if grant permission list is empty, create a new one
		if err.Code() == model.ErrGrantPubKeyNotFound().Code() {
			return accManager.storage.SetGrantPermissions(ctx, me, grantTo, []*model.GrantPermission{&newGrantPubKey})
		}
		return err
	}

	// iterate grant public key list
	for i, pubkey := range pubkeys {
		if pubkey.Permission == grantLevel {
			pubkeys[i] = &newGrantPubKey
			return accManager.storage.SetGrantPermissions(ctx, me, grantTo, pubkeys)
		}
	}
	// If grant permission doesn't have record in store, add to grant public key list
	pubkeys = append(pubkeys, &newGrantPubKey)
	return accManager.storage.SetGrantPermissions(ctx, me, grantTo, pubkeys)
}

// RevokePermission - revoke permission from a developer
func (accManager AccountManager) RevokePermission(
	ctx sdk.Context, me types.AccountKey, grantTo types.AccountKey, permission types.Permission) sdk.Error {
	pubkeys, err := accManager.storage.GetGrantPermissions(ctx, me, grantTo)
	if err != nil {
		return err
	}

	// iterate grant public key list
	for i, pubkey := range pubkeys {
		if pubkey.Permission == permission {
			if len(pubkeys) == 1 {
				accManager.storage.DeleteAllGrantPermissions(ctx, me, grantTo)
				return nil
			}
			return accManager.storage.SetGrantPermissions(ctx, me, grantTo, append(pubkeys[:i], pubkeys[i+1:]...))
		}
	}
	return model.ErrGrantPubKeyNotFound()
}

// CheckSigningPubKeyOwner - given a public key, check if it is valid for given permission
func (accManager AccountManager) CheckSigningPubKeyOwner(
	ctx sdk.Context, me types.AccountKey, signKey crypto.PubKey,
	permission types.Permission, amount types.Coin) (types.AccountKey, sdk.Error) {
	if !accManager.DoesAccountExist(ctx, me) {
		return "", ErrAccountNotFound(me)
	}
	// if permission is reset, only reset key can sign for the msg
	if permission == types.ResetPermission {
		pubKey, err := accManager.GetResetKey(ctx, me)
		if err != nil {
			return "", err
		}
		if reflect.DeepEqual(pubKey, signKey) {
			return me, nil
		}
		return "", ErrCheckResetKey()
	}

	// otherwise transaction key has the highest permission
	pubKey, err := accManager.GetTransactionKey(ctx, me)
	if err != nil {
		return "", err
	}
	if reflect.DeepEqual(pubKey, signKey) {
		return me, nil
	}
	if permission == types.TransactionPermission {
		return "", ErrCheckTransactionKey()
	}

	// if all above keys not matched, check last one, app key
	if permission == types.AppPermission || permission == types.GrantAppPermission {
		pubKey, err = accManager.GetAppKey(ctx, me)
		if err != nil {
			return "", err
		}
		if reflect.DeepEqual(pubKey, signKey) {
			return me, nil
		}
	}

	if permission == types.GrantAppPermission {
		return "", ErrCheckGrantAppKey()
	}

	// if user doesn't use his own key, check his grant user pubkey
	grantPubKeys, err := accManager.storage.GetAllGrantPermissions(ctx, me)
	if err != nil {
		return "", err
	}

	for _, pubKey := range grantPubKeys {
		if pubKey.ExpiresAt < ctx.BlockHeader().Time.Unix() {
			continue
		}
		if permission != pubKey.Permission {
			continue
		}
		if permission == types.PreAuthorizationPermission {
			txKey, err := accManager.GetTransactionKey(ctx, pubKey.GrantTo)
			if err != nil {
				return "", err
			}
			if !reflect.DeepEqual(signKey, txKey) {
				continue
			}
			if amount.IsGT(pubKey.Amount) {
				return "", ErrPreAuthAmountInsufficient(pubKey.GrantTo, pubKey.Amount, amount)
			}
			// override previous grant public key
			if err := accManager.AuthorizePermission(ctx, me, pubKey.GrantTo, pubKey.ExpiresAt-ctx.BlockHeader().Time.Unix(), pubKey.Permission, pubKey.Amount.Minus(amount)); err != nil {
				return "", nil
			}
			return pubKey.GrantTo, nil
		}

		if permission == types.AppPermission {
			appKey, err := accManager.GetAppKey(ctx, pubKey.GrantTo)
			if err != nil {
				return "", err
			}
			if !reflect.DeepEqual(signKey, appKey) {
				continue
			}
			return pubKey.GrantTo, nil
		}
	}
	return "", ErrCheckAuthenticatePubKeyOwner(me)
}

func (accManager AccountManager) addPendingCoinDayToQueue(
	ctx sdk.Context, username types.AccountKey, bank *model.AccountBank,
	pendingCoinDay model.PendingCoinDay) sdk.Error {
	pendingCoinDayQueue, err := accManager.storage.GetPendingCoinDayQueue(ctx, username)
	if err != nil {
		return err
	}
	accManager.updateTXFromPendingCoinDayQueue(ctx, bank, pendingCoinDayQueue)
	idx := len(pendingCoinDayQueue.PendingCoinDays) - 1
	if len(pendingCoinDayQueue.PendingCoinDays) > 0 && pendingCoinDayQueue.PendingCoinDays[idx].StartTime == pendingCoinDay.StartTime {
		pendingCoinDayQueue.PendingCoinDays[idx].Coin = pendingCoinDayQueue.PendingCoinDays[idx].Coin.Plus(pendingCoinDay.Coin)
		pendingCoinDayQueue.TotalCoin = pendingCoinDayQueue.TotalCoin.Plus(pendingCoinDay.Coin)
	} else {
		pendingCoinDayQueue.PendingCoinDays = append(pendingCoinDayQueue.PendingCoinDays, pendingCoinDay)
		pendingCoinDayQueue.TotalCoin = pendingCoinDayQueue.TotalCoin.Plus(pendingCoinDay.Coin)
	}
	return accManager.storage.SetPendingCoinDayQueue(ctx, username, pendingCoinDayQueue)
}

// RecoverAccount - reset three public key pairs
func (accManager AccountManager) RecoverAccount(
	ctx sdk.Context, username types.AccountKey,
	newResetPubKey, newTransactionPubKey, newAppPubKey crypto.PubKey) sdk.Error {
	accInfo, err := accManager.storage.GetInfo(ctx, username)
	if err != nil {
		return err
	}

	accInfo.ResetKey = newResetPubKey
	accInfo.TransactionKey = newTransactionPubKey
	accInfo.AppKey = newAppPubKey
	if err := accManager.storage.SetInfo(ctx, username, accInfo); err != nil {
		return err
	}
	return nil
}

func (accManager AccountManager) updateTXFromPendingCoinDayQueue(
	ctx sdk.Context, bank *model.AccountBank, pendingCoinDayQueue *model.PendingCoinDayQueue) sdk.Error {
	// remove expired transaction
	coinDayParams, err := accManager.paramHolder.GetCoinDayParam(ctx)
	if err != nil {
		return err
	}

	currentTimeSlot := ctx.BlockHeader().Time.Unix() / types.CoinDayRecordIntervalSec * types.CoinDayRecordIntervalSec
	for len(pendingCoinDayQueue.PendingCoinDays) > 0 {
		pendingCoinDay := pendingCoinDayQueue.PendingCoinDays[0]
		if pendingCoinDay.EndTime <= currentTimeSlot {
			// remove the transaction from queue, clean coin day coin in queue and minus total coin
			// coinDayRatioOfThisTransaction means the ratio of coin day of this transaction was added last time
			coinDayRatioOfThisTransaction := types.NewDecFromRat(
				pendingCoinDayQueue.LastUpdatedAt-pendingCoinDay.StartTime,
				coinDayParams.SecondsToRecoverCoinDay)
			// remove the coin day in the queue of this transaction
			pendingCoinDayQueue.TotalCoinDay =
				pendingCoinDayQueue.TotalCoinDay.Sub(
					coinDayRatioOfThisTransaction.Mul(pendingCoinDay.Coin.ToDec()))
			// update bank coin day
			bank.CoinDay = bank.CoinDay.Plus(pendingCoinDay.Coin)

			pendingCoinDayQueue.TotalCoin = pendingCoinDayQueue.TotalCoin.Minus(pendingCoinDay.Coin)

			pendingCoinDayQueue.PendingCoinDays = pendingCoinDayQueue.PendingCoinDays[1:]
		} else {
			break
		}
	}
	if len(pendingCoinDayQueue.PendingCoinDays) == 0 {
		pendingCoinDayQueue.TotalCoin = types.NewCoinFromInt64(0)
		pendingCoinDayQueue.TotalCoinDay = sdk.ZeroDec()
	} else {
		// update all pending coin day at the same time
		// recoverRatio = (currentTime - lastUpdateTime)/totalRecoverSeconds
		// totalCoinDay += recoverRatio * totalCoin

		// XXX(yumin): @mul-first-form transform to
		// totalcoin * (currentTime - lastUpdateTime)/totalRecoverSeconds
		pendingCoinDayQueue.TotalCoinDay =
			pendingCoinDayQueue.TotalCoinDay.Add(
				pendingCoinDayQueue.TotalCoin.ToDec().Mul(
					sdk.NewDec(currentTimeSlot - pendingCoinDayQueue.LastUpdatedAt)).Quo(
					sdk.NewDec(coinDayParams.SecondsToRecoverCoinDay)))
	}

	pendingCoinDayQueue.LastUpdatedAt = currentTimeSlot
	return nil
}

// AddFrozenMoney - add frozen money to user's frozen money list
func (accManager AccountManager) AddFrozenMoney(
	ctx sdk.Context, username types.AccountKey,
	amount types.Coin, start, interval, times int64) sdk.Error {
	accountBank, err := accManager.storage.GetBankFromAccountKey(ctx, username)
	if err != nil {
		return err
	}
	accManager.cleanExpiredFrozenMoney(ctx, accountBank)
	frozenMoney := model.FrozenMoney{
		Amount:   amount,
		StartAt:  start,
		Interval: interval,
		Times:    times,
	}

	accParams, err := accManager.paramHolder.GetAccountParam(ctx)
	if err != nil {
		return err
	}

	if int64(len(accountBank.FrozenMoneyList)) >= accParams.MaxNumFrozenMoney {
		return ErrFrozenMoneyListTooLong()
	}

	accountBank.FrozenMoneyList = append(accountBank.FrozenMoneyList, frozenMoney)
	if err := accManager.storage.SetBankFromAccountKey(ctx, username, accountBank); err != nil {
		return err
	}
	return nil
}

func (accManager AccountManager) cleanExpiredFrozenMoney(ctx sdk.Context, bank *model.AccountBank) {
	idx := 0
	for idx < len(bank.FrozenMoneyList) {
		frozenMoney := bank.FrozenMoneyList[idx]
		if ctx.BlockHeader().Time.Unix() > frozenMoney.StartAt+3600*frozenMoney.Interval*frozenMoney.Times {
			bank.FrozenMoneyList = append(bank.FrozenMoneyList[:idx], bank.FrozenMoneyList[idx+1:]...)
			continue
		}

		idx++
	}
}

// Export -
func (accManager AccountManager) Export(ctx sdk.Context) *model.AccountTables {
	return accManager.storage.Export(ctx)
}

// Import -
func (accManager AccountManager) Import(ctx sdk.Context, dt *model.AccountTablesIR) {
	accManager.storage.Import(ctx, dt)
	// XXX(yumin): during upgrade-1, we changed the kv of grantPubKey, so we import them here
	// by calling AuthorizePermission.
	for _, v := range dt.AccountGrantPubKeys {
		grant := v.GrantPubKey
		remainingTime := grant.ExpiresAt - ctx.BlockHeader().Time.Unix()
		if remainingTime > 0 {
			// fmt.Printf("%s %s %d %d %d", v.Username, grant.Username,
			// 	remainingTime, grant.Permission, grant.Amount)
			accManager.AuthorizePermission(ctx, v.Username, grant.Username,
				remainingTime, grant.Permission, grant.Amount)
		}
	}
}

// IterateAccounts - iterate accounts in KVStore
func (accManager AccountManager) IterateAccounts(ctx sdk.Context, process func(model.AccountInfo, model.AccountBank) (stop bool)) {
	accManager.storage.IterateAccounts(ctx, process)
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a < b {
		return b
	}
	return a
}
