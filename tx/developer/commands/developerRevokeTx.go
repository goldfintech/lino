package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/lino-network/lino/client"
	developer "github.com/lino-network/lino/tx/developer"

	"github.com/cosmos/cosmos-sdk/wire"
)

func DeveloperRevokeTxCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "developer-revoke",
		Short: "developer revoke",
		RunE:  sendDeveloperRevokeTx(cdc),
	}
	cmd.Flags().String(FlagDeveloper, "", "developer name of this transaction")
	return cmd
}

// send developer revoke transaction to the blockchain
func sendDeveloperRevokeTx(cdc *wire.Codec) client.CommandTxCallback {
	return func(cmd *cobra.Command, args []string) error {
		ctx := client.NewCoreContextFromViper()
		username := viper.GetString(FlagDeveloper)
		msg := developer.NewDeveloperRevokeMsg(username)

		// build and sign the transaction, then broadcast to Tendermint
		res, signErr := ctx.SignBuildBroadcast(username, msg, cdc)
		if signErr != nil {
			return signErr
		}

		fmt.Printf("Committed at block %d. Hash: %s\n", res.Height, res.Hash.String())
		return nil
	}
}
