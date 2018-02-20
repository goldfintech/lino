package types

import (
	"fmt"
	"github.com/tendermint/go-wire"
)

type PostID []byte

type Post struct {
	Title        string        `json:"denom"`
	Category     []string      `json:"category"`
	Content      string        `json:"content"`
	Author       AccountName   `json:"author"`
	Sequence     int           `json:"sequence"`
	Parent       PostID        `json:"parent"` // non-empty if it is a comment.
	Source       PostID        `json:"source"` // non-empty if it is a reblog
	Created      uint64     `json:"created"`
	Metadata     JsonFormat    `json:"metadata"`
	LastUpdate   uint64     `json:"last_update"`
	LastActivity uint64     `json:"last_activitu"`
	AllowReplies bool          `json:"allow_replies"`
	AllowVotes   bool          `json:"allow_votes"`
	Reward       Coins         `json:"reward"`
	Comments     []PostID      `json:"comments"`
	Likes        []AccountName `json:"likes"`
	ViewCount    int           `json:"view_count"`
}

func (post Post) String() string {
	return fmt.Sprintf(`"author:%v, seq:%v, title:%v, content:%v, category:%v, parent:%v, created:%v, metadata:%v
		                , last update:%v, last activity:%v, allow replies:%v, allow votes:%v, reward:%v
		                , comments:%v, likes:%v", views:%v, source:%v`,
					   post.Author, post.Sequence, post.Title, post.Content, post.Category, post.Parent, post.Created, post.Metadata,
					   post.LastUpdate, post.LastActivity, post.AllowReplies, post.AllowVotes, post.Reward, post.Comments,
					   post.Likes, post.ViewCount, post.Source)
}

// Post id is computed by the address and sequence.
func GetPostID(addr []byte, seq int) PostID {
	return append(addr, wire.BinaryBytes(seq)...)
}
