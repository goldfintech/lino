package types

const (
	// Decimals - Total decimals in Lino Blockchain
	// Used by both LNO and IDA.
	Decimals = 100000

	// KVStoreKey presents store which used by app
	MainKVStoreKey         = "main"
	AccountKVStoreKey      = "account"
	PostKVStoreKey         = "post"
	ValidatorKVStoreKey    = "validator"
	GlobalKVStoreKey       = "global"
	VoteKVStoreKey         = "vote"
	InfraKVStoreKey        = "infra"
	DeveloperKVStoreKey    = "developer"
	ParamKVStoreKey        = "param"
	ProposalKVStoreKey     = "proposal"
	ReputationV2KVStoreKey = "repv2"

	// Different permission level for msg
	UnknownPermission                = Permission(0)
	AppPermission                    = Permission(1)
	TransactionPermission            = Permission(2)
	ResetPermission                  = Permission(3)
	GrantAppPermission               = Permission(4)
	PreAuthorizationPermission       = Permission(5)
	AppAndPreAuthorizationPermission = Permission(6)

	// since upgrade2
	// signed by the app or its affiliated accounts.
	AppOrAffiliatedPermission = Permission(7)

	// Different proposal result
	ProposalNotPass = ProposalResult(0)
	ProposalPass    = ProposalResult(1)
	ProposalRevoked = ProposalResult(2)

	// Different proposal types
	ChangeParam       = ProposalType(0)
	ContentCensorship = ProposalType(1)
	ProtocolUpgrade   = ProposalType(2)

	// Different donation types
	DirectDeposit = DonationType(0)
	Inflation     = DonationType(1)

	// Different possible incomes
	TransferIn           = TransferDetailType(0)
	DonationIn           = TransferDetailType(1)
	ClaimReward          = TransferDetailType(2)
	ValidatorInflation   = TransferDetailType(3)
	DeveloperInflation   = TransferDetailType(4)
	InfraInflation       = TransferDetailType(5)
	VoteReturnCoin       = TransferDetailType(6)
	DelegationReturnCoin = TransferDetailType(7)
	ValidatorReturnCoin  = TransferDetailType(8)
	DeveloperReturnCoin  = TransferDetailType(9)
	InfraReturnCoin      = TransferDetailType(10)
	ProposalReturnCoin   = TransferDetailType(11)
	GenesisCoin          = TransferDetailType(12)
	ClaimInterest        = TransferDetailType(13)

	// Different possible outcomes
	TransferOut      = TransferDetailType(20)
	DonationOut      = TransferDetailType(21)
	Delegate         = TransferDetailType(22)
	VoterDeposit     = TransferDetailType(23)
	ValidatorDeposit = TransferDetailType(24)
	DeveloperDeposit = TransferDetailType(25)
	InfraDeposit     = TransferDetailType(26)
	ProposalDeposit  = TransferDetailType(27)

	// punishment type
	UnknownPunish      = PunishType(0)
	PunishByzantine    = PunishType(1)
	PunishAbsentCommit = PunishType(2)
	PunishDidntVote    = PunishType(3)

	// UsernameReCheck - UsernameReCheck is used to check user registration
	UsernameReCheck        = "^[a-z]([a-z0-9-\\.]){1,19}[a-z0-9]$"
	IllegalUsernameReCheck = "^[a-z0-9\\.-]*([-\\.]){2,}[a-z0-9\\.-]*$"

	// MinimumUsernameLength - minimum username length
	MinimumUsernameLength = 3

	// MaximumUsernameLength - maximum username length
	MaximumUsernameLength = 20

	// MaximumMemoLength - maximum length of memo
	MaximumMemoLength = 100

	// MaximumJSONMetaLength - maximum length of account JSON meta
	MaximumJSONMetaLength = 500

	// MaxPostTitleLength - maximum length of post title
	MaxPostTitleLength = 100

	// MaxPostContentLength - maximum length of post content
	MaxPostContentLength = 1000

	// MaxGranPermValiditySec - maximum validity period, 10 years
	MaxGranPermValiditySec = 10 * 3600 * 24 * 365

	// KeySeparator - separate different key component
	KeySeparator = "/"

	// HoursPerYear - as defined by a julian year of 365.25 days
	HoursPerYear = 8766

	// MinutesPerYear - as defined by a julian year of 365.25 days
	MinutesPerYear = HoursPerYear * 60

	// MinutesPerMonth - as defined by a julian year of 365.25 days
	MinutesPerMonth = MinutesPerYear / 12

	// MinutesPerDay - as defined by a julian year of 365.25 days
	MinutesPerDay = 60 * 24

	// MaximumSdkRatLength - maximum length of sdk.Dec can pass into blockchain
	MaximumSdkRatLength = 10

	// MaximumLinkIdentifier - maximum length of Links identifier
	MaximumLinkIdentifier = 20

	// MaximumLinkURL - maximum length of Links URL
	MaximumLinkURL = 100

	// MaximumLengthOfPostID - maximum length of post ID
	MaximumLengthOfPostID = 50

	// MaximumNumOfLinks - maximum number of links per post
	MaximumNumOfLinks = 10

	// MaximumLengthOfDeveloperWebsite - maximum length of developer website
	MaximumLengthOfDeveloperWebsite = 100

	// MaximumLengthOfDeveloperDesctiption - maximum length of developer description
	MaximumLengthOfDeveloperDesctiption = 1000

	// MaximumLengthOfAppMetadata - maximum length of developer App meta data
	MaximumLengthOfAppMetadata = 1000

	// MaximumLengthOfProposalReason - maximum length of proposal reason
	MaximumLengthOfProposalReason = 1000

	// InitAccountWithFullCoinDayMemo - init account with full coin day memo
	InitAccountWithFullCoinDayMemo = "open account deposit"

	// InitAccountRegisterDepositMemo - init account deposit fee memo
	InitAccountRegisterDepositMemo = "init deposit"

	// PermlinkSeparator - permlink separator
	PermlinkSeparator = "#"

	// ParamChangeTimeout - time in secs for ParamChange to happen.
	ParamChangeTimeout = 3600

	// BalanceHistoryBundleSize - bundle size for balance history
	BalanceHistoryBundleSize = 100

	// RewardHistoryBundleSize - bundle size for reward history
	RewardHistoryBundleSize = 100

	// CoinDayRecordIntervalSec - coin day record in the same interval bucket will be merged
	CoinDayRecordIntervalSec = 1200

	// TendermintValidatorPower - every validator has const power in tendermint engine.
	TendermintValidatorPower = 1000

	// BlockchainUpgrade1Update1Height - since this height, donation > 1 will not cost bandwidth.
	BlockchainUpgrade1Update1Height = 21610

	// BlockchainUpgrade1Update2Height - global growth rate is corrected.
	BlockchainUpgrade1Update2Height = 146000

	// BlockchainUpgrade1Update3Height - reward pool is adjusted.
	BlockchainUpgrade1Update3Height = 148000

	// BlockchainUpgrade1Update4Height - fix donation bandwidth check.
	BlockchainUpgrade1Update4Height = 386000

	// NoTPSLimitDonationMin - donation >= this value will not cost bandwidth, in coin.
	NoTPSLimitDonationMin = 100000
)
