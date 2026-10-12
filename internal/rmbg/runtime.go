//go:build linux

package rmbg

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// Device names the execution backend.
type Device string

const (
	DeviceAuto Device = "auto"
	DeviceCPU  Device = "cpu"
	DeviceCUDA Device = "cuda"
)

// FindORTLib resolves the ONNX Runtime shared library using the search order
// from the spec:
//
//  1. the explicit -ort-lib value
//  2. the ONNXRUNTIME_LIB environment variable
//  3. ./lib/libonnxruntime.so next to the binary
//  4. libonnxruntime.so on the system loader path
func FindORTLib(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("-ort-lib %q: %w", explicit, err)
		}
		return explicit, nil
	}
	if env := os.Getenv("ONNXRUNTIME_LIB"); env != "" {
		return env, nil
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "lib", "libonnxruntime.so")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "libonnxruntime.so", nil
}

// The ONNX Runtime environment is process-global, so reference-count it to
// support more than one Remover in a process (for example in tests).
var (
	ortMu   sync.Mutex
	ortRefs int
	ortLib  string
)

func initORT(lib string) error {
	ortMu.Lock()
	defer ortMu.Unlock()

	if ortRefs == 0 {
		ort.SetSharedLibraryPath(lib)
		if err := ort.InitializeEnvironment(ort.WithLogLevelError()); err != nil {
			return err
		}
		ortLib = lib
	} else if ortLib != lib {
		return fmt.Errorf("onnxruntime already initialized from %q, cannot switch to %q", ortLib, lib)
	}
	ortRefs++
	return nil
}

func closeORT() {
	ortMu.Lock()
	defer ortMu.Unlock()

	ortRefs--
	if ortRefs <= 0 {
		ortRefs = 0
		_ = ort.DestroyEnvironment()
		ortLib = ""
	}
}

// newSessionOptions builds session options, adding CUDA when useCUDA is set.
func newSessionOptions(opts Options, useCUDA bool) (*ort.SessionOptions, error) {
	so, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("creating session options: %w", err)
	}

	if useCUDA {
		cuda, err := ort.NewCUDAProviderOptions()
		if err == nil {
			err = cuda.Update(map[string]string{"device_id": fmt.Sprint(opts.GPUID)})
		}
		if err == nil {
			err = so.AppendExecutionProviderCUDA(cuda)
		}
		if cuda != nil {
			_ = cuda.Destroy()
		}
		if err != nil {
			_ = so.Destroy()
			return nil, fmt.Errorf("cuda unavailable: %w", err)
		}
	}

	if opts.Threads > 0 {
		if err := so.SetIntraOpNumThreads(opts.Threads); err != nil {
			_ = so.Destroy()
			return nil, fmt.Errorf("setting intra-op threads: %w", err)
		}
	}
	return so, nil
}
