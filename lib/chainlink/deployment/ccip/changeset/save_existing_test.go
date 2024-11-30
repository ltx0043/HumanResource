package changeset

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink/deployment"
	commonchangeset "github.com/smartcontractkit/chainlink/deployment/common/changeset"
	"github.com/smartcontractkit/chainlink/deployment/environment/memory"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
)

func TestSaveExistingCCIP(t *testing.T) {
	t.Parallel()
	lggr := logger.TestLogger(t)
	e := memory.NewMemoryEnvironment(t, lggr, zapcore.InfoLevel, memory.MemoryEnvironmentConfig{
		Bootstraps: 1,
		Chains:     2,
		Nodes:      4,
	})
	chains := e.AllChainSelectors()
	chain1 := chains[0]
	chain2 := chains[1]
	cfg := commonchangeset.ExistingContractsConfig{
		ExistingContracts: []commonchangeset.Contract{
			{
				Address:        common.BigToAddress(big.NewInt(1)),
				TypeAndVersion: deployment.NewTypeAndVersion(LinkToken, deployment.Version1_0_0),
				ChainSelector:  chain1,
			},
			{
				Address:        common.BigToAddress(big.NewInt(2)),
				TypeAndVersion: deployment.NewTypeAndVersion(WETH9, deployment.Version1_0_0),
				ChainSelector:  chain1,
			},
			{
				Address:        common.BigToAddress(big.NewInt(3)),
				TypeAndVersion: deployment.NewTypeAndVersion(TokenAdminRegistry, deployment.Version1_5_0),
				ChainSelector:  chain1,
			},
			{
				Address:        common.BigToAddress(big.NewInt(4)),
				TypeAndVersion: deployment.NewTypeAndVersion(RegistryModule, deployment.Version1_5_0),
				ChainSelector:  chain2,
			},
			{
				Address:        common.BigToAddress(big.NewInt(5)),
				TypeAndVersion: deployment.NewTypeAndVersion(Router, deployment.Version1_2_0),
				ChainSelector:  chain2,
			},
		},
	}

	output, err := commonchangeset.SaveExistingContracts(e, cfg)
	require.NoError(t, err)
	err = e.ExistingAddresses.Merge(output.AddressBook)
	require.NoError(t, err)
	state, err := LoadOnchainState(e)
	require.NoError(t, err)
	require.Equal(t, state.Chains[chain1].LinkToken.Address(), common.BigToAddress(big.NewInt(1)))
	require.Equal(t, state.Chains[chain1].Weth9.Address(), common.BigToAddress(big.NewInt(2)))
	require.Equal(t, state.Chains[chain1].TokenAdminRegistry.Address(), common.BigToAddress(big.NewInt(3)))
	require.Equal(t, state.Chains[chain2].RegistryModule.Address(), common.BigToAddress(big.NewInt(4)))
	require.Equal(t, state.Chains[chain2].Router.Address(), common.BigToAddress(big.NewInt(5)))
}
