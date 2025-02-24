package brahma

const (
	RegisterExecutorPath                           = "/v1/automations/executor"
	GetExecutorPath                                = "/v1/automations/executor/{address}/{chainID}"
	GetSubscriptionsByRegistryIDPath               = "/v1/automations/executor/{registryID}/subscriptions"
	ExecuteTaskPath                                = "/v1/automations/tasks/execute/{chainID}"
	GetTaskStatusPath                              = "/v1/relayer/tasks/status/{taskID}"
	GetSubscriptionsByConsoleAddressAndChainIDPath = "/v1/vendor/automations/subscriptions/console/{address}/{chainId}"
	GetConsoleAccountsPath                         = "/v1/vendor/user/consoles/{eoa}"
)

const (
	TaskStatusCancelled  = "cancelled"
	TaskStatusExecuting  = "executing"
	TaskStatusSuccessful = "successful"
)

const (
	TaskTimeoutInSecond = 15
)

const (
	SubscriptionStatusActive   = 2
	SubscriptionStatusInactive = 4
)
