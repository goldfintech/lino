package client

import "github.com/spf13/cobra"

// nolint
const (
	FlagChainID   = "chain-id"
	FlagNode      = "node"
	FlagHeight    = "height"
	FlagTrustNode = "trust-node"
	FlagName      = "name"
	FlagSequence  = "sequence"
	FlagFee       = "fee"
	FlagPrivKey   = "priv-key"
	FlagPubKey    = "pub-key"

	// Account
	FlagIsFollow = "is-follow"
	FlagFollowee = "followee"
	FlagFollower = "follower"
	FlagSender   = "sender"
	FlagReceiver = "receiver"
	FlagAmount   = "amount"
	FlagMemo     = "memo"

	// Developer
	FlagDeveloper   = "developer"
	FlagDeposit     = "deposit"
	FlagWebsite     = "website"
	FlagDescription = "description"
	FlagAppMeta     = "appmeta"
	FlagUser        = "user"
	FlagRevokeFrom  = "revoke-from"
	FlagReferrer    = "referrer"
	FlagSeconds     = "seconds"
	FlagPermission  = "permission"
	FlagGrantAmount = "grant-amount"

	// Infra
	FlagProvider = "provider"
	FlagUsage    = "usage"

	// Vote
	FlagVoter      = "voter"
	FlagProposalID = "proposal-id"
	FlagResult     = "result"
	FlagLink       = "link"
)

// LineBreak can be included in a command list to provide a blank line
// to help with readability
var LineBreak = &cobra.Command{Run: func(*cobra.Command, []string) {}}

// GetCommands adds common flags to query commands
func GetCommands(cmds ...*cobra.Command) []*cobra.Command {
	for _, c := range cmds {
		c.Flags().Bool(FlagTrustNode, false, "Don't verify proofs for responses")
		c.Flags().String(FlagChainID, "", "Chain ID of tendermint node")
		c.Flags().String(FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
		c.Flags().Int64(FlagHeight, 0, "block height to query, omit to get most recent provable block")
	}
	return cmds
}

// PostCommands adds common flags for commands to post tx
func PostCommands(cmds ...*cobra.Command) []*cobra.Command {
	for _, c := range cmds {
		c.Flags().Int64(FlagSequence, 0, "Sequence number to sign the tx")
		c.Flags().String(FlagChainID, "", "Chain ID of tendermint node")
		c.Flags().String(FlagPrivKey, "", "Private key to sign the transaction")
		c.Flags().String(FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
	}
	return cmds
}
