package queue

// result type
var ResultTypeTransaction = "TRANSACTION"
var ResultTypeConflictHeader = "CONFLICT_HEADER"
var ResultTypeLabel = "LABEL"
var ResultTypeDataLabel = "DATE_LABEL"

// Conflict types
var ConflictTypeNext = "HasNext"
var ConflictTypeNone = "None"
var ConflictTypeEnd = "End"

// transaction type
var TransactionTypeTransfer = "Transfer"
var TransactionTypeSettingsChange = "SettingsChange"
var TransactionTypeCustom = "Custom"
var TransactionTypeCreation = "Creation"

// Transfer Info
var TransferTypeNative = "NATIVE_COIN"
var TransferTypeERC20 = "ERC20"
var TransferTypeERC721 = "ERC721"

// SettingsChange
var SetFallbackHandler = "SET_FALLBACK_HANDLER"
var AddOnwer = "ADD_OWNER"
var RemoveOwner = "REMOVE_OWNER"
var SwapOwner = "SWAP_OWNER"
var ChangeThreshold = "CHANGE_THRESHOLD"
var ChangeImplementation = "CHANGE_IMPLEMENTATION"
var EnableModule = "ENABLE_MODULE"
var DisableModule = "DISABLE_MODULE"
var SetGuard = "SET_GUARD"
var DeleteGuard = "DELETE_GUARD"
