package brahma

import (
	"context"
	"fmt"
	"github.com/Brahma-fi/go-safe/encoders"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/pye-org/console-strategies-common/pkg/abi/multisendcallonly"
	"time"

	"github.com/pye-org/console-strategies-common/pkg/abi/executorplugin"
	"github.com/pye-org/console-strategies-common/pkg/rpcregistry"
	"github.com/pye-org/console-strategies-common/pkg/util/bignumber"
)

type Console struct {
	client                IClient
	rpcRegistry           rpcregistry.IRegistry
	executorPluginAddress common.Address
}

func NewConsole(client IClient, rpcRegistry rpcregistry.IRegistry, executorPluginAddress common.Address) *Console {
	return &Console{
		client:                client,
		rpcRegistry:           rpcRegistry,
		executorPluginAddress: executorPluginAddress,
	}
}

// Execute executes a safe transaction and return task ID
func (c *Console) Execute(ctx context.Context, params *ExecuteParams) (string, error) {
	safeTx, err := encoders.GetEncodedSafeTx(
		common.Address{},
		params.MultiSendCallOnlyAddress,
		multisendcallonly.ABI,
		params.Transactions,
		params.ChainID,
	)
	if err != nil {
		return "", err
	}

	// Step 1: get executor nonce
	executorPluginCaller, err := c.newExecutorPluginCaller(params.ChainID)
	if err != nil {
		return "", err
	}

	nonce, err := executorPluginCaller.ExecutorNonce(
		&bind.CallOpts{Context: ctx},
		params.SubAccount,
		params.ExecutorAddress,
	)
	if err != nil {
		return "", err
	}

	data, err := hexutil.Decode(safeTx.Data.String())
	if err != nil {
		return "", err
	}

	// Step 2: get executable digest
	executableDigest, err := GetExecutableDigest(
		apitypes.TypedDataDomain{
			Name:              "ExecutorPlugin",
			Version:           "1.0",
			ChainId:           math.NewHexOrDecimal256(params.ChainID),
			VerifyingContract: c.executorPluginAddress.String(),
		},
		TypedDataExecutionMessage{
			Operation:      safeTx.Operation,
			To:             safeTx.To.Address(),
			Account:        params.SubAccount,
			Executor:       params.ExecutorAddress,
			GasToken:       common.HexToAddress(""),
			RefundReceiver: common.HexToAddress(""),
			Value:          bignumber.SetFromDecimal256(safeTx.Value),
			Nonce:          nonce,
			SafeTxGas:      bignumber.Zero,
			BaseGas:        bignumber.Zero,
			GasPrice:       bignumber.Zero,
			Data:           data,
		},
	)
	if err != nil {
		return "", err
	}
	// Step 3: sign the executable digest
	signature, err := params.Signer.Sign(ctx, executableDigest)
	if err != nil {
		return "", err
	}

	// Step 4: prepare and send the execute task request
	executeTaskRequestBody := &ExecuteTaskRequestBody{
		SubAccount:        params.SubAccount.Hex(),
		Executor:          params.ExecutorAddress.Hex(),
		ExecutorSignature: hexutil.Encode(signature),
		Executable: Executable{
			CallType: safeTx.Operation,
			To:       safeTx.To.Address().Hex(),
			Value:    safeTx.Value.String(),
			Data:     safeTx.Data.String(),
		},
	}

	taskId, err := c.client.ExecuteTask(
		ctx,
		params.ChainID,
		executeTaskRequestBody,
	)

	if err != nil {
		return taskId, err
	}

	return taskId, c.waitForTaskSuccess(ctx, taskId, TaskTimeoutInSecond*time.Second)
}

func (c *Console) newExecutorPluginCaller(chainID int64) (*executorplugin.ExecutorPluginCaller, error) {
	rpcClient, err := c.rpcRegistry.GetClient(chainID)
	if err != nil {
		return nil, err
	}

	return executorplugin.NewExecutorPluginCaller(c.executorPluginAddress, rpcClient)
}

func (c *Console) waitForTaskSuccess(ctx context.Context, taskID string, timeout time.Duration) error {
	if taskID == "" {
		return nil
	}

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout reached while waiting for task %s to succeed", taskID)
		case <-ticker.C:
			status, err := c.client.GetTaskStatus(ctx, taskID)
			if err != nil {
				continue
			}

			if status == nil {
				continue
			}

			if status.Status == TaskStatusSuccessful {
				return nil
			} else if status.Status == TaskStatusExecuting {
				continue
			} else if status.Status == TaskStatusCancelled {
				return fmt.Errorf("task %s was cancelled", taskID)
			} else {
				return fmt.Errorf("task %s failed with status %s", taskID, status.Status)
			}
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout reached while waiting for task %s to succeed", taskID)
		}
	}
}
