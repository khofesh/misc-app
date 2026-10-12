//go:build cuda && linux

package rmbg

import (
	"image"
	"testing"
)

// TestCPUvsCUDA checks that the CUDA mask matches the CPU mask within the
// tolerance in the spec. Run manually on a machine with an NVIDIA GPU and the
// GPU ONNX Runtime build.
func TestCPUvsCUDA(t *testing.T) {
	lib := ortLibPath(t)

	cpu, err := New(Options{Model: "u2netp", Device: DeviceCPU, ORTLibPath: lib})
	if err != nil {
		t.Fatal(err)
	}
	defer cpu.Close()

	cuda, err := New(Options{Model: "u2netp", Device: DeviceCUDA, ORTLibPath: lib})
	if err != nil {
		t.Fatal(err)
	}
	defer cuda.Close()

	if cuda.Device() != DeviceCUDA {
		t.Fatalf("device = %s, want cuda", cuda.Device())
	}

	img := fixture()
	cpuMask, err := cpu.Mask(img)
	if err != nil {
		t.Fatal(err)
	}
	cudaMask, err := cuda.Mask(img)
	if err != nil {
		t.Fatal(err)
	}

	if cpuMask.Bounds() != cudaMask.Bounds() {
		t.Fatalf("bounds differ: cpu %v, cuda %v", cpuMask.Bounds(), cudaMask.Bounds())
	}

	diff := meanAbsGrayDiff(cpuMask, cudaMask)
	if diff > 2 {
		t.Errorf("mean absolute difference = %.2f, want <= 2", diff)
	}
}

func meanAbsGrayDiff(a, b *image.Gray) float64 {
	var sum, n float64
	for i := range a.Pix {
		d := int(a.Pix[i]) - int(b.Pix[i])
		if d < 0 {
			d = -d
		}
		sum += float64(d)
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / n
}
