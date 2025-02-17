package auth

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	abci "github.com/tendermint/tendermint/abci/types"
	crypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/lino-network/lino/param"
	"github.com/lino-network/lino/types"
	acc "github.com/lino-network/lino/x/account"
	"github.com/lino-network/lino/x/global"
	post "github.com/lino-network/lino/x/post"
	postmn "github.com/lino-network/lino/x/post/manager"
	posttypes "github.com/lino-network/lino/x/post/types"
)

type TestMsg struct {
	Signers    []types.AccountKey
	Permission types.Permission
	Amount     types.Coin
}

var _ types.Msg = TestMsg{}

func (msg TestMsg) Route() string                   { return "normal msg" }
func (msg TestMsg) Type() string                    { return "normal msg" }
func (msg TestMsg) GetPermission() types.Permission { return msg.Permission }
func (msg TestMsg) GetSignBytes() []byte {
	bz, err := json.Marshal(msg.Signers)
	if err != nil {
		panic(err)
	}
	return bz
}
func (msg TestMsg) ValidateBasic() sdk.Error { return nil }
func (msg TestMsg) GetSigners() []sdk.AccAddress {
	addrs := make([]sdk.AccAddress, len(msg.Signers))
	for i, signer := range msg.Signers {
		addrs[i] = sdk.AccAddress(signer)
	}
	return addrs
}
func (msg TestMsg) GetConsumeAmount() types.Coin {
	return msg.Amount
}

func newTestMsg(accKeys ...types.AccountKey) TestMsg {
	return TestMsg{
		Signers:    accKeys,
		Permission: types.AppPermission,
		Amount:     types.NewCoinFromInt64(10),
	}
}

func newTestTx(
	ctx sdk.Context, msgs []sdk.Msg, privs []crypto.PrivKey, seqs []uint64) sdk.Tx {
	sigs := make([]auth.StdSignature, len(privs))

	for i, priv := range privs {
		signBytes := auth.StdSignBytes(ctx.ChainID(), 0, seqs[i], auth.StdFee{}, msgs, "")
		bz, _ := priv.Sign(signBytes)
		sigs[i] = auth.StdSignature{
			PubKey: priv.PubKey(), Signature: bz}
	}
	tx := auth.NewStdTx(msgs, auth.StdFee{}, sigs, "")
	return tx
}

func initGlobalManager(ctx sdk.Context, gm global.GlobalManager) error {
	return gm.InitGlobalManager(ctx, types.NewCoinFromInt64(10000*types.Decimals))
}

type AnteTestSuite struct {
	suite.Suite
	am   acc.AccountManager
	pm   post.PostKeeper
	gm   global.GlobalManager
	ph   param.ParamHolder
	ctx  sdk.Context
	ante sdk.AnteHandler
}

func (suite *AnteTestSuite) SetupTest() {
	TestAccountKVStoreKey := sdk.NewKVStoreKey("account")
	TestPostKVStoreKey := sdk.NewKVStoreKey("post")
	TestGlobalKVStoreKey := sdk.NewKVStoreKey("global")
	TestParamKVStoreKey := sdk.NewKVStoreKey("param")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(TestAccountKVStoreKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(TestPostKVStoreKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(TestGlobalKVStoreKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(TestParamKVStoreKey, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	ctx := sdk.NewContext(
		ms, abci.Header{ChainID: "Lino", Height: 1, Time: time.Now()}, false, log.NewNopLogger())

	ph := param.NewParamHolder(TestParamKVStoreKey)
	ph.InitParam(ctx)
	am := acc.NewAccountManager(TestAccountKVStoreKey, ph)
	gm := global.NewGlobalManager(TestGlobalKVStoreKey, ph)
	// dev, rep, price = nil
	pm := postmn.NewPostManager(TestPostKVStoreKey, am, &gm, nil, nil, nil)
	initGlobalManager(ctx, gm)
	anteHandler := NewAnteHandler(am, gm, pm)

	suite.am = am
	suite.pm = pm
	suite.gm = gm
	suite.ph = ph
	suite.ctx = ctx
	suite.ante = anteHandler
}

func (suite *AnteTestSuite) createTestAccount(username string) (secp256k1.PrivKeySecp256k1,
	secp256k1.PrivKeySecp256k1, secp256k1.PrivKeySecp256k1, types.AccountKey) {
	resetKey := secp256k1.GenPrivKey()
	transactionKey := secp256k1.GenPrivKey()
	appKey := secp256k1.GenPrivKey()
	accParams, _ := suite.ph.GetAccountParam(suite.ctx)
	suite.am.CreateAccount(suite.ctx, "referrer", types.AccountKey(username),
		resetKey.PubKey(), transactionKey.PubKey(), appKey.PubKey(), accParams.RegisterFee)
	return resetKey, transactionKey, appKey, types.AccountKey(username)
}

func (suite *AnteTestSuite) createTestPost(postid string, author types.AccountKey) {
	msg := post.CreatePostMsg{
		PostID:    postid,
		Title:     "testTitle",
		Content:   "qqqqqqq",
		Author:    author,
		CreatedBy: author,
	}
	err := suite.pm.CreatePost(suite.ctx, msg.Author, msg.PostID, msg.CreatedBy, msg.Content, msg.Title)
	suite.Require().Nil(err)
}

// run the tx through the anteHandler and ensure its valid
func (suite *AnteTestSuite) checkValidTx(tx sdk.Tx) {
	_, result, abort := suite.ante(suite.ctx, tx, false)
	suite.Assert().False(abort)
	suite.Assert().True(result.Code.IsOK()) // redundent
	suite.Assert().True(result.IsOK())
}

// run the tx through the anteHandler and ensure it fails with the given code
func (suite *AnteTestSuite) checkInvalidTx(tx sdk.Tx, result sdk.Result) {
	_, r, abort := suite.ante(suite.ctx, tx, false)

	suite.Assert().True(abort)
	suite.Assert().Equal(result, r)
}

// Test various error cases in the AnteHandler control flow.
func (suite *AnteTestSuite) TestAnteHandlerSigErrors() {
	// get private key and username
	_, transaction1, _, user1 := suite.createTestAccount("user1")
	_, transaction2, _, user2 := suite.createTestAccount("user2")

	// msg and signatures
	var tx sdk.Tx
	msg := newTestMsg(user1, user2)

	// test no signatures
	privs, seqs := []crypto.PrivKey{}, []uint64{}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, ErrNoSignatures().Result())

	// test num sigs less than GetSigners
	privs, seqs = []crypto.PrivKey{transaction1}, []uint64{0}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, ErrWrongNumberOfSigners().Result())

	// test sig user mismatch
	privs, seqs = []crypto.PrivKey{transaction2, transaction1}, []uint64{0, 0}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, acc.ErrCheckAuthenticatePubKeyOwner(user1).Result())
}

// Test various error cases in the AnteHandler control flow.
func (suite *AnteTestSuite) TestAnteHandlerNormalTx() {
	// keys and username
	_, transaction1, _, user1 := suite.createTestAccount("user1")
	_, transaction2, _, _ := suite.createTestAccount("user2")

	// msg and signatures
	var tx sdk.Tx
	msg := newTestMsg(user1)

	// test valid transaction
	privs, seqs := []crypto.PrivKey{transaction1}, []uint64{0}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkValidTx(tx)
	seq, err := suite.am.GetSequence(suite.ctx, user1)
	suite.Nil(err)
	suite.Equal(seq, uint64(1))

	// test no signatures
	privs, seqs = []crypto.PrivKey{}, []uint64{}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, ErrNoSignatures().Result())

	// test wrong sequence number, now we return signature failed even it's seq number error.
	privs, seqs = []crypto.PrivKey{transaction1}, []uint64{0}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, ErrUnverifiedBytes(
		"signature verification failed, chain-id:Lino, seq:1").Result())

	// test wrong priv key
	privs, seqs = []crypto.PrivKey{transaction2}, []uint64{1}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, acc.ErrCheckAuthenticatePubKeyOwner(user1).Result())

	// test wrong sig number
	privs, seqs = []crypto.PrivKey{transaction2, transaction1}, []uint64{2, 0}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, ErrWrongNumberOfSigners().Result())
}

// Test grant authentication.
func (suite *AnteTestSuite) TestGrantAuthenticationTx() {
	// keys and username
	_, transaction1, _, user1 := suite.createTestAccount("user1")
	_, transaction2, post2, user2 := suite.createTestAccount("user2")
	_, transaction3, post3, user3 := suite.createTestAccount("user3")

	// msg and signatures
	var tx sdk.Tx
	msg := newTestMsg(user1)

	// test valid transaction
	privs, seqs := []crypto.PrivKey{transaction1}, []uint64{0}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkValidTx(tx)
	seq, err := suite.am.GetSequence(suite.ctx, user1)
	suite.Nil(err)
	suite.Equal(seq, uint64(1))

	// test wrong priv key
	privs, seqs = []crypto.PrivKey{transaction2}, []uint64{1}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, acc.ErrCheckAuthenticatePubKeyOwner(user1).Result())

	privs, seqs = []crypto.PrivKey{post2}, []uint64{1}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, acc.ErrCheckAuthenticatePubKeyOwner(user1).Result())

	err = suite.am.AuthorizePermission(suite.ctx, user1, user2, 3600, types.AppPermission, types.NewCoinFromInt64(0))
	suite.Nil(err)

	// should still fail by using transaction key
	privs, seqs = []crypto.PrivKey{transaction2}, []uint64{1}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, acc.ErrCheckAuthenticatePubKeyOwner(user1).Result())

	// should pass authentication check after grant the app permission
	privs, seqs = []crypto.PrivKey{post2}, []uint64{1}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkValidTx(tx)
	seq, err = suite.am.GetSequence(suite.ctx, user2)
	suite.Nil(err)
	suite.Equal(seq, uint64(0))
	seq, err = suite.am.GetSequence(suite.ctx, user1)
	suite.Nil(err)
	suite.Equal(seq, uint64(2))

	suite.ctx = suite.ctx.WithBlockHeader(abci.Header{
		ChainID: "Lino", Height: 2,
		Time: suite.ctx.BlockHeader().Time.Add(time.Duration(3601) * time.Second)})
	suite.checkInvalidTx(tx, acc.ErrCheckAuthenticatePubKeyOwner(user1).Result())

	// test pre authorization permission
	err = suite.am.AuthorizePermission(suite.ctx, user1, user3, 3600, types.PreAuthorizationPermission, types.NewCoinFromInt64(100))
	suite.Nil(err)
	msg.Permission = types.PreAuthorizationPermission
	privs, seqs = []crypto.PrivKey{post3}, []uint64{2}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, acc.ErrCheckAuthenticatePubKeyOwner(user1).Result())

	privs, seqs = []crypto.PrivKey{transaction3}, []uint64{2}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkValidTx(tx)
	seq, err = suite.am.GetSequence(suite.ctx, user3)
	suite.Nil(err)
	suite.Equal(seq, uint64(0))
	seq, err = suite.am.GetSequence(suite.ctx, user1)
	suite.Nil(err)
	suite.Equal(seq, uint64(3))

	// test pre authorization exceeds limitation
	msg.Amount = types.NewCoinFromInt64(100)
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(
		tx,
		acc.ErrPreAuthAmountInsufficient(
			user3, msg.Amount.Minus(types.NewCoinFromInt64(10)), msg.Amount).Result())

}

// Test various error cases in the AnteHandler control flow.
func (suite *AnteTestSuite) TestTPSCapacity() {
	// keys and username
	_, transaction1, _, user1 := suite.createTestAccount("user1")

	// msg and signatures
	var tx sdk.Tx
	msg := newTestMsg(user1)

	// test valid transaction
	privs, seqs := []crypto.PrivKey{transaction1}, []uint64{0}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkValidTx(tx)

	seq, err := suite.am.GetSequence(suite.ctx, user1)
	suite.Nil(err)
	suite.Equal(seq, uint64(1))

	suite.ctx = suite.ctx.WithBlockHeader(
		abci.Header{ChainID: "Lino", Height: 2, Time: time.Now(), NumTxs: 1000})
	suite.gm.SetLastBlockTime(suite.ctx, time.Now().Unix()-1)
	suite.gm.UpdateTPS(suite.ctx)

	seqs = []uint64{1}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkValidTx(tx)
	seqs = []uint64{2}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, acc.ErrAccountTPSCapacityNotEnough(user1).Result())
}

// before BlockchainUpgrade1Update1Height donation cost bandwidth.
func (suite *AnteTestSuite) TestTPSCapacityDonationBeforeUpdate1() {
	// keys and username
	_, transaction1, _, user1 := suite.createTestAccount("user1")
	suite.createTestAccount("user2")
	suite.createTestPost("post1", "user2")

	// donation msg and signatures
	var tx sdk.Tx
	msg := posttypes.NewDonateMsg("user1", types.LNO("1"), "user2", "post1", "", "memee")

	// test valid transaction
	privs, seqs := []crypto.PrivKey{transaction1}, []uint64{0}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkValidTx(tx)

	seq, err := suite.am.GetSequence(suite.ctx, user1)
	suite.Nil(err)
	suite.Equal(seq, uint64(1))

	suite.ctx = suite.ctx.WithBlockHeader(
		abci.Header{ChainID: "Lino", Height: 2, Time: time.Now(), NumTxs: 1000})
	suite.gm.SetLastBlockTime(suite.ctx, time.Now().Unix()-1)
	suite.gm.UpdateTPS(suite.ctx)

	seqs = []uint64{1}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkValidTx(tx)
	seqs = []uint64{2}
	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
	suite.checkInvalidTx(tx, acc.ErrAccountTPSCapacityNotEnough(user1).Result())
}

// // after BlockchainUpgrade1Update1Height, Test Donation message > NoTPSLimitDonationMin will no check or cost bandwidth.
// func (suite *AnteTestSuite) TestTPSCapacityDonationAfterUpdate1() {
// 	// keys and username
// 	_, transaction1, _, user1 := suite.createTestAccount("user1")
// 	suite.createTestAccount("user2")
// 	suite.createTestPost("post1", "user2")

// 	// donation msg and signatures
// 	var tx sdk.Tx
// 	msg := post.NewDonateMsg("user1", types.LNO("1"), "user2", "post1", "", "memee")

// 	// test valid transaction
// 	privs, seqs := []crypto.PrivKey{transaction1}, []uint64{0}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkValidTx(tx)

// 	seq, err := suite.am.GetSequence(suite.ctx, user1)
// 	suite.Nil(err)
// 	suite.Equal(seq, uint64(1))

// 	suite.ctx = suite.ctx.WithBlockHeader(
// 		abci.Header{ChainID: "Lino", Height: types.BlockchainUpgrade1Update1Height,
// 			Time: time.Now(), NumTxs: 1000})
// 	suite.gm.SetLastBlockTime(suite.ctx, time.Now().Unix()-1)
// 	suite.gm.UpdateTPS(suite.ctx)

// 	seqs = []uint64{1}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkValidTx(tx)
// 	seqs = []uint64{2}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkValidTx(tx)

// 	// then normal messages are still blocked by TPS limit
// 	testMsg := newTestMsg(user1)
// 	seqs = []uint64{3}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{testMsg}, privs, seqs)
// 	suite.checkValidTx(tx) // first one OK
// 	seqs = []uint64{4}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{testMsg}, privs, seqs)
// 	suite.checkInvalidTx(tx, acc.ErrAccountTPSCapacityNotEnough(user1).Result())

// 	// BUG EXPECTED
// 	// Invalid amount donation will still pass.
// 	msg = post.NewDonateMsg("user1", types.LNO("100000"), "user2", "post1", "", "memee")
// 	privs, seqs = []crypto.PrivKey{transaction1}, []uint64{4}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkValidTx(tx)
// }

// // after BlockchainUpgrade1Update4Height, Test Donation message > NoTPSLimitDonationMin
// // and with enough saving will no check or cost bandwidth.
// func (suite *AnteTestSuite) TestTPSCapacityDonationAfterUpdate4() {
// 	// keys and username
// 	_, transaction1, _, user1 := suite.createTestAccount("user1")
// 	suite.createTestAccount("user2")
// 	suite.createTestPost("post1", "user2")

// 	// donation msg and signatures
// 	var tx sdk.Tx
// 	msg := post.NewDonateMsg("user1", types.LNO("1"), "user2", "post1", "", "memee")

// 	// test valid transaction
// 	privs, seqs := []crypto.PrivKey{transaction1}, []uint64{0}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkValidTx(tx)

// 	seq, err := suite.am.GetSequence(suite.ctx, user1)
// 	suite.Nil(err)
// 	suite.Equal(seq, uint64(1))

// 	suite.ctx = suite.ctx.WithBlockHeader(
// 		abci.Header{ChainID: "Lino", Height: types.BlockchainUpgrade1Update4Height,
// 			Time: time.Now(), NumTxs: 1000})
// 	suite.gm.SetLastBlockTime(suite.ctx, time.Now().Unix()-1)
// 	suite.gm.UpdateTPS(suite.ctx)

// 	seqs = []uint64{1}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkValidTx(tx)
// 	seqs = []uint64{2}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkValidTx(tx)

// 	// first one with wrong permlink, ok, but will cost TPS
// 	msg = post.NewDonateMsg("user1", types.LNO("1"), "user2", "postNotExist", "", "memee")
// 	privs, seqs = []crypto.PrivKey{transaction1}, []uint64{3}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkValidTx(tx) // first one cost TPS

// 	// second one shall be blocked.
// 	testMsg := newTestMsg(user1)
// 	seqs = []uint64{4}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{testMsg}, privs, seqs)
// 	suite.checkInvalidTx(tx, acc.ErrAccountTPSCapacityNotEnough(user1).Result())

// 	// Invalid amount donation will not pass now.
// 	msg = post.NewDonateMsg("user1", types.LNO("100000000"), "user2", "post1", "", "memee")
// 	privs, seqs = []crypto.PrivKey{transaction1}, []uint64{4}
// 	tx = newTestTx(suite.ctx, []sdk.Msg{msg}, privs, seqs)
// 	suite.checkInvalidTx(tx, acc.ErrAccountTPSCapacityNotEnough(user1).Result())
// }

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, &AnteTestSuite{})
}
