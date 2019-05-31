package repv2

import (
	"math/big"
	"testing"

	"github.com/lino-network/lino/x/reputation/repv2/internal"
	"github.com/stretchr/testify/suite"
)

type StoreTestSuite struct {
	suite.Suite
	mockDB *internal.MockStore
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, &StoreTestSuite{})
}

func (suite *StoreTestSuite) SetupTest() {
	suite.mockDB = internal.NewMockStore()
}

func (suite *StoreTestSuite) TestPrefix() {
	suite.Equal(append(repUserMetaPrefix, []byte("qwe")...), getUserMetaKey("qwe"))
	suite.Equal(append(repRoundMetaPrefix, []byte("3")...), getRoundMetaKey(3))
	suite.Equal(append(repRoundPostMetaPrefix,
		[]byte{byte('b'), byte('/'), byte('x'), byte('y')}...), getRoundPostMetaKey(11, "xy"))
	suite.Equal(append(repRoundPostMetaPrefix,
		[]byte{byte('z'), byte('/'), byte('x'), byte('y')}...), getRoundPostMetaKey(35, "xy"))
	suite.Equal(append(repRoundPostMetaPrefix,
		[]byte{byte('2'), byte('f'), byte('/'), byte('a'), byte('b'), byte('c'), byte('d')}...),
		getRoundPostMetaKey(87, "abcd"))
	suite.Equal(repGameMetaPrefix, getGameKey())
}

func (suite *StoreTestSuite) TestInitValues() {
	store := NewReputationStore(suite.mockDB)

	// user
	user := store.GetUserMeta("no")
	suite.Equal(big.NewInt(InitialReputation), user.Reputation)
	suite.Equal(int64(0), user.LastSettledRound)
	suite.Equal(int64(0), user.LastDonationRound)
	suite.Empty(user.Unsettled)

	// round meta
	round := store.GetRoundMeta(333)
	suite.Empty(round.Result)
	suite.Equal(big.NewInt(0), round.SumIF)
	suite.Equal(int64(0), round.StartAt)
	suite.Empty(round.TopN)

	// current round
	suite.Equal(int64(1), store.GetCurrentRound())

	// post
	post := store.GetRoundPostMeta(33, "xxx")
	suite.Equal(big.NewInt(0), post.SumIF)

	// game
	game := store.GetGameMeta()
	suite.Equal(int64(1), game.CurrentRound)
}

func (suite *StoreTestSuite) TestStoreGetSet() {
	store := NewReputationStore(suite.mockDB)

	user1 := "test"
	user2 := "test2"
	post1 := "post1"
	post2 := "post2"

	u1 := &userMeta{
		Reputation:        big.NewInt(123),
		LastSettledRound:  3,
		LastDonationRound: 3,
		Unsettled: []Donation{
			Donation{Pid: post1, Amount: big.NewInt(3), Impact: big.NewInt(2)},
			Donation{Pid: post2, Amount: big.NewInt(4), Impact: big.NewInt(7)},
		}}
	u2 := &userMeta{
		Reputation:        big.NewInt(456),
		LastSettledRound:  4,
		LastDonationRound: 5,
		Unsettled: []Donation{
			Donation{Pid: post2, Amount: big.NewInt(6), Impact: big.NewInt(11)},
		}}
	store.SetUserMeta(user1, u2)
	store.SetUserMeta(user1, u1)
	store.SetUserMeta(user2, u2)
	defer func() {
		suite.Equal(u1, store.GetUserMeta(user1))
		suite.Equal(u2, store.GetUserMeta(user2))
	}()

	round1 := &roundMeta{
		Result:  []Pid{"123", "4ed6", "xzz"},
		SumIF:   big.NewInt(33333),
		StartAt: 324324,
		TopN:    nil,
	}
	round2 := &roundMeta{
		Result:  []Pid{"xzz"},
		SumIF:   big.NewInt(234134),
		StartAt: 342,
		TopN: []PostIFPair{
			PostIFPair{
				Pid:   post1,
				SumIF: big.NewInt(234235311),
			},
		},
	}
	store.SetRoundMeta(3, round2)
	store.SetRoundMeta(3, round1)
	store.SetRoundMeta(4, round2)
	defer func() {
		suite.Equal(round1, store.GetRoundMeta(3))
		suite.Equal(round2, store.GetRoundMeta(4))
	}()

	rp1 := &roundPostMeta{
		SumIF: big.NewInt(342),
	}
	rp2 := &roundPostMeta{
		SumIF: big.NewInt(666),
	}

	store.SetRoundPostMeta(123, post1, rp2)
	store.SetRoundPostMeta(123, post1, rp1)
	store.SetRoundPostMeta(342, post1, rp2)
	defer func() {
		suite.Equal(rp1, store.GetRoundPostMeta(123, post1))
		suite.Equal(rp2, store.GetRoundPostMeta(342, post1))
	}()

	store.SetGameMeta(&gameMeta{CurrentRound: 33})
	store.SetGameMeta(&gameMeta{CurrentRound: 443})
	defer func() {
		suite.Equal(int64(443), store.GetCurrentRound())
		suite.Equal(&gameMeta{CurrentRound: 443}, store.GetGameMeta())
	}()
}

func (suite *StoreTestSuite) TestStoreImportExporter() {
	store := NewReputationStore(suite.mockDB)

	user1 := "test"
	user2 := "test2"
	post1 := "post1"
	post2 := "post2"

	u1 := &userMeta{
		Reputation:        big.NewInt(123),
		LastSettledRound:  3,
		LastDonationRound: 3,
		Unsettled: []Donation{
			Donation{Pid: post1, Amount: big.NewInt(3), Impact: big.NewInt(2)},
			Donation{Pid: post2, Amount: big.NewInt(4), Impact: big.NewInt(7)},
		}}
	u2 := &userMeta{
		Reputation:        big.NewInt(456),
		LastSettledRound:  4,
		LastDonationRound: 5,
		Unsettled: []Donation{
			Donation{Pid: post2, Amount: big.NewInt(6), Impact: big.NewInt(11)},
		}}
	store.SetUserMeta(user1, u1)
	store.SetUserMeta(user2, u2)

	// export data
	data := store.Export()
	db2 := internal.NewMockStore()
	store2 := NewReputationStore(db2)
	store2.Import(data)
	suite.Equal(u1.Reputation, store2.GetUserMeta(user1).Reputation)
	suite.Equal(u2.Reputation, store2.GetUserMeta(user2).Reputation)
	suite.Equal(int64(0), store2.GetUserMeta(user1).LastDonationRound)
	suite.Equal(int64(0), store2.GetUserMeta(user2).LastDonationRound)
	suite.Equal(int64(0), store2.GetUserMeta(user1).LastSettledRound)
	suite.Equal(int64(0), store2.GetUserMeta(user2).LastSettledRound)
	suite.Empty(store2.GetUserMeta(user1).Unsettled)
	suite.Empty(store2.GetUserMeta(user2).Unsettled)
}
