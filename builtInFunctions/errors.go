package builtInFunctions

import (
	"errors"
)

// ErrNilAccountsAdapter defines the error when trying to use a nil AccountsAddapter
var ErrNilAccountsAdapter = errors.New("nil AccountsAdapter")

// ErrInsufficientFunds signals the funds are insufficient for the move balance operation but the
// transaction fee is covered by the current balance
var ErrInsufficientFunds = errors.New("insufficient funds")

// ErrNilValue signals the value is nil
var ErrNilValue = errors.New("nil value")

// ErrNilBlockHeader signals that an operation has been attempted to or with a nil block header
var ErrNilBlockHeader = errors.New("nil block header")

// ErrNilMarshalizer signals that an operation has been attempted to or with a nil Marshalizer implementation
var ErrNilMarshalizer = errors.New("nil Marshalizer")

// ErrInvalidRcvAddr signals that an invalid receiver address was provided
var ErrInvalidRcvAddr = errors.New("invalid receiver address")

// ErrInvalidSndAddr signals that an invalid sender address was provided
var ErrInvalidSndAddr = errors.New("invalid sender address")

// ErrNegativeValue signals that a negative value has been detected and it is not allowed
var ErrNegativeValue = errors.New("negative value")

// ErrNilShardCoordinator signals that an operation has been attempted to or with a nil shard coordinator
var ErrNilShardCoordinator = errors.New("nil shard coordinator")

// ErrNilSingleSigner signals that a nil single signer is used
var ErrNilSingleSigner = errors.New("nil single signer")

// ErrNilDataToProcess signals that nil data was provided
var ErrNilDataToProcess = errors.New("nil data to process")

// ErrNilPoolsHolder signals that an operation has been attempted to or with a nil pools holder object
var ErrNilPoolsHolder = errors.New("nil pools holder")

// ErrNilShardedDataCacherNotifier signals that a nil sharded data cacher notifier has been provided
var ErrNilShardedDataCacherNotifier = errors.New("nil sharded data cacher notifier")

// ErrNilTxProcessor signals that a nil transactions processor was used
var ErrNilTxProcessor = errors.New("nil transactions processor")

// ErrNilForkDetector signals that the fork detector is nil
var ErrNilForkDetector = errors.New("nil fork detector")

// ErrWrongTypeAssertion signals that an type assertion failed
var ErrWrongTypeAssertion = errors.New("wrong type assertion")

// ErrHigherRoundInBlock signals that a block with higher round than permitted has been provided
var ErrHigherRoundInBlock = errors.New("higher round in block")

// ErrHigherNonceInBlock signals that a block with higher nonce than permitted has been provided
var ErrHigherNonceInBlock = errors.New("higher nonce in block")

// ErrNilSCDestAccount signals that destination account is nil
var ErrNilSCDestAccount = errors.New("nil destination SC account")

// ErrNilScAddress signals that a nil smart contract address has been provided
var ErrNilScAddress = errors.New("nil SC address")

// ErrNilPreProcessor signals that preprocessors is nil
var ErrNilPreProcessor = errors.New("preprocessor is nil")

// ErrInvalidPeerAccount signals that a peer account is invalid
var ErrInvalidPeerAccount = errors.New("invalid peer account")

// ErrNilEpochHandler signals that a nil epoch handler was provided
var ErrNilEpochHandler = errors.New("nil epoch handler")

// ErrNilTxValidator signals that a nil tx validator has been provided
var ErrNilTxValidator = errors.New("nil transaction validator")

// ErrNilHdrValidator signals that a nil header validator has been provided
var ErrNilHdrValidator = errors.New("nil header validator")

// ErrNilBlockTracker signals that a nil block tracker was provided
var ErrNilBlockTracker = errors.New("nil block tracker")

// ErrNotEnoughGas signals that not enough gas has been provided
var ErrNotEnoughGas = errors.New("not enough gas was sent in the transaction")

// ErrNilHeaderIntegrityVerifier signals that a nil header integrity verifier has been provided
var ErrNilHeaderIntegrityVerifier = errors.New("nil header integrity verifier")

// ErrNilBadTxHandler signals that bad tx handler is nil
var ErrNilBadTxHandler = errors.New("nil bad tx handler")

// ErrNotEpochStartBlock signals that block is not of type epoch start
var ErrNotEpochStartBlock = errors.New("not epoch start block")

// ErrInvalidArguments signals that invalid arguments were given to process built-in function
var ErrInvalidArguments = errors.New("invalid arguments to process built-in function")

// ErrOperationNotPermitted signals that operation is not permitted
var ErrOperationNotPermitted = errors.New("operation in account not permitted")

// ErrInvalidAddressLength signals that address length is invalid
var ErrInvalidAddressLength = errors.New("invalid address length")

// ErrInvalidShardCacherIdentifier signals an invalid identifier
var ErrInvalidShardCacherIdentifier = errors.New("invalid identifier for shard cacher")

// ErrNilVmInput signals that provided vm input is nil
var ErrNilVmInput = errors.New("nil vm input")

// ErrNilDnsAddresses signals that nil dns addresses map was provided
var ErrNilDnsAddresses = errors.New("nil dns addresses map")

// ErrCallerIsNotTheDNSAddress signals that called address is not the DNS address
var ErrCallerIsNotTheDNSAddress = errors.New("not a dns address")

// ErrUserNameChangeIsDisabled signals the user name change is not allowed
var ErrUserNameChangeIsDisabled = errors.New("user name change is disabled")

// ErrNilBalanceComputationHandler signals that a nil balance computation handler has been provided
var ErrNilBalanceComputationHandler = errors.New("nil balance computation handler")

// ErrBuiltInFunctionCalledWithValue signals that builtin function was called with value that is not allowed
var ErrBuiltInFunctionCalledWithValue = errors.New("built in function called with tx value is not allowed")

// ErrAccountNotPayable will be sent when trying to send money to a non-payable account
var ErrAccountNotPayable = errors.New("sending value to non payable contract")

// ErrNilUserAccount signals that nil user account was provided
var ErrNilUserAccount = errors.New("nil user account")

// ErrAddressIsNotESDTSystemSC signals that destination is not a system sc address
var ErrAddressIsNotESDTSystemSC = errors.New("destination is not system sc address")

// ErrOnlySystemAccountAccepted signals that only system account is accepted
var ErrOnlySystemAccountAccepted = errors.New("only system account is accepted")

// ErrNilGlobalSettingsHandler signals that nil pause handler has been provided
var ErrNilGlobalSettingsHandler = errors.New("nil pause handler")

// ErrNilRolesHandler signals that nil roles handler has been provided
var ErrNilRolesHandler = errors.New("nil roles handler")

// ErrESDTTokenIsPaused signals that esdt token is paused
var ErrESDTTokenIsPaused = errors.New("esdt token is paused")

// ErrESDTIsFrozenForAccount signals that account is frozen for given esdt token
var ErrESDTIsFrozenForAccount = errors.New("account is frozen for this esdt token")

// ErrCannotWipeAccountNotFrozen signals that account isn't frozen so the wipe is not possible
var ErrCannotWipeAccountNotFrozen = errors.New("cannot wipe because the account is not frozen for this esdt token")

// ErrNilPayableHandler signals that nil payableHandler was provided
var ErrNilPayableHandler = errors.New("nil payableHandler was provided")

// ErrNilFallbackHeaderValidator signals that a nil fallback header validator has been provided
var ErrNilFallbackHeaderValidator = errors.New("nil fallback header validator")

// ErrNilTransactionVersionChecker signals that provided transaction version checker is nil
var ErrNilTransactionVersionChecker = errors.New("nil transaction version checker")

// ErrNilOrEmptyList signals that a nil or empty list was provided
var ErrNilOrEmptyList = errors.New("nil or empty provided list")

// ErrActionNotAllowed signals that action is not allowed
var ErrActionNotAllowed = errors.New("action is not allowed")

// ErrOnlyFungibleTokensHaveBalanceTransfer signals that only fungible tokens have balance transfer
var ErrOnlyFungibleTokensHaveBalanceTransfer = errors.New("only fungible tokens have balance transfer")

// ErrNFTTokenDoesNotExist signals that NFT token does not exist
var ErrNFTTokenDoesNotExist = errors.New("NFT token does not exist")

// ErrNFTDoesNotHaveMetadata signals that NFT does not have metadata
var ErrNFTDoesNotHaveMetadata = errors.New("NFT does not have metadata")

// ErrInvalidNFTQuantity signals that invalid NFT quantity was provided
var ErrInvalidNFTQuantity = errors.New("invalid NFT quantity")

// ErrNewNFTDataOnSenderAddress signals that a new NFT data was found on the sender address
var ErrNewNFTDataOnSenderAddress = errors.New("new NFT data on sender")

// ErrNilContainerElement signals when trying to add a nil element in the container
var ErrNilContainerElement = errors.New("element cannot be nil")

// ErrInvalidContainerKey signals that an element does not exist in the container's map
var ErrInvalidContainerKey = errors.New("element does not exist in container")

// ErrContainerKeyAlreadyExists signals that an element was already set in the container's map
var ErrContainerKeyAlreadyExists = errors.New("provided key already exists in container")

// ErrWrongTypeInContainer signals that a wrong type of object was found in container
var ErrWrongTypeInContainer = errors.New("wrong type of object inside container")

// ErrEmptyFunctionName signals that an empty function name has been provided
var ErrEmptyFunctionName = errors.New("empty function name")

// ErrInsufficientQuantityESDT signals the funds are insufficient for the ESDT transfer
var ErrInsufficientQuantityESDT = errors.New("insufficient quantity")

// ErrNilESDTNFTStorageHandler signals that a nil nft storage handler has been provided
var ErrNilESDTNFTStorageHandler = errors.New("nil esdt nft storage handler")

// ErrNilTransactionHandler signals that a nil transaction handler has been provided
var ErrNilTransactionHandler = errors.New("nil transaction handler")
