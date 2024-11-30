package smoke

import (
	"context"
	"encoding/binary"
	"errors"
	"math/big"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-ccip/commit/merkleroot/rmn/types"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-testing-framework/lib/utils/osutil"
	"github.com/smartcontractkit/chainlink-testing-framework/lib/utils/testcontext"
	"github.com/smartcontractkit/chainlink/deployment/environment/devenv"

	"github.com/smartcontractkit/chainlink/deployment/ccip/changeset"

	"github.com/smartcontractkit/chainlink/deployment"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/ccip/generated/rmn_home"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/ccip/generated/rmn_remote"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/ccip/generated/router"

	testsetups "github.com/smartcontractkit/chainlink/integration-tests/testsetups/ccip"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
)

func TestRMN_TwoMessagesOnTwoLanesIncludingBatching(t *testing.T) {
	runRmnTestCase(t, rmnTestCase{
		name:        "messages on two lanes including batching",
		waitForExec: true,
		homeChainConfig: homeChainConfig{
			f: map[int]int{chain0: 1, chain1: 1},
		},
		remoteChainsConfig: []remoteChainConfig{
			{chainIdx: chain0, f: 1},
			{chainIdx: chain1, f: 1},
		},
		rmnNodes: []rmnNode{
			{id: 0, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 1, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 2, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
		},
		messagesToSend: []messageToSend{
			{fromChainIdx: chain0, toChainIdx: chain1, count: 1},
			{fromChainIdx: chain1, toChainIdx: chain0, count: 5},
		},
	})
}

func TestRMN_MultipleMessagesOnOneLaneNoWaitForExec(t *testing.T) {
	runRmnTestCase(t, rmnTestCase{
		name:        "multiple messages for rmn batching inspection and one rmn node down",
		waitForExec: false, // do not wait for execution reports
		homeChainConfig: homeChainConfig{
			f: map[int]int{chain0: 1, chain1: 1},
		},
		remoteChainsConfig: []remoteChainConfig{
			{chainIdx: chain0, f: 1},
			{chainIdx: chain1, f: 1},
		},
		rmnNodes: []rmnNode{
			{id: 0, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 1, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 2, isSigner: true, observedChainIdxs: []int{chain0, chain1}, forceExit: true}, // one rmn node is down
		},
		messagesToSend: []messageToSend{
			{fromChainIdx: chain1, toChainIdx: chain0, count: 10},
		},
	})
}

func TestRMN_NotEnoughObservers(t *testing.T) {
	runRmnTestCase(t, rmnTestCase{
		name:                "one message but not enough observers, should not get a commit report",
		passIfNoCommitAfter: 15 * time.Second,
		homeChainConfig: homeChainConfig{
			f: map[int]int{chain0: 1, chain1: 1},
		},
		remoteChainsConfig: []remoteChainConfig{
			{chainIdx: chain0, f: 1},
			{chainIdx: chain1, f: 1},
		},
		rmnNodes: []rmnNode{
			{id: 0, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 1, isSigner: true, observedChainIdxs: []int{chain0, chain1}, forceExit: true},
			{id: 2, isSigner: true, observedChainIdxs: []int{chain0, chain1}, forceExit: true},
		},
		messagesToSend: []messageToSend{
			{fromChainIdx: chain0, toChainIdx: chain1, count: 1},
		},
	})
}

func TestRMN_DifferentSigners(t *testing.T) {
	runRmnTestCase(t, rmnTestCase{
		name: "different signers and different observers",
		homeChainConfig: homeChainConfig{
			f: map[int]int{chain0: 1, chain1: 1},
		},
		remoteChainsConfig: []remoteChainConfig{
			{chainIdx: chain0, f: 1},
			{chainIdx: chain1, f: 1},
		},
		rmnNodes: []rmnNode{
			{id: 0, isSigner: false, observedChainIdxs: []int{chain0, chain1}},
			{id: 1, isSigner: false, observedChainIdxs: []int{chain0, chain1}},
			{id: 2, isSigner: false, observedChainIdxs: []int{chain0, chain1}},
			{id: 3, isSigner: true, observedChainIdxs: []int{}},
			{id: 4, isSigner: true, observedChainIdxs: []int{}},
			{id: 5, isSigner: true, observedChainIdxs: []int{}},
		},
		messagesToSend: []messageToSend{
			{fromChainIdx: chain0, toChainIdx: chain1, count: 1},
		},
	})
}

func TestRMN_NotEnoughSigners(t *testing.T) {
	runRmnTestCase(t, rmnTestCase{
		name:                "different signers and different observers",
		passIfNoCommitAfter: 15 * time.Second,
		homeChainConfig: homeChainConfig{
			f: map[int]int{chain0: 1, chain1: 1},
		},
		remoteChainsConfig: []remoteChainConfig{
			{chainIdx: chain0, f: 1},
			{chainIdx: chain1, f: 1},
		},
		rmnNodes: []rmnNode{
			{id: 0, isSigner: false, observedChainIdxs: []int{chain0, chain1}},
			{id: 1, isSigner: false, observedChainIdxs: []int{chain0, chain1}},
			{id: 2, isSigner: false, observedChainIdxs: []int{chain0, chain1}},
			{id: 3, isSigner: true, observedChainIdxs: []int{}},
			{id: 4, isSigner: true, observedChainIdxs: []int{}, forceExit: true}, // signer is down
			{id: 5, isSigner: true, observedChainIdxs: []int{}, forceExit: true}, // signer is down
		},
		messagesToSend: []messageToSend{
			{fromChainIdx: chain0, toChainIdx: chain1, count: 1},
		},
	})
}

func TestRMN_DifferentRmnNodesForDifferentChains(t *testing.T) {
	runRmnTestCase(t, rmnTestCase{
		name:        "different rmn nodes support different chains",
		waitForExec: false,
		homeChainConfig: homeChainConfig{
			f: map[int]int{chain0: 1, chain1: 1},
		},
		remoteChainsConfig: []remoteChainConfig{
			{chainIdx: chain0, f: 1},
			{chainIdx: chain1, f: 1},
		},
		rmnNodes: []rmnNode{
			{id: 0, isSigner: true, observedChainIdxs: []int{chain0}},
			{id: 1, isSigner: true, observedChainIdxs: []int{chain0}},
			{id: 2, isSigner: true, observedChainIdxs: []int{chain0}},
			{id: 3, isSigner: true, observedChainIdxs: []int{chain1}},
			{id: 4, isSigner: true, observedChainIdxs: []int{chain1}},
			{id: 5, isSigner: true, observedChainIdxs: []int{chain1}},
		},
		messagesToSend: []messageToSend{
			{fromChainIdx: chain0, toChainIdx: chain1, count: 1},
			{fromChainIdx: chain1, toChainIdx: chain0, count: 1},
		},
	})
}

func TestRMN_TwoMessagesOneSourceChainCursed(t *testing.T) {
	runRmnTestCase(t, rmnTestCase{
		name:                "two messages, one source chain is cursed",
		passIfNoCommitAfter: 15 * time.Second,
		cursedSubjectsPerChain: map[int][]int{
			chain1: {chain0},
		},
		homeChainConfig: homeChainConfig{
			f: map[int]int{chain0: 1, chain1: 1},
		},
		remoteChainsConfig: []remoteChainConfig{
			{chainIdx: chain0, f: 1},
			{chainIdx: chain1, f: 1},
		},
		rmnNodes: []rmnNode{
			{id: 0, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 1, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 2, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
		},
		messagesToSend: []messageToSend{
			{fromChainIdx: chain0, toChainIdx: chain1, count: 1}, // <----- this message should not be committed
			{fromChainIdx: chain1, toChainIdx: chain0, count: 1},
		},
	})
}

func TestRMN_GlobalCurseTwoMessagesOnTwoLanes(t *testing.T) {
	runRmnTestCase(t, rmnTestCase{
		name:        "global curse messages on two lanes",
		waitForExec: false,
		homeChainConfig: homeChainConfig{
			f: map[int]int{chain0: 1, chain1: 1},
		},
		remoteChainsConfig: []remoteChainConfig{
			{chainIdx: chain0, f: 1},
			{chainIdx: chain1, f: 1},
		},
		rmnNodes: []rmnNode{
			{id: 0, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 1, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
			{id: 2, isSigner: true, observedChainIdxs: []int{chain0, chain1}},
		},
		messagesToSend: []messageToSend{
			{fromChainIdx: chain0, toChainIdx: chain1, count: 1},
			{fromChainIdx: chain1, toChainIdx: chain0, count: 5},
		},
		cursedSubjectsPerChain: map[int][]int{
			chain1: {globalCurse},
			chain0: {globalCurse},
		},
		passIfNoCommitAfter: 15 * time.Second,
	})
}

const (
	chain0      = 0
	chain1      = 1
	globalCurse = 1000
)

func runRmnTestCase(t *testing.T, tc rmnTestCase) {
	require.NoError(t, os.Setenv("ENABLE_RMN", "true"))
	require.NoError(t, tc.validate())

	ctx := testcontext.Get(t)
	t.Logf("Running RMN test case: %s", tc.name)

	envWithRMN, rmnCluster := testsetups.NewLocalDevEnvironmentWithRMN(t, logger.TestLogger(t), len(tc.rmnNodes))
	t.Logf("envWithRmn: %#v", envWithRMN)

	tc.populateFields(t, envWithRMN, rmnCluster)

	onChainState, err := changeset.LoadOnchainState(envWithRMN.Env)
	require.NoError(t, err)
	t.Logf("onChainState: %#v", onChainState)

	homeChain, ok := envWithRMN.Env.Chains[envWithRMN.HomeChainSel]
	require.True(t, ok)

	homeChainState, ok := onChainState.Chains[envWithRMN.HomeChainSel]
	require.True(t, ok)

	allDigests, err := homeChainState.RMNHome.GetConfigDigests(&bind.CallOpts{Context: ctx})
	require.NoError(t, err)

	t.Logf("RMNHome candidateDigest before setting new candidate: %x, activeDigest: %x",
		allDigests.CandidateConfigDigest[:], allDigests.ActiveConfigDigest[:])

	staticConfig := rmn_home.RMNHomeStaticConfig{Nodes: tc.pf.rmnHomeNodes, OffchainConfig: []byte{}}
	dynamicConfig := rmn_home.RMNHomeDynamicConfig{SourceChains: tc.pf.rmnHomeSourceChains, OffchainConfig: []byte{}}
	t.Logf("Setting RMNHome candidate with staticConfig: %+v, dynamicConfig: %+v, current candidateDigest: %x",
		staticConfig, dynamicConfig, allDigests.CandidateConfigDigest[:])
	tx, err := homeChainState.RMNHome.SetCandidate(homeChain.DeployerKey, staticConfig, dynamicConfig, allDigests.CandidateConfigDigest)
	require.NoError(t, err)

	_, err = deployment.ConfirmIfNoError(homeChain, tx, err)
	require.NoError(t, err)

	candidateDigest, err := homeChainState.RMNHome.GetCandidateDigest(&bind.CallOpts{Context: ctx})
	require.NoError(t, err)

	t.Logf("RMNHome candidateDigest after setting new candidate: %x", candidateDigest[:])
	t.Logf("Promoting RMNHome candidate with candidateDigest: %x", candidateDigest[:])

	tx, err = homeChainState.RMNHome.PromoteCandidateAndRevokeActive(
		homeChain.DeployerKey, candidateDigest, allDigests.ActiveConfigDigest)
	require.NoError(t, err)

	_, err = deployment.ConfirmIfNoError(homeChain, tx, err)
	require.NoError(t, err)

	// check the active digest is the same as the candidate digest
	activeDigest, err := homeChainState.RMNHome.GetActiveDigest(&bind.CallOpts{Context: ctx})
	require.NoError(t, err)
	require.Equalf(t, candidateDigest, activeDigest,
		"active digest should be the same as the previously candidate digest after promotion, previous candidate: %x, active: %x",
		candidateDigest[:], activeDigest[:])

	tc.setRmnRemoteConfig(ctx, t, onChainState, activeDigest, envWithRMN)

	tc.killMarkedRmnNodes(t, rmnCluster)

	changeset.ReplayLogs(t, envWithRMN.Env.Offchain, envWithRMN.ReplayBlocks)
	require.NoError(t, changeset.AddLanesForAll(envWithRMN.Env, onChainState))
	disabledNodes := tc.disableOraclesIfThisIsACursingTestCase(ctx, t, envWithRMN)

	startBlocks, seqNumCommit, seqNumExec := tc.sendMessages(t, onChainState, envWithRMN)
	t.Logf("Sent all messages, seqNumCommit: %v seqNumExec: %v", seqNumCommit, seqNumExec)

	tc.callContractsToCurseChains(ctx, t, onChainState, envWithRMN)

	tc.enableOracles(ctx, t, envWithRMN, disabledNodes)

	expectedSeqNum := make(map[changeset.SourceDestPair]uint64)
	for k, v := range seqNumCommit {
		cursedSubjectsOfDest, exists := tc.pf.cursedSubjectsPerChainSel[k.DestChainSelector]
		shouldSkip := exists && (slices.Contains(cursedSubjectsOfDest, globalCurse) ||
			slices.Contains(cursedSubjectsOfDest, k.SourceChainSelector))

		if !shouldSkip {
			expectedSeqNum[k] = v
		}
	}

	t.Logf("expectedSeqNums: %v", expectedSeqNum)
	t.Logf("expectedSeqNums including cursed chains: %v", seqNumCommit)

	if len(tc.cursedSubjectsPerChain) > 0 && len(seqNumCommit) == len(expectedSeqNum) {
		t.Fatalf("test case is wrong: no message was sent to non-cursed chains when you " +
			"define curse subjects, your test case should have at least one message not expected to be delivered")
	}

	commitReportReceived := make(chan struct{})
	go func() {
		if len(expectedSeqNum) > 0 {
			changeset.ConfirmCommitForAllWithExpectedSeqNums(t, envWithRMN.Env, onChainState, expectedSeqNum, startBlocks)
			commitReportReceived <- struct{}{}
		}

		if len(seqNumCommit) > 0 && len(seqNumCommit) > len(expectedSeqNum) {
			// wait for a duration and assert that commit reports were not delivered for cursed source chains
			changeset.ConfirmCommitForAllWithExpectedSeqNums(t, envWithRMN.Env, onChainState, seqNumCommit, startBlocks)
			commitReportReceived <- struct{}{}
		}
	}()

	if tc.passIfNoCommitAfter > 0 { // wait for a duration and assert that commit reports were not delivered
		if len(expectedSeqNum) > 0 && len(seqNumCommit) > len(expectedSeqNum) {
			t.Logf("⌛ Waiting for commit reports of non-cursed chains...")
			<-commitReportReceived
			t.Logf("✅ Commit reports of non-cursed chains received")
		}

		tim := time.NewTimer(tc.passIfNoCommitAfter)
		t.Logf("waiting for %s before asserting that commit report was not received", tc.passIfNoCommitAfter)

		select {
		case <-commitReportReceived:
			t.Errorf("Commit report was received while it was not expected")
			return
		case <-tim.C:
			return
		}
	}

	t.Logf("⌛ Waiting for commit reports...")
	<-commitReportReceived // wait for commit reports
	t.Logf("✅ Commit report")

	if tc.waitForExec {
		t.Logf("⌛ Waiting for exec reports...")
		changeset.ConfirmExecWithSeqNrsForAll(t, envWithRMN.Env, onChainState, seqNumExec, startBlocks)
		t.Logf("✅ Exec report")
	}
}

func createObserverNodesBitmap(chainSel uint64, rmnNodes []rmnNode, chainSelectors []uint64) *big.Int {
	bitmap := new(big.Int)
	for _, n := range rmnNodes {
		observedChainSelectors := mapset.NewSet[uint64]()
		for _, chainIdx := range n.observedChainIdxs {
			observedChainSelectors.Add(chainSelectors[chainIdx])
		}

		if !observedChainSelectors.Contains(chainSel) {
			continue
		}

		bitmap.SetBit(bitmap, n.id, 1)
	}

	return bitmap
}

type homeChainConfig struct {
	f map[int]int
}

type remoteChainConfig struct {
	chainIdx int
	f        int
}

type rmnNode struct {
	id                int
	isSigner          bool
	observedChainIdxs []int
	forceExit         bool // force exit will simply force exit the rmn node to simulate failure scenarios
}

type messageToSend struct {
	fromChainIdx int
	toChainIdx   int
	count        int
}

type rmnTestCase struct {
	name string
	// If set to 0, the test will wait for commit reports.
	// If set to a positive value, the test will wait for that duration and will assert that commit report was not delivered.
	passIfNoCommitAfter    time.Duration
	cursedSubjectsPerChain map[int][]int
	waitForExec            bool
	homeChainConfig        homeChainConfig
	remoteChainsConfig     []remoteChainConfig
	rmnNodes               []rmnNode
	messagesToSend         []messageToSend

	// populated fields after environment setup
	pf testCasePopulatedFields
}

type testCasePopulatedFields struct {
	chainSelectors            []uint64
	rmnHomeNodes              []rmn_home.RMNHomeNode
	rmnRemoteSigners          []rmn_remote.RMNRemoteSigner
	rmnHomeSourceChains       []rmn_home.RMNHomeSourceChain
	cursedSubjectsPerChainSel map[uint64][]uint64
}

func (tc *rmnTestCase) populateFields(t *testing.T, envWithRMN changeset.DeployedEnv, rmnCluster devenv.RMNCluster) {
	require.GreaterOrEqual(t, len(envWithRMN.Env.Chains), 2, "test assumes at least two chains")
	for _, chain := range envWithRMN.Env.Chains {
		tc.pf.chainSelectors = append(tc.pf.chainSelectors, chain.Selector)
	}

	for _, rmnNodeInfo := range tc.rmnNodes {
		rmn := rmnCluster.Nodes["rmn_"+strconv.Itoa(rmnNodeInfo.id)]

		var offchainPublicKey [32]byte
		copy(offchainPublicKey[:], rmn.RMN.OffchainPublicKey)

		tc.pf.rmnHomeNodes = append(tc.pf.rmnHomeNodes, rmn_home.RMNHomeNode{
			PeerId:            rmn.Proxy.PeerID,
			OffchainPublicKey: offchainPublicKey,
		})

		if rmnNodeInfo.isSigner {
			if rmnNodeInfo.id < 0 {
				t.Fatalf("node id is negative: %d", rmnNodeInfo.id)
			}
			tc.pf.rmnRemoteSigners = append(tc.pf.rmnRemoteSigners, rmn_remote.RMNRemoteSigner{
				OnchainPublicKey: rmn.RMN.EVMOnchainPublicKey,
				NodeIndex:        uint64(rmnNodeInfo.id),
			})
		}
	}

	for remoteChainIdx, remoteF := range tc.homeChainConfig.f {
		if remoteF < 0 {
			t.Fatalf("negative remote F: %d", remoteF)
		}
		// configure remote chain details on the home contract
		tc.pf.rmnHomeSourceChains = append(tc.pf.rmnHomeSourceChains, rmn_home.RMNHomeSourceChain{
			ChainSelector:       tc.pf.chainSelectors[remoteChainIdx],
			F:                   uint64(remoteF),
			ObserverNodesBitmap: createObserverNodesBitmap(tc.pf.chainSelectors[remoteChainIdx], tc.rmnNodes, tc.pf.chainSelectors),
		})
	}

	// populate cursed subjects with actual chain selectors
	tc.pf.cursedSubjectsPerChainSel = make(map[uint64][]uint64)
	for chainIdx, subjects := range tc.cursedSubjectsPerChain {
		chainSel := tc.pf.chainSelectors[chainIdx]
		for _, subject := range subjects {
			subjSel := uint64(globalCurse)
			if subject != globalCurse {
				subjSel = tc.pf.chainSelectors[subject]
			}
			tc.pf.cursedSubjectsPerChainSel[chainSel] = append(tc.pf.cursedSubjectsPerChainSel[chainSel], subjSel)
		}
	}
}

func (tc rmnTestCase) validate() error {
	if len(tc.cursedSubjectsPerChain) > 0 && tc.passIfNoCommitAfter == 0 {
		return errors.New("when you define cursed subjects you also need to define the duration that the " +
			"test will wait for non-transmitted roots")
	}
	return nil
}

func (tc rmnTestCase) setRmnRemoteConfig(
	ctx context.Context,
	t *testing.T,
	onChainState changeset.CCIPOnChainState,
	activeDigest [32]byte,
	envWithRMN changeset.DeployedEnv) {
	for _, remoteCfg := range tc.remoteChainsConfig {
		remoteSel := tc.pf.chainSelectors[remoteCfg.chainIdx]
		chState, ok := onChainState.Chains[remoteSel]
		require.True(t, ok)
		if remoteCfg.f < 0 {
			t.Fatalf("negative F: %d", remoteCfg.f)
		}
		rmnRemoteConfig := rmn_remote.RMNRemoteConfig{
			RmnHomeContractConfigDigest: activeDigest,
			Signers:                     tc.pf.rmnRemoteSigners,
			F:                           uint64(remoteCfg.f),
		}

		chain := envWithRMN.Env.Chains[tc.pf.chainSelectors[remoteCfg.chainIdx]]

		t.Logf("Setting RMNRemote config with RMNHome active digest: %x, cfg: %+v", activeDigest[:], rmnRemoteConfig)
		tx2, err2 := chState.RMNRemote.SetConfig(chain.DeployerKey, rmnRemoteConfig)
		require.NoError(t, err2)
		_, err2 = deployment.ConfirmIfNoError(chain, tx2, err2)
		require.NoError(t, err2)

		// confirm the config is set correctly
		config, err2 := chState.RMNRemote.GetVersionedConfig(&bind.CallOpts{Context: ctx})
		require.NoError(t, err2)
		require.Equalf(t,
			activeDigest,
			config.Config.RmnHomeContractConfigDigest,
			"RMNRemote config digest should be the same as the active digest of RMNHome after setting, RMNHome active: %x, RMNRemote config: %x",
			activeDigest[:], config.Config.RmnHomeContractConfigDigest[:])

		t.Logf("RMNRemote config digest after setting: %x", config.Config.RmnHomeContractConfigDigest[:])
	}
}

func (tc rmnTestCase) killMarkedRmnNodes(t *testing.T, rmnCluster devenv.RMNCluster) {
	for _, n := range tc.rmnNodes {
		if n.forceExit {
			t.Logf("Pausing RMN node %d", n.id)
			rmnN := rmnCluster.Nodes["rmn_"+strconv.Itoa(n.id)]
			require.NoError(t, osutil.ExecCmd(zerolog.Nop(), "docker kill "+rmnN.Proxy.ContainerName))
			t.Logf("Paused RMN node %d", n.id)
		}
	}
}

func (tc rmnTestCase) disableOraclesIfThisIsACursingTestCase(ctx context.Context, t *testing.T, envWithRMN changeset.DeployedEnv) []string {
	disabledNodes := make([]string, 0)

	if len(tc.cursedSubjectsPerChain) > 0 {
		listNodesResp, err := envWithRMN.Env.Offchain.ListNodes(ctx, &node.ListNodesRequest{})
		require.NoError(t, err)

		for _, n := range listNodesResp.Nodes {
			if strings.HasPrefix(n.Name, "bootstrap") {
				continue
			}
			_, err := envWithRMN.Env.Offchain.DisableNode(ctx, &node.DisableNodeRequest{Id: n.Id})
			require.NoError(t, err)
			disabledNodes = append(disabledNodes, n.Id)
			t.Logf("node %s disabled", n.Id)
		}
	}

	return disabledNodes
}

func (tc rmnTestCase) sendMessages(t *testing.T, onChainState changeset.CCIPOnChainState, envWithRMN changeset.DeployedEnv) (map[uint64]*uint64, map[changeset.SourceDestPair]uint64, map[changeset.SourceDestPair][]uint64) {
	startBlocks := make(map[uint64]*uint64)
	seqNumCommit := make(map[changeset.SourceDestPair]uint64)
	seqNumExec := make(map[changeset.SourceDestPair][]uint64)

	for _, msg := range tc.messagesToSend {
		fromChain := tc.pf.chainSelectors[msg.fromChainIdx]
		toChain := tc.pf.chainSelectors[msg.toChainIdx]

		for i := 0; i < msg.count; i++ {
			msgSentEvent := changeset.TestSendRequest(t, envWithRMN.Env, onChainState, fromChain, toChain, false, router.ClientEVM2AnyMessage{
				Receiver:     common.LeftPadBytes(onChainState.Chains[toChain].Receiver.Address().Bytes(), 32),
				Data:         []byte("hello world"),
				TokenAmounts: nil,
				FeeToken:     common.HexToAddress("0x0"),
				ExtraArgs:    nil,
			})
			seqNumCommit[changeset.SourceDestPair{
				SourceChainSelector: fromChain,
				DestChainSelector:   toChain,
			}] = msgSentEvent.SequenceNumber
			seqNumExec[changeset.SourceDestPair{
				SourceChainSelector: fromChain,
				DestChainSelector:   toChain,
			}] = []uint64{msgSentEvent.SequenceNumber}
			t.Logf("Sent message from chain %d to chain %d with seqNum %d", fromChain, toChain, msgSentEvent.SequenceNumber)
		}

		zero := uint64(0)
		startBlocks[toChain] = &zero
	}

	return startBlocks, seqNumCommit, seqNumExec
}

func (tc rmnTestCase) callContractsToCurseChains(ctx context.Context, t *testing.T, onChainState changeset.CCIPOnChainState, envWithRMN changeset.DeployedEnv) {
	for _, remoteCfg := range tc.remoteChainsConfig {
		remoteSel := tc.pf.chainSelectors[remoteCfg.chainIdx]
		chState, ok := onChainState.Chains[remoteSel]
		require.True(t, ok)
		chain, ok := envWithRMN.Env.Chains[remoteSel]
		require.True(t, ok)

		cursedSubjects, ok := tc.cursedSubjectsPerChain[remoteCfg.chainIdx]
		if !ok {
			continue // nothing to curse on this chain
		}

		for _, subjectDescription := range cursedSubjects {
			subj := types.GlobalCurseSubject
			if subjectDescription != globalCurse {
				subj = chainSelectorToBytes16(tc.pf.chainSelectors[subjectDescription])
			}
			t.Logf("cursing subject %d (%d)", subj, subjectDescription)
			txCurse, errCurse := chState.RMNRemote.Curse(chain.DeployerKey, subj)
			_, errConfirm := deployment.ConfirmIfNoError(chain, txCurse, errCurse)
			require.NoError(t, errConfirm)
		}

		cs, err := chState.RMNRemote.GetCursedSubjects(&bind.CallOpts{Context: ctx})
		require.NoError(t, err)
		t.Logf("Cursed subjects: %v", cs)
	}
}

func (tc rmnTestCase) enableOracles(ctx context.Context, t *testing.T, envWithRMN changeset.DeployedEnv, nodeIDs []string) {
	for _, n := range nodeIDs {
		_, err := envWithRMN.Env.Offchain.EnableNode(ctx, &node.EnableNodeRequest{Id: n})
		require.NoError(t, err)
		t.Logf("node %s enabled", n)
	}
}

func chainSelectorToBytes16(chainSel uint64) [16]byte {
	var result [16]byte
	// Convert the uint64 to bytes and place it in the last 8 bytes of the array
	binary.BigEndian.PutUint64(result[8:], chainSel)
	return result
}
