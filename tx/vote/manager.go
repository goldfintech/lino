package vote

import (
	"github.com/lino-network/lino/param"
	"github.com/lino-network/lino/tx/vote/model"
	"github.com/lino-network/lino/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	DelegatorSubstore    = []byte{0x00}
	VoterSubstore        = []byte{0x01}
	ProposalSubstore     = []byte{0x02}
	VoteSubstore         = []byte{0x03}
	ProposalListSubStore = []byte("ProposalList/ProposalListKey")
)

const decideProposalEvent = 0x1
const returnCoinEvent = 0x2

type VoteManager struct {
	storage     model.VoteStorage `json:"vote_storage"`
	paramHolder param.ParamHolder `json:"param_holder"`
}

func NewVoteManager(key sdk.StoreKey, holder param.ParamHolder) VoteManager {
	return VoteManager{
		storage:     model.NewVoteStorage(key),
		paramHolder: holder,
	}
}

func (vm VoteManager) InitGenesis(ctx sdk.Context) error {
	if err := vm.storage.InitGenesis(ctx); err != nil {
		return err
	}
	return nil
}

func (vm VoteManager) IsVoterExist(ctx sdk.Context, accKey types.AccountKey) bool {
	voterByte, _ := vm.storage.GetVoter(ctx, accKey)
	return voterByte != nil
}

func (vm VoteManager) IsInValidatorList(ctx sdk.Context, username types.AccountKey) bool {
	lst, err := vm.storage.GetReferenceList(ctx)
	if err != nil {
		return false
	}
	for _, validator := range lst.AllValidators {
		if validator == username {
			return true
		}
	}
	return false
}

func (vm VoteManager) IsDelegationExist(ctx sdk.Context, voter types.AccountKey, delegator types.AccountKey) bool {
	delegationByte, _ := vm.storage.GetDelegation(ctx, voter, delegator)
	return delegationByte != nil
}

func (vm VoteManager) IsLegalVoterWithdraw(
	ctx sdk.Context, username types.AccountKey, coin types.Coin) bool {
	voter, err := vm.storage.GetVoter(ctx, username)
	if err != nil {
		return false
	}
	// reject if this is a validator
	if vm.IsInValidatorList(ctx, username) {
		return false
	}

	param, err := vm.paramHolder.GetVoteParam(ctx)
	if err != nil {
		return false
	}
	// reject if withdraw is less than minimum voter withdraw
	if !coin.IsGTE(param.VoterMinWithdraw) {
		return false
	}
	//reject if the remaining coins are less than voter minimum deposit
	remaining := voter.Deposit.Minus(coin)
	if !remaining.IsGTE(param.VoterMinDeposit) {
		return false
	}
	return true
}

func (vm VoteManager) IsLegalDelegatorWithdraw(
	ctx sdk.Context, voterName types.AccountKey, delegatorName types.AccountKey, coin types.Coin) bool {
	delegation, err := vm.storage.GetDelegation(ctx, voterName, delegatorName)
	if err != nil {
		return false
	}

	param, err := vm.paramHolder.GetVoteParam(ctx)
	if err != nil {
		return false
	}

	// reject if withdraw is less than minimum delegator withdraw
	if !coin.IsGTE(param.DelegatorMinWithdraw) {
		return false
	}
	//reject if the remaining delegation are less than zero
	res := delegation.Amount.Minus(coin)
	return res.IsNotNegative()
}

func (vm VoteManager) IsOngoingProposal(ctx sdk.Context, proposalID types.ProposalKey) bool {
	lst, err := vm.storage.GetReferenceList(ctx)
	if err != nil {
		return false
	}
	for _, id := range lst.OngoingProposal {
		if id == proposalID {
			return true
		}
	}
	return false
}

func (vm VoteManager) CanBecomeValidator(ctx sdk.Context, username types.AccountKey) bool {
	voter, err := vm.storage.GetVoter(ctx, username)
	if err != nil {
		return false
	}
	param, err := vm.paramHolder.GetValidatorParam(ctx)
	if err != nil {
		return false
	}
	// check minimum voting deposit for validator
	return voter.Deposit.IsGTE(param.ValidatorMinVotingDeposit)
}

// only support change parameter proposal now
func (vm VoteManager) AddVote(ctx sdk.Context, proposalID types.ProposalKey, voter types.AccountKey, res bool) sdk.Error {
	vote := model.Vote{
		Voter:  voter,
		Result: res,
	}
	// will overwrite the old vote
	if err := vm.storage.SetVote(ctx, proposalID, voter, &vote); err != nil {
		return err
	}
	return nil
}

func (vm VoteManager) AddDelegation(ctx sdk.Context, voterName types.AccountKey, delegatorName types.AccountKey, coin types.Coin) sdk.Error {
	var delegation *model.Delegation
	var err sdk.Error

	if !vm.IsDelegationExist(ctx, voterName, delegatorName) {
		delegation = &model.Delegation{
			Delegator: delegatorName,
		}
	} else {
		delegation, err = vm.storage.GetDelegation(ctx, voterName, delegatorName)
		if err != nil {
			return err
		}
	}

	voter, err := vm.storage.GetVoter(ctx, voterName)
	if err != nil {
		return err
	}

	voter.DelegatedPower = voter.DelegatedPower.Plus(coin)
	delegation.Amount = delegation.Amount.Plus(coin)

	if err := vm.storage.SetDelegation(ctx, voterName, delegatorName, delegation); err != nil {
		return err
	}
	if err := vm.storage.SetVoter(ctx, voterName, voter); err != nil {
		return err
	}
	return nil
}

func (vm VoteManager) AddVoter(ctx sdk.Context, username types.AccountKey, coin types.Coin) sdk.Error {
	voter := &model.Voter{
		Username: username,
		Deposit:  coin,
	}

	param, err := vm.paramHolder.GetVoteParam(ctx)
	if err != nil {
		return err
	}

	// check minimum requirements for registering as a voter
	if !coin.IsGTE(param.VoterMinDeposit) {
		return ErrRegisterFeeNotEnough()
	}

	if err := vm.storage.SetVoter(ctx, username, voter); err != nil {
		return err
	}
	return nil
}

func (vm VoteManager) Deposit(ctx sdk.Context, username types.AccountKey, coin types.Coin) sdk.Error {
	voter, err := vm.storage.GetVoter(ctx, username)
	if err != nil {
		return err
	}
	voter.Deposit = voter.Deposit.Plus(coin)
	if err := vm.storage.SetVoter(ctx, username, voter); err != nil {
		return err
	}
	return nil
}

// this method won't check if it is a legal withdraw, caller should check by itself
func (vm VoteManager) VoterWithdraw(ctx sdk.Context, username types.AccountKey, coin types.Coin) sdk.Error {
	if coin.IsZero() {
		return ErrNoCoinToWithdraw()
	}
	voter, err := vm.storage.GetVoter(ctx, username)
	if err != nil {
		return err
	}
	voter.Deposit = voter.Deposit.Minus(coin)

	if voter.Deposit.IsZero() {
		if err := vm.storage.DeleteVoter(ctx, username); err != nil {
			return err
		}
	} else {
		if err := vm.storage.SetVoter(ctx, username, voter); err != nil {
			return err
		}
	}

	return nil
}

func (vm VoteManager) VoterWithdrawAll(ctx sdk.Context, username types.AccountKey) (types.Coin, sdk.Error) {
	voter, err := vm.storage.GetVoter(ctx, username)
	if err != nil {
		return types.NewCoin(0), err
	}
	if err := vm.VoterWithdraw(ctx, username, voter.Deposit); err != nil {
		return types.NewCoin(0), err
	}
	return voter.Deposit, nil
}

func (vm VoteManager) DelegatorWithdraw(ctx sdk.Context, voterName types.AccountKey, delegatorName types.AccountKey, coin types.Coin) sdk.Error {
	if coin.IsZero() {
		return ErrNoCoinToWithdraw()
	}
	// change voter's delegated power
	voter, err := vm.storage.GetVoter(ctx, voterName)
	if err != nil {
		return err
	}
	voter.DelegatedPower = voter.DelegatedPower.Minus(coin)
	if err := vm.storage.SetVoter(ctx, voterName, voter); err != nil {
		return err
	}

	// change this delegation's amount
	delegation, err := vm.storage.GetDelegation(ctx, voterName, delegatorName)
	if err != nil {
		return err
	}
	delegation.Amount = delegation.Amount.Minus(coin)

	if delegation.Amount.IsZero() {
		if err := vm.storage.DeleteDelegation(ctx, voterName, delegatorName); err != nil {
			return err
		}
	} else {
		vm.storage.SetDelegation(ctx, voterName, delegatorName, delegation)
	}

	return nil
}

func (vm VoteManager) DelegatorWithdrawAll(ctx sdk.Context, voterName types.AccountKey, delegatorName types.AccountKey) (types.Coin, sdk.Error) {
	delegation, err := vm.storage.GetDelegation(ctx, voterName, delegatorName)
	if err != nil {
		return types.NewCoin(0), err
	}
	if err := vm.DelegatorWithdraw(ctx, voterName, delegatorName, delegation.Amount); err != nil {
		return types.NewCoin(0), err
	}
	return delegation.Amount, nil
}

func (vm VoteManager) GetAllDelegators(ctx sdk.Context, voterName types.AccountKey) ([]types.AccountKey, sdk.Error) {
	return vm.storage.GetAllDelegators(ctx, voterName)
}

func (vm VoteManager) GetValidatorReferenceList(ctx sdk.Context) (*model.ReferenceList, sdk.Error) {
	return vm.storage.GetReferenceList(ctx)
}

func (vm VoteManager) SetValidatorReferenceList(ctx sdk.Context, lst *model.ReferenceList) sdk.Error {
	return vm.storage.SetReferenceList(ctx, lst)
}

func (vm VoteManager) GetVotingPower(ctx sdk.Context, voterName types.AccountKey) (types.Coin, sdk.Error) {
	voter, err := vm.storage.GetVoter(ctx, voterName)
	if err != nil {
		return types.Coin{}, err
	}
	res := voter.Deposit.Plus(voter.DelegatedPower)
	return res, nil
}
