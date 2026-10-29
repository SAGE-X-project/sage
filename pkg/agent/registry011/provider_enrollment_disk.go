package registry011

import (
	"bytes"
	"errors"
	"io"
	"os"
)

type candidateNamespaceState struct {
	root, entry []byte
	tip         string
}

func openCandidateProviderNamespace(config CandidateNamespaceConfig, recovery *ProviderCandidateIdentity) (*CandidateProviderNamespace, *CandidateRetentionReceipt, error) {
	if config.Directory == "" || !lowerHash(config.Namespace) || !lowerHash(config.Instance) || !lowerHash(config.AnchorDigest) ||
		!lowerHash(config.ExpectedTip) || config.MaxBytes <= 0 || config.MaxBytes > 64<<10 {
		return nil, nil, ErrConfiguration
	}
	disk, err := openJournalDisk(config.Directory, false)
	if err != nil {
		return nil, nil, ErrProviderRetention
	}
	success := false
	defer func() {
		if !success {
			_ = disk.close()
		}
	}()
	state, err := loadCandidateNamespace(disk, config)
	if err != nil {
		return nil, nil, err
	}
	if recovery == nil {
		if state.tip != config.ExpectedTip {
			return nil, nil, ErrProviderRetentionConflict
		}
	} else {
		if config.ExpectedTip != config.AnchorDigest || len(state.entry) == 0 {
			return nil, nil, ErrProviderRetentionConflict
		}
		root, entry, err := makeCandidateEnrollment(*recovery, config.AnchorDigest)
		if err != nil || !bytes.Equal(root, state.root) || !bytes.Equal(entry, state.entry) {
			return nil, nil, ErrProviderRetentionConflict
		}
		// Reestablish durability of the identical existing operation. No new
		// root/write, estimated completion, truncation or retry is performed.
		f, err := disk.root.OpenFile("ENROLLMENT", os.O_RDWR, 0)
		if err != nil {
			return nil, nil, ErrProviderRetention
		}
		if errors.Join(disk.syncFile(f), f.Close(), disk.sync()) != nil {
			return nil, nil, ErrProviderRetention
		}
		confirmed, err := loadCandidateNamespace(disk, config)
		if err != nil || confirmed.tip != state.tip || !bytes.Equal(confirmed.root, root) || !bytes.Equal(confirmed.entry, entry) {
			return nil, nil, ErrProviderRetention
		}
	}
	n := &CandidateProviderNamespace{disk: disk, config: config, tip: state.tip, root: string(state.root)}
	success = true
	if recovery != nil {
		return n, &CandidateRetentionReceipt{n, *recovery, config.ExpectedTip, state.tip, string(state.root)}, nil
	}
	return n, nil, nil
}

func loadCandidateNamespace(disk *journalDisk, config CandidateNamespaceConfig) (candidateNamespaceState, error) {
	var state candidateNamespaceState
	f, err := disk.root.Open(".")
	if err != nil {
		return state, ErrProviderRetention
	}
	names, readErr := f.Readdirnames(4)
	closeErr := f.Close()
	if (readErr != nil && readErr != io.EOF) || closeErr != nil || len(names) > 3 {
		return state, ErrProviderRetention
	}
	anchor, locked, entry := false, false, false
	for _, name := range names {
		switch name {
		case "ANCHOR":
			anchor = true
		case "LOCK":
			locked = true
		case "ENROLLMENT":
			entry = true
		default:
			return state, ErrProviderRetention
		}
	}
	if !anchor || !locked {
		return state, ErrProviderRetention
	}
	raw, err := disk.read("ANCHOR", int64(config.MaxBytes))
	if err != nil {
		return state, ErrProviderRetention
	}
	state.root, err = candidateNamespaceAnchor(raw, config.Namespace, config.Instance, config.AnchorDigest)
	if err != nil {
		return state, err
	}
	state.tip = config.AnchorDigest
	if entry {
		if len(state.root) != 0 {
			return state, ErrProviderRetentionConflict
		}
		state.entry, err = disk.read("ENROLLMENT", int64(config.MaxBytes))
		if err != nil {
			return state, ErrProviderRetention
		}
		state.root, err = candidateEnrollmentEntry(state.entry, config.Namespace, config.Instance, config.AnchorDigest)
		if err != nil {
			return state, err
		}
		state.tip = providerCandidateHash("sage-provider-namespace-entry-v1", state.entry)
	}
	return state, nil
}
