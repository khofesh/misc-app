//go:build linux

package rmbg

import (
	"fmt"
	"image"
	"sync"

	"github.com/rs/zerolog/log"
	ort "github.com/yalue/onnxruntime_go"
)

// Options configures a Remover.
type Options struct {
	Model      string // registry name or path to a .onnx file
	Device     Device
	GPUID      int
	ORTLibPath string
	Threads    int
}

// Remover runs background segmentation with a single loaded model session.
type Remover struct {
	session *ort.AdvancedSession
	input   *ort.Tensor[float32]
	outputs []*ort.Tensor[float32]
	spec    ModelSpec
	device  Device
	size    int
	outW    int
	outH    int

	mu sync.Mutex // serializes session.Run
}

// New loads the model and creates an inference session on the requested device.
func New(opts Options) (*Remover, error) {
	if opts.Device == "" {
		opts.Device = DeviceAuto
	}
	switch opts.Device {
	case DeviceAuto, DeviceCPU, DeviceCUDA:
	default:
		return nil, fmt.Errorf("invalid device %q", opts.Device)
	}

	path, spec, err := ModelPath(opts.Model)
	if err != nil {
		return nil, err
	}

	lib, err := FindORTLib(opts.ORTLibPath)
	if err != nil {
		return nil, err
	}
	if err := initORT(lib); err != nil {
		closeORT()
		return nil, fmt.Errorf("loading onnxruntime (%s): %w (see the rmbg section of README.md for install steps)", lib, err)
	}

	r := &Remover{spec: spec}
	if err := r.initSession(path, opts); err != nil {
		closeORT()
		return nil, err
	}
	return r, nil
}

func (r *Remover) initSession(path string, opts Options) error {
	inputs, outputs, err := ort.GetInputOutputInfo(path)
	if err != nil {
		return fmt.Errorf("reading model %q: %w", path, err)
	}
	if len(inputs) != 1 {
		return fmt.Errorf("model %q has %d inputs, want 1", path, len(inputs))
	}
	if len(outputs) == 0 {
		return fmt.Errorf("model %q has no outputs", path)
	}

	inputName := inputs[0].Name
	if inputName == "" {
		inputName = r.spec.InputName
	}

	r.size = r.spec.Size
	if d := inputs[0].Dimensions; len(d) == 4 && d[2] > 0 {
		r.size = int(d[2])
	}
	if r.size <= 0 {
		return fmt.Errorf("cannot determine input size for model %q", path)
	}

	r.outW, r.outH = r.size, r.size
	if d := outputs[0].Dimensions; len(d) == 4 && d[2] > 0 && d[3] > 0 {
		r.outH, r.outW = int(d[2]), int(d[3])
	}

	// u2net-style models expose several outputs; the first is the fused mask.
	outNames := make([]string, len(outputs))
	outShapes := make([]ort.Shape, len(outputs))
	for i, o := range outputs {
		name := o.Name
		if i == 0 && name == "" {
			name = r.spec.OutputName
		}
		outNames[i] = name
		if d := o.Dimensions; len(d) == 4 && d[1] > 0 && d[2] > 0 && d[3] > 0 {
			outShapes[i] = ort.NewShape(d[0], d[1], d[2], d[3])
		} else {
			outShapes[i] = ort.NewShape(1, 1, int64(r.outH), int64(r.outW))
		}
	}

	dev, sess, err := r.createSession(path, inputName, outNames, outShapes, opts)
	if err != nil {
		return err
	}
	r.device = dev
	r.session = sess
	return nil
}

// createSession creates the tensors and the session, falling back to CPU when
// the device is auto and CUDA is unavailable.
func (r *Remover) createSession(path, inputName string, outNames []string, outShapes []ort.Shape, opts Options) (Device, *ort.AdvancedSession, error) {
	wantCUDA := opts.Device == DeviceCUDA || opts.Device == DeviceAuto

	dev, sess, err := r.trySession(path, inputName, outNames, outShapes, opts, wantCUDA)
	if err == nil {
		return dev, sess, nil
	}
	if opts.Device == DeviceAuto && wantCUDA {
		log.Warn().Err(err).Msg("cuda unavailable, falling back to cpu")
		return r.trySession(path, inputName, outNames, outShapes, opts, false)
	}
	return "", nil, err
}

func (r *Remover) trySession(path, inputName string, outNames []string, outShapes []ort.Shape, opts Options, useCUDA bool) (Device, *ort.AdvancedSession, error) {
	so, err := newSessionOptions(opts, useCUDA)
	if err != nil {
		return "", nil, err
	}
	defer so.Destroy()

	input, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 3, int64(r.size), int64(r.size)))
	if err != nil {
		return "", nil, fmt.Errorf("allocating input tensor: %w", err)
	}

	outputs := make([]*ort.Tensor[float32], len(outShapes))
	values := make([]ort.Value, len(outShapes))
	for i, shape := range outShapes {
		t, err := ort.NewEmptyTensor[float32](shape)
		if err != nil {
			_ = input.Destroy()
			destroyTensors(outputs)
			return "", nil, fmt.Errorf("allocating output tensor %d: %w", i, err)
		}
		outputs[i] = t
		values[i] = t
	}

	sess, err := ort.NewAdvancedSession(path, []string{inputName}, outNames,
		[]ort.Value{input}, values, so)
	if err != nil {
		_ = input.Destroy()
		destroyTensors(outputs)
		return "", nil, fmt.Errorf("creating session: %w", err)
	}

	r.input = input
	r.outputs = outputs
	if useCUDA {
		return DeviceCUDA, sess, nil
	}
	return DeviceCPU, sess, nil
}

func destroyTensors(ts []*ort.Tensor[float32]) {
	for _, t := range ts {
		if t != nil {
			_ = t.Destroy()
		}
	}
}

// Device returns the device actually in use.
func (r *Remover) Device() Device { return r.device }

// Size returns the model's square input size.
func (r *Remover) Size() int { return r.size }

// Preprocess resizes and normalizes img into the model's input tensor layout.
func (r *Remover) Preprocess(img image.Image) ([]float32, error) {
	return Preprocess(img, r.size, r.spec.Mean, r.spec.Std)
}

// Infer runs the model on an input tensor and returns a copy of the raw output.
func (r *Remover) Infer(input []float32) ([]float32, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	copy(r.input.GetData(), input)
	if err := r.session.Run(); err != nil {
		return nil, fmt.Errorf("inference: %w", err)
	}
	out := r.outputs[0]
	raw := make([]float32, len(out.GetData()))
	copy(raw, out.GetData())
	return raw, nil
}

// Postprocess turns the raw model output into an alpha mask of size w×h.
func (r *Remover) Postprocess(raw []float32, w, h int) *image.Gray {
	norm := NormalizeMask(raw)
	resized := ResizeMask(norm, r.outW, r.outH, w, h)
	return MaskToGray(resized, w, h)
}

// Mask runs the model and returns the alpha mask resized to the input image.
func (r *Remover) Mask(img image.Image) (*image.Gray, error) {
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return nil, fmt.Errorf("empty image")
	}

	tensor, err := r.Preprocess(img)
	if err != nil {
		return nil, err
	}
	raw, err := r.Infer(tensor)
	if err != nil {
		return nil, err
	}
	return r.Postprocess(raw, b.Dx(), b.Dy()), nil
}

// Remove runs the model and returns the image with its background made
// transparent.
func (r *Remover) Remove(img image.Image) (*image.NRGBA, error) {
	mask, err := r.Mask(img)
	if err != nil {
		return nil, err
	}
	return ApplyAlpha(img, mask), nil
}

// Close releases the session and the ONNX Runtime environment.
func (r *Remover) Close() error {
	var err error
	if r.session != nil {
		err = r.session.Destroy()
	}
	if r.input != nil {
		_ = r.input.Destroy()
	}
	destroyTensors(r.outputs)
	closeORT()
	return err
}
