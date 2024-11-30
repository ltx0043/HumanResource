package keystone

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink/deployment"

	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/keystone/generated/capabilities_registry"
	kcr "github.com/smartcontractkit/chainlink/v2/core/gethwrappers/keystone/generated/capabilities_registry"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/p2pkey"
)

var (
	CapabilitiesRegistry deployment.ContractType = "CapabilitiesRegistry" // https://github.com/smartcontractkit/chainlink/blob/50c1b3dbf31bd145b312739b08967600a5c67f30/contracts/src/v0.8/keystone/CapabilitiesRegistry.sol#L392
	KeystoneForwarder    deployment.ContractType = "KeystoneForwarder"    // https://github.com/smartcontractkit/chainlink/blob/50c1b3dbf31bd145b312739b08967600a5c67f30/contracts/src/v0.8/keystone/KeystoneForwarder.sol#L90
	OCR3Capability       deployment.ContractType = "OCR3Capability"       // https://github.com/smartcontractkit/chainlink/blob/50c1b3dbf31bd145b312739b08967600a5c67f30/contracts/src/v0.8/keystone/OCR3Capability.sol#L12
	FeedConsumer         deployment.ContractType = "FeedConsumer"         // no type and a version in contract https://github.com/smartcontractkit/chainlink/blob/89183a8a5d22b1aeca0ade3b76d16aa84067aa57/contracts/src/v0.8/keystone/KeystoneFeedsConsumer.sol#L1
)

type DeployResponse struct {
	Address common.Address
	Tx      common.Hash // todo: chain agnostic
	Tv      deployment.TypeAndVersion
}

type DeployRequest struct {
	Chain deployment.Chain
}

type DonNode struct {
	Don  string
	Node string // not unique across environments
}

type CapabilityHost struct {
	NodeID       string // globally unique
	Capabilities []capabilities_registry.CapabilitiesRegistryCapability
}

type Nop struct {
	capabilities_registry.CapabilitiesRegistryNodeOperator
	NodeIDs []string // nodes run by this operator
}

func toNodeKeys(o *deployment.Node, registryChainSel uint64) NodeKeys {
	var aptosOcr2KeyBundleId string
	var aptosOnchainPublicKey string
	var aptosCC *deployment.OCRConfig
	for details, cfg := range o.SelToOCRConfig {
		if family, err := chainsel.GetSelectorFamily(details.ChainSelector); err == nil && family == chainsel.FamilyAptos {
			aptosCC = &cfg
			break
		}
	}
	if aptosCC != nil {
		aptosOcr2KeyBundleId = aptosCC.KeyBundleID
		aptosOnchainPublicKey = fmt.Sprintf("%x", aptosCC.OnchainPublicKey[:])
	}
	registryChainID, err := chainsel.ChainIdFromSelector(registryChainSel)
	if err != nil {
		panic(err)
	}
	registryChainDetails, err := chainsel.GetChainDetailsByChainIDAndFamily(strconv.Itoa(int(registryChainID)), chainsel.FamilyEVM)
	if err != nil {
		panic(err)
	}
	evmCC := o.SelToOCRConfig[registryChainDetails]
	return NodeKeys{
		EthAddress:            string(evmCC.TransmitAccount),
		P2PPeerID:             strings.TrimPrefix(o.PeerID.String(), "p2p_"),
		OCR2BundleID:          evmCC.KeyBundleID,
		OCR2OffchainPublicKey: fmt.Sprintf("%x", evmCC.OffchainPublicKey[:]),
		OCR2OnchainPublicKey:  fmt.Sprintf("%x", evmCC.OnchainPublicKey[:]),
		OCR2ConfigPublicKey:   fmt.Sprintf("%x", evmCC.ConfigEncryptionPublicKey[:]),
		CSAPublicKey:          o.CSAKey,
		// default value of encryption public key is the CSA public key
		// TODO: DEVSVCS-760
		EncryptionPublicKey: strings.TrimPrefix(o.CSAKey, "csa_"),
		// TODO Aptos support. How will that be modeled in clo data?
		// TODO: AptosAccount is unset but probably unused
		AptosBundleID:         aptosOcr2KeyBundleId,
		AptosOnchainPublicKey: aptosOnchainPublicKey,
	}
}
func makeNodeKeysSlice(nodes []deployment.Node, registryChainSel uint64) []NodeKeys {
	var out []NodeKeys
	for _, n := range nodes {
		out = append(out, toNodeKeys(&n, registryChainSel))
	}
	return out
}

type NOP struct {
	Name  string
	Nodes []string // peerID
}

func (v NOP) Validate() error {
	if v.Name == "" {
		return errors.New("name is empty")
	}
	if len(v.Nodes) == 0 {
		return errors.New("no nodes")
	}
	for i, n := range v.Nodes {
		_, err := p2pkey.MakePeerID(n)
		if err != nil {
			return fmt.Errorf("failed to nop %s: node %d is not valid peer id %s: %w", v.Name, i, n, err)
		}
	}

	return nil
}

// DonCapabilities is a set of capabilities hosted by a set of node operators
// in is in a convenient form to handle the CLO representation of the nop data
type DonCapabilities struct {
	Name         string
	Nops         []NOP
	Capabilities []kcr.CapabilitiesRegistryCapability // every capability is hosted on each nop
}

func (v DonCapabilities) Validate() error {
	if v.Name == "" {
		return errors.New("name is empty")
	}
	if len(v.Nops) == 0 {
		return errors.New("no nops")
	}
	for i, n := range v.Nops {
		if err := n.Validate(); err != nil {
			return fmt.Errorf("failed to validate nop %d '%s': %w", i, n.Name, err)
		}
	}
	if len(v.Capabilities) == 0 {
		return errors.New("no capabilities")
	}
	return nil
}

func NodeOperator(name string, adminAddress string) capabilities_registry.CapabilitiesRegistryNodeOperator {
	return capabilities_registry.CapabilitiesRegistryNodeOperator{
		Name:  name,
		Admin: adminAddr(adminAddress),
	}
}

func nopsToNodes(donInfos []DonInfo, dons []DonCapabilities, chainSelector uint64) (map[capabilities_registry.CapabilitiesRegistryNodeOperator][]string, error) {
	out := make(map[capabilities_registry.CapabilitiesRegistryNodeOperator][]string)
	for _, don := range dons {
		for _, nop := range don.Nops {
			idx := slices.IndexFunc(donInfos, func(donInfo DonInfo) bool {
				return donInfo.Name == don.Name
			})
			if idx < 0 {
				return nil, fmt.Errorf("couldn't find donInfo for %v", don.Name)
			}
			donInfo := donInfos[idx]
			idx = slices.IndexFunc(donInfo.Nodes, func(node deployment.Node) bool {
				return node.PeerID.String() == nop.Nodes[0]
			})
			if idx < 0 {
				return nil, fmt.Errorf("couldn't find node with p2p_id '%v'", nop.Nodes[0])
			}
			node := donInfo.Nodes[idx]
			a := node.AdminAddr
			nodeOperator := NodeOperator(nop.Name, a)
			for _, node := range nop.Nodes {
				idx = slices.IndexFunc(donInfo.Nodes, func(n deployment.Node) bool {
					return n.PeerID.String() == node
				})
				if idx < 0 {
					return nil, fmt.Errorf("couldn't find node with p2p_id '%v'", node)
				}
				out[nodeOperator] = append(out[nodeOperator], donInfo.Nodes[idx].NodeID)
			}
		}
	}

	return out, nil
}

// mapDonsToCaps converts a list of DonCapabilities to a map of don name to capabilities
func mapDonsToCaps(dons []DonInfo) map[string][]kcr.CapabilitiesRegistryCapability {
	out := make(map[string][]kcr.CapabilitiesRegistryCapability)
	for _, don := range dons {
		out[don.Name] = don.Capabilities
	}
	return out
}

// mapDonsToNodes returns a map of don name to simplified representation of their nodes
// all nodes must have evm config and ocr3 capability nodes are must also have an aptos chain config
func mapDonsToNodes(dons []DonInfo, excludeBootstraps bool, registryChainSel uint64) (map[string][]deployment.Node, error) {
	donToNodes := make(map[string][]deployment.Node)
	// get the nodes for each don from the offchain client, get ocr2 config from one of the chain configs for the node b/c
	// they are equivalent

	for _, don := range dons {
		for _, node := range don.Nodes {
			if excludeBootstraps && node.IsBootstrap {
				continue
			}
			if _, ok := donToNodes[don.Name]; !ok {
				donToNodes[don.Name] = make([]deployment.Node, 0)
			}
			donToNodes[don.Name] = append(donToNodes[don.Name], node)
		}
	}

	return donToNodes, nil
}

// RegisteredDon is a representation of a don that exists in the in the capabilities registry all with the enriched node data
type RegisteredDon struct {
	Name  string
	Info  capabilities_registry.CapabilitiesRegistryDONInfo
	Nodes []deployment.Node
}

func (d RegisteredDon) signers(chainFamily string) []common.Address {
	sort.Slice(d.Nodes, func(i, j int) bool {
		return d.Nodes[i].PeerID.String() < d.Nodes[j].PeerID.String()
	})
	var out []common.Address
	for _, n := range d.Nodes {
		if n.IsBootstrap {
			continue
		}
		var found bool
		var registryChainDetails chainsel.ChainDetails
		for details, _ := range n.SelToOCRConfig {
			if family, err := chainsel.GetSelectorFamily(details.ChainSelector); err == nil && family == chainFamily {
				found = true
				registryChainDetails = details

			}
		}
		if !found {
			panic(fmt.Sprintf("chainType not found: %v", chainFamily))
		}
		// eth address is the first 20 bytes of the Signer
		config, exists := n.SelToOCRConfig[registryChainDetails]
		if !exists {
			panic(fmt.Sprintf("chainID not found: %v", registryChainDetails))
		}
		signer := config.OnchainPublicKey
		signerAddress := common.BytesToAddress(signer)
		out = append(out, signerAddress)
	}
	return out
}

func joinInfoAndNodes(donInfos map[string]kcr.CapabilitiesRegistryDONInfo, dons []DonInfo, registryChainSel uint64) ([]RegisteredDon, error) {
	// all maps should have the same keys
	nodes, err := mapDonsToNodes(dons, true, registryChainSel)
	if err != nil {
		return nil, fmt.Errorf("failed to map dons to capabilities: %w", err)
	}
	if len(donInfos) != len(nodes) {
		return nil, fmt.Errorf("mismatched lengths don infos %d,  nodes %d", len(donInfos), len(nodes))
	}
	var out []RegisteredDon
	for donName, info := range donInfos {

		ocr2nodes, ok := nodes[donName]
		if !ok {
			return nil, fmt.Errorf("nodes not found for don %s", donName)
		}
		out = append(out, RegisteredDon{
			Name:  donName,
			Info:  info,
			Nodes: ocr2nodes,
		})
	}

	return out, nil
}

var emptyAddr = "0x0000000000000000000000000000000000000000"

// compute the admin address from the string. If the address is empty, replaces the 0s with fs
// contract registry disallows 0x0 as an admin address, but our test net nops use it
func adminAddr(addr string) common.Address {
	needsFixing := addr == emptyAddr
	addr = strings.TrimPrefix(addr, "0x")
	if needsFixing {
		addr = strings.ReplaceAll(addr, "0", "f")
	}
	return common.HexToAddress(strings.TrimPrefix(addr, "0x"))
}