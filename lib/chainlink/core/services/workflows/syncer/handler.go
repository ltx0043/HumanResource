package syncer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/smartcontractkit/chainlink-common/pkg/custmsg"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/secrets"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/wasm/host"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/smartcontractkit/chainlink/v2/core/platform"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/workflowkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/workflows"
	"github.com/smartcontractkit/chainlink/v2/core/services/workflows/store"
)

var ErrNotImplemented = errors.New("not implemented")

// WorkflowRegistryrEventType is the type of event that is emitted by the WorkflowRegistry
type WorkflowRegistryEventType string

var (
	// ForceUpdateSecretsEvent is emitted when a request to force update a workflows secrets is made
	ForceUpdateSecretsEvent WorkflowRegistryEventType = "WorkflowForceUpdateSecretsRequestedV1"

	// WorkflowRegisteredEvent is emitted when a workflow is registered
	WorkflowRegisteredEvent WorkflowRegistryEventType = "WorkflowRegisteredV1"

	// WorkflowUpdatedEvent is emitted when a workflow is updated
	WorkflowUpdatedEvent WorkflowRegistryEventType = "WorkflowUpdatedV1"

	// WorkflowPausedEvent is emitted when a workflow is paused
	WorkflowPausedEvent WorkflowRegistryEventType = "WorkflowPausedV1"

	// WorkflowActivatedEvent is emitted when a workflow is activated
	WorkflowActivatedEvent WorkflowRegistryEventType = "WorkflowActivatedV1"

	// WorkflowDeletedEvent is emitted when a workflow is deleted
	WorkflowDeletedEvent WorkflowRegistryEventType = "WorkflowDeletedV1"
)

// WorkflowRegistryForceUpdateSecretsRequestedV1 is a chain agnostic definition of the WorkflowRegistry
// ForceUpdateSecretsRequested event.
type WorkflowRegistryForceUpdateSecretsRequestedV1 struct {
	SecretsURLHash []byte
	Owner          []byte
	WorkflowName   string
}

type WorkflowRegistryWorkflowRegisteredV1 struct {
	WorkflowID   [32]byte
	Owner        []byte
	DonID        uint32
	Status       uint8
	WorkflowName string
	BinaryURL    string
	ConfigURL    string
	SecretsURL   string
}

type WorkflowRegistryWorkflowUpdatedV1 struct {
	OldWorkflowID [32]byte
	WorkflowOwner []byte
	DonID         uint32
	NewWorkflowID [32]byte
	WorkflowName  string
	BinaryURL     string
	ConfigURL     string
	SecretsURL    string
}

type WorkflowRegistryWorkflowPausedV1 struct {
	WorkflowID    [32]byte
	WorkflowOwner []byte
	DonID         uint32
	WorkflowName  string
}

type WorkflowRegistryWorkflowActivatedV1 struct {
	WorkflowID    [32]byte
	WorkflowOwner []byte
	DonID         uint32
	WorkflowName  string
}

type WorkflowRegistryWorkflowDeletedV1 struct {
	WorkflowID    [32]byte
	WorkflowOwner []byte
	DonID         uint32
	WorkflowName  string
}

type lastFetchedAtMap struct {
	m map[string]time.Time
	sync.RWMutex
}

func (l *lastFetchedAtMap) Set(url string, at time.Time) {
	l.Lock()
	defer l.Unlock()
	l.m[url] = at
}

func (l *lastFetchedAtMap) Get(url string) (time.Time, bool) {
	l.RLock()
	defer l.RUnlock()
	got, ok := l.m[url]
	return got, ok
}

func newLastFetchedAtMap() *lastFetchedAtMap {
	return &lastFetchedAtMap{
		m: map[string]time.Time{},
	}
}

// eventHandler is a handler for WorkflowRegistryEvent events.  Each event type has a corresponding
// method that handles the event.
type eventHandler struct {
	lggr                     logger.Logger
	orm                      WorkflowRegistryDS
	fetcher                  FetcherFunc
	workflowStore            store.Store
	capRegistry              core.CapabilitiesRegistry
	engineRegistry           *engineRegistry
	emitter                  custmsg.MessageEmitter
	lastFetchedAtMap         *lastFetchedAtMap
	clock                    clockwork.Clock
	secretsFreshnessDuration time.Duration
	encryptionKey            workflowkey.Key
}

type Event interface {
	GetEventType() WorkflowRegistryEventType
	GetData() any
}

var defaultSecretsFreshnessDuration = 24 * time.Hour

// NewEventHandler returns a new eventHandler instance.
func NewEventHandler(
	lggr logger.Logger,
	orm ORM,
	gateway FetcherFunc,
	workflowStore store.Store,
	capRegistry core.CapabilitiesRegistry,
	emitter custmsg.MessageEmitter,
	clock clockwork.Clock,
	encryptionKey workflowkey.Key,
) *eventHandler {
	return &eventHandler{
		lggr:                     lggr,
		orm:                      orm,
		fetcher:                  gateway,
		workflowStore:            workflowStore,
		capRegistry:              capRegistry,
		engineRegistry:           newEngineRegistry(),
		emitter:                  emitter,
		lastFetchedAtMap:         newLastFetchedAtMap(),
		clock:                    clock,
		secretsFreshnessDuration: defaultSecretsFreshnessDuration,
		encryptionKey:            encryptionKey,
	}
}

func (h *eventHandler) refreshSecrets(ctx context.Context, workflowOwner, workflowName, workflowID, secretsURLHash string) (string, error) {
	owner, err := hex.DecodeString(workflowOwner)
	if err != nil {
		return "", err
	}

	decodedHash, err := hex.DecodeString(secretsURLHash)
	if err != nil {
		return "", err
	}

	updatedSecrets, err := h.forceUpdateSecretsEvent(
		ctx,
		WorkflowRegistryForceUpdateSecretsRequestedV1{
			SecretsURLHash: decodedHash,
			Owner:          owner,
			WorkflowName:   name,
		},
	)
	if err != nil {
		return "", err
	}

	return updatedSecrets, nil
}

func (h *eventHandler) SecretsFor(ctx context.Context, workflowOwner, workflowName, workflowID string) (map[string]string, error) {
	secretsURLHash, secretsPayload, err := h.orm.GetContentsByWorkflowID(ctx, workflowID)
	if err != nil {
		// The workflow record was found, but secrets_id was empty.
		// Let's just stub out the response.
		if errors.Is(err, ErrEmptySecrets) {
			return map[string]string{}, nil
		}

		return nil, fmt.Errorf("failed to fetch secrets by workflow ID: %w", err)
	}

	lastFetchedAt, ok := h.lastFetchedAtMap.Get(secretsURLHash)
	if !ok || h.clock.Now().Sub(lastFetchedAt) > h.secretsFreshnessDuration {
		updatedSecrets, innerErr := h.refreshSecrets(ctx, workflowOwner, workflowName, workflowID, secretsURLHash)
		if innerErr != nil {
			msg := fmt.Sprintf("could not refresh secrets: proceeding with stale secrets for workflowID %s: %s", workflowID, innerErr)
			h.lggr.Error(msg)
			logCustMsg(
				ctx,
				h.emitter.With(
					platform.KeyWorkflowID, workflowID,
					platform.KeyWorkflowName, workflowName,
					platform.KeyWorkflowOwner, workflowOwner,
				),
				msg,
				h.lggr,
			)
		} else {
			secretsPayload = updatedSecrets
		}
	}

	res := secrets.EncryptedSecretsResult{}
	err = json.Unmarshal([]byte(secretsPayload), &res)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal secrets: %w", err)
	}

	return secrets.DecryptSecretsForNode(
		res,
		h.encryptionKey,
		workflowOwner,
	)
}

func (h *eventHandler) Handle(ctx context.Context, event Event) error {
	switch event.GetEventType() {
	case ForceUpdateSecretsEvent:
		payload, ok := event.GetData().(WorkflowRegistryForceUpdateSecretsRequestedV1)
		if !ok {
			return newHandlerTypeError(event.GetData())
		}

		cma := h.emitter.With(
			platform.KeyWorkflowName, payload.WorkflowName,
			platform.KeyWorkflowOwner, hex.EncodeToString(payload.Owner),
		)

		if _, err := h.forceUpdateSecretsEvent(ctx, payload); err != nil {
			logCustMsg(ctx, cma, fmt.Sprintf("failed to handle force update secrets event: %v", err), h.lggr)
			return err
		}

		return nil
	case WorkflowRegisteredEvent:
		payload, ok := event.GetData().(WorkflowRegistryWorkflowRegisteredV1)
		if !ok {
			return newHandlerTypeError(event.GetData())
		}
		wfID := hex.EncodeToString(payload.WorkflowID[:])

		cma := h.emitter.With(
			platform.KeyWorkflowID, wfID,
			platform.KeyWorkflowName, payload.WorkflowName,
			platform.KeyWorkflowOwner, hex.EncodeToString(payload.Owner),
		)

		if err := h.workflowRegisteredEvent(ctx, payload); err != nil {
			logCustMsg(ctx, cma, fmt.Sprintf("failed to handle workflow registered event: %v", err), h.lggr)
			return err
		}

		h.lggr.Debugf("workflow 0x%x registered and started", wfID)
		return nil
	case WorkflowUpdatedEvent:
		payload, ok := event.GetData().(WorkflowRegistryWorkflowUpdatedV1)
		if !ok {
			return fmt.Errorf("invalid data type %T for event", event.GetData())
		}

		newWorkflowID := hex.EncodeToString(payload.NewWorkflowID[:])
		cma := h.emitter.With(
			platform.KeyWorkflowID, newWorkflowID,
			platform.KeyWorkflowName, payload.WorkflowName,
			platform.KeyWorkflowOwner, hex.EncodeToString(payload.WorkflowOwner),
		)

		if err := h.workflowUpdatedEvent(ctx, payload); err != nil {
			logCustMsg(ctx, cma, fmt.Sprintf("failed to handle workflow updated event: %v", err), h.lggr)
			return err
		}

		return nil
	case WorkflowPausedEvent:
		payload, ok := event.GetData().(WorkflowRegistryWorkflowPausedV1)
		if !ok {
			return fmt.Errorf("invalid data type %T for event", event.GetData())
		}

		wfID := hex.EncodeToString(payload.WorkflowID[:])

		cma := h.emitter.With(
			platform.KeyWorkflowID, wfID,
			platform.KeyWorkflowName, payload.WorkflowName,
			platform.KeyWorkflowOwner, hex.EncodeToString(payload.WorkflowOwner),
		)

		if err := h.workflowPausedEvent(ctx, payload); err != nil {
			logCustMsg(ctx, cma, fmt.Sprintf("failed to handle workflow paused event: %v", err), h.lggr)
			return err
		}
		return nil
	case WorkflowActivatedEvent:
		payload, ok := event.GetData().(WorkflowRegistryWorkflowActivatedV1)
		if !ok {
			return fmt.Errorf("invalid data type %T for event", event.GetData())
		}

		wfID := hex.EncodeToString(payload.WorkflowID[:])

		cma := h.emitter.With(
			platform.KeyWorkflowID, wfID,
			platform.KeyWorkflowName, payload.WorkflowName,
			platform.KeyWorkflowOwner, hex.EncodeToString(payload.WorkflowOwner),
		)
		if err := h.workflowActivatedEvent(ctx, payload); err != nil {
			logCustMsg(ctx, cma, fmt.Sprintf("failed to handle workflow activated event: %v", err), h.lggr)
			return err
		}

		return nil
	case WorkflowDeletedEvent:
		payload, ok := event.GetData().(WorkflowRegistryWorkflowDeletedV1)
		if !ok {
			return fmt.Errorf("invalid data type %T for event", event.GetData())
		}

		wfID := hex.EncodeToString(payload.WorkflowID[:])

		cma := h.emitter.With(
			platform.KeyWorkflowID, wfID,
			platform.KeyWorkflowName, payload.WorkflowName,
			platform.KeyWorkflowOwner, hex.EncodeToString(payload.WorkflowOwner),
		)

		if err := h.workflowDeletedEvent(ctx, payload); err != nil {
			logCustMsg(ctx, cma, fmt.Sprintf("failed to handle workflow deleted event: %v", err), h.lggr)
			return err
		}

		return nil
	default:
		return fmt.Errorf("event type unsupported: %v", event.GetEventType())
	}
}

// workflowRegisteredEvent handles the WorkflowRegisteredEvent event type.
func (h *eventHandler) workflowRegisteredEvent(
	ctx context.Context,
	payload WorkflowRegistryWorkflowRegisteredV1,
) error {
	wfID := hex.EncodeToString(payload.WorkflowID[:])

	// Download the contents of binaryURL, configURL and secretsURL and cache them locally.
	binary, err := h.fetcher(ctx, payload.BinaryURL)
	if err != nil {
		return fmt.Errorf("failed to fetch binary from %s : %w", payload.BinaryURL, err)
	}

	config, err := h.fetcher(ctx, payload.ConfigURL)
	if err != nil {
		return fmt.Errorf("failed to fetch config from %s : %w", payload.ConfigURL, err)
	}

	secrets, err := h.fetcher(ctx, payload.SecretsURL)
	if err != nil {
		return fmt.Errorf("failed to fetch secrets from %s : %w", payload.SecretsURL, err)
	}

	// Calculate the hash of the binary and config files
	hash := workflowID(binary, config, []byte(payload.SecretsURL))

	// Pre-check: verify that the workflowID matches; if it doesn’t abort and log an error via Beholder.
	if hash != wfID {
		return fmt.Errorf("workflowID mismatch: %s != %s", hash, wfID)
	}

	// Save the workflow secrets
	urlHash, err := h.orm.GetSecretsURLHash(payload.Owner, []byte(payload.SecretsURL))
	if err != nil {
		return fmt.Errorf("failed to get secrets URL hash: %w", err)
	}

	// Create a new entry in the workflow_spec table corresponding for the new workflow, with the contents of the binaryURL + configURL in the table
	status := job.WorkflowSpecStatusActive
	if payload.Status == 1 {
		status = job.WorkflowSpecStatusPaused
	}

	entry := &job.WorkflowSpec{
		Workflow:      hex.EncodeToString(binary),
		Config:        string(config),
		WorkflowID:    wfID,
		Status:        status,
		WorkflowOwner: hex.EncodeToString(payload.Owner),
		WorkflowName:  payload.WorkflowName,
		SpecType:      job.WASMFile,
		BinaryURL:     payload.BinaryURL,
		ConfigURL:     payload.ConfigURL,
	}
	if _, err = h.orm.UpsertWorkflowSpecWithSecrets(ctx, entry, payload.SecretsURL, hex.EncodeToString(urlHash), string(secrets)); err != nil {
		return fmt.Errorf("failed to upsert workflow spec with secrets: %w", err)
	}

	if status != job.WorkflowSpecStatusActive {
		return nil
	}

	// If status == active, start a new WorkflowEngine instance, and add it to local engine registry
	moduleConfig := &host.ModuleConfig{Logger: h.lggr, Labeler: h.emitter}
	sdkSpec, err := host.GetWorkflowSpec(ctx, moduleConfig, binary, config)
	if err != nil {
		return fmt.Errorf("failed to get workflow sdk spec: %w", err)
	}

	cfg := workflows.Config{
		Lggr:           h.lggr,
		Workflow:       *sdkSpec,
		WorkflowID:     wfID,
		WorkflowOwner:  string(payload.Owner), // this gets hex encoded in the engine.
		WorkflowName:   payload.WorkflowName,
		Registry:       h.capRegistry,
		Store:          h.workflowStore,
		Config:         config,
		Binary:         binary,
		SecretsFetcher: h,
	}
	e, err := workflows.NewEngine(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create workflow engine: %w", err)
	}

	if err := e.Start(ctx); err != nil {
		return fmt.Errorf("failed to start workflow engine: %w", err)
	}

	h.engineRegistry.Add(wfID, e)

	return nil
}

// workflowUpdatedEvent handles the WorkflowUpdatedEvent event type by first finding the
// current workflow engine, stopping it, and then starting a new workflow engine with the
// updated workflow spec.
func (h *eventHandler) workflowUpdatedEvent(
	ctx context.Context,
	payload WorkflowRegistryWorkflowUpdatedV1,
) error {
	// Remove the old workflow engine from the local registry if it exists
	if err := h.tryEngineCleanup(hex.EncodeToString(payload.OldWorkflowID[:])); err != nil {
		return err
	}

	registeredEvent := WorkflowRegistryWorkflowRegisteredV1{
		WorkflowID:   payload.NewWorkflowID,
		Owner:        payload.WorkflowOwner,
		DonID:        payload.DonID,
		Status:       0,
		WorkflowName: payload.WorkflowName,
		BinaryURL:    payload.BinaryURL,
		ConfigURL:    payload.ConfigURL,
		SecretsURL:   payload.SecretsURL,
	}

	return h.workflowRegisteredEvent(ctx, registeredEvent)
}

// workflowPausedEvent handles the WorkflowPausedEvent event type.
func (h *eventHandler) workflowPausedEvent(
	ctx context.Context,
	payload WorkflowRegistryWorkflowPausedV1,
) error {
	// Remove the workflow engine from the local registry if it exists
	if err := h.tryEngineCleanup(hex.EncodeToString(payload.WorkflowID[:])); err != nil {
		return err
	}

	// get existing workflow spec from DB
	spec, err := h.orm.GetWorkflowSpec(ctx, hex.EncodeToString(payload.WorkflowOwner), payload.WorkflowName)
	if err != nil {
		return fmt.Errorf("failed to get workflow spec: %w", err)
	}

	// update the status of the workflow spec
	spec.Status = job.WorkflowSpecStatusPaused
	if _, err := h.orm.UpsertWorkflowSpec(ctx, spec); err != nil {
		return fmt.Errorf("failed to update workflow spec: %w", err)
	}

	return nil
}

// workflowActivatedEvent handles the WorkflowActivatedEvent event type.
func (h *eventHandler) workflowActivatedEvent(
	ctx context.Context,
	payload WorkflowRegistryWorkflowActivatedV1,
) error {
	// fetch the workflow spec from the DB
	spec, err := h.orm.GetWorkflowSpec(ctx, hex.EncodeToString(payload.WorkflowOwner), payload.WorkflowName)
	if err != nil {
		return fmt.Errorf("failed to get workflow spec: %w", err)
	}

	// Do nothing if the workflow is already active
	if spec.Status == job.WorkflowSpecStatusActive && h.engineRegistry.IsRunning(hex.EncodeToString(payload.WorkflowID[:])) {
		return nil
	}

	// get the secrets url by the secrets id
	secretsURL, err := h.orm.GetSecretsURLByID(ctx, spec.SecretsID.Int64)
	if err != nil {
		return fmt.Errorf("failed to get secrets URL by ID: %w", err)
	}

	// start a new workflow engine
	registeredEvent := WorkflowRegistryWorkflowRegisteredV1{
		WorkflowID:   payload.WorkflowID,
		Owner:        payload.WorkflowOwner,
		DonID:        payload.DonID,
		Status:       0,
		WorkflowName: payload.WorkflowName,
		BinaryURL:    spec.BinaryURL,
		ConfigURL:    spec.ConfigURL,
		SecretsURL:   secretsURL,
	}

	return h.workflowRegisteredEvent(ctx, registeredEvent)
}

// workflowDeletedEvent handles the WorkflowDeletedEvent event type.
func (h *eventHandler) workflowDeletedEvent(
	ctx context.Context,
	payload WorkflowRegistryWorkflowDeletedV1,
) error {
	if err := h.tryEngineCleanup(hex.EncodeToString(payload.WorkflowID[:])); err != nil {
		return err
	}

	if err := h.orm.DeleteWorkflowSpec(ctx, hex.EncodeToString(payload.WorkflowOwner), payload.WorkflowName); err != nil {
		return fmt.Errorf("failed to delete workflow spec: %w", err)
	}
	return nil
}

// forceUpdateSecretsEvent handles the ForceUpdateSecretsEvent event type.
func (h *eventHandler) forceUpdateSecretsEvent(
	ctx context.Context,
	payload WorkflowRegistryForceUpdateSecretsRequestedV1,
) (string, error) {
	// Get the URL of the secrets file from the event data
	hash := hex.EncodeToString(payload.SecretsURLHash)

	url, err := h.orm.GetSecretsURLByHash(ctx, hash)
	if err != nil {
		return "", fmt.Errorf("failed to get URL by hash %s : %w", hash, err)
	}

	// Fetch the contents of the secrets file from the url via the fetcher
	secrets, err := h.fetcher(ctx, url)
	if err != nil {
		return "", err
	}

	h.lastFetchedAtMap.Set(hash, h.clock.Now())

	// Update the secrets in the ORM
	if _, err := h.orm.Update(ctx, hash, string(secrets)); err != nil {
		return "", fmt.Errorf("failed to update secrets: %w", err)
	}

	return string(secrets), nil
}

// tryEngineCleanup attempts to stop the workflow engine for the given workflow ID.  Does nothing if the
// workflow engine is not running.
func (h *eventHandler) tryEngineCleanup(wfID string) error {
	if h.engineRegistry.IsRunning(wfID) {
		// Remove the engine from the registry
		e, err := h.engineRegistry.Pop(wfID)
		if err != nil {
			return fmt.Errorf("failed to get workflow engine: %w", err)
		}

		// Stop the engine
		if err := e.Close(); err != nil {
			return fmt.Errorf("failed to close workflow engine: %w", err)
		}
	}
	return nil
}

// workflowID returns a hex encoded sha256 hash of the wasm, config and secretsURL.
func workflowID(wasm, config, secretsURL []byte) string {
	sum := sha256.New()
	sum.Write(wasm)
	sum.Write(config)
	sum.Write(secretsURL)
	return hex.EncodeToString(sum.Sum(nil))
}

// logCustMsg emits a custom message to the external sink and logs an error if that fails.
func logCustMsg(ctx context.Context, cma custmsg.MessageEmitter, msg string, log logger.Logger) {
	err := cma.Emit(ctx, msg)
	if err != nil {
		log.Helper(1).Errorf("failed to send custom message with msg: %s, err: %v", msg, err)
	}
}

func newHandlerTypeError(data any) error {
	return fmt.Errorf("invalid data type %T for event", data)
}
