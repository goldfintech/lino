package manager

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	linotypes "github.com/lino-network/lino/types"
)

type PostManagerEventTestSuite struct {
	*PostManagerTestSuite
}

func TestPostManagerEventTestSuite(t *testing.T) {
	suite.Run(t, &PostManagerEventTestSuite{
		PostManagerTestSuite: new(PostManagerTestSuite),
	})
}

func (suite *PostManagerEventTestSuite) TestRewardEvent() {
	user1 := suite.user1
	user2 := suite.user2
	app1 := suite.app1
	postID := "post1"
	err := suite.pm.CreatePost(suite.Ctx, user1, postID, app1, "content", "title")
	suite.Require().Nil(err)

	testCases := []struct {
		testName string
		event    RewardEvent
		reward   linotypes.Coin
		hasDev   bool
		hasPost  bool
	}{
		{
			testName: "OK",
			event: RewardEvent{
				PostAuthor: user1,
				PostID:     postID,
				Consumer:   user2,
				Evaluate:   linotypes.NewMiniDollar(100),
				FromApp:    app1,
			},
			reward:  linotypes.NewCoinFromInt64(3333),
			hasDev:  true,
			hasPost: true,
		},
		{
			testName: "PostDeleted",
			event: RewardEvent{
				PostAuthor: user1,
				PostID:     "deletedpost",
				Consumer:   user2,
				Evaluate:   linotypes.NewMiniDollar(100),
				FromApp:    app1,
			},
			reward:  linotypes.NewCoinFromInt64(3333),
			hasDev:  true,
			hasPost: false,
		},
		{
			testName: "NoDev",
			event: RewardEvent{
				PostAuthor: user1,
				PostID:     postID,
				Consumer:   user2,
				Evaluate:   linotypes.NewMiniDollar(100),
				FromApp:    user2,
			},
			reward:  linotypes.NewCoinFromInt64(3333),
			hasDev:  false,
			hasPost: true,
		},
	}

	for _, tc := range testCases {
		if tc.hasPost {
			suite.global.On("GetRewardAndPopFromWindow", mock.Anything, tc.event.Evaluate).Return(
				tc.reward, nil,
			).Once()
			suite.am.On("AddSavingCoin", mock.Anything,
				tc.event.PostAuthor, tc.reward, tc.event.PostAuthor, "", linotypes.ClaimReward,
			).Return(nil).Once()
			if tc.hasDev {
				suite.dev.On(
					"ReportConsumption", mock.Anything, tc.event.FromApp, tc.reward).Return(nil).Once()
			}
		}
		err := tc.event.Execute(suite.Ctx, suite.pm)
		suite.Nil(err)
		suite.global.AssertExpectations(suite.T())
		suite.am.AssertExpectations(suite.T())
		suite.dev.AssertExpectations(suite.T())
	}
}
