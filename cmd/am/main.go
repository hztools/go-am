package main

import (
	"os"

	"hz.tools/am"
	"hz.tools/fftw"
	"hz.tools/pulseaudio"
	"hz.tools/rf"
	"hz.tools/rfcap"
	"hz.tools/sdr"
	"hz.tools/sdr/stream"
)

func main() {
	reader, _, err := rfcap.Reader(os.Stdin)
	if err != nil {
		panic(err)
	}

	reader, err = stream.ConvertReader(reader, sdr.SampleFormatC64)
	if err != nil {
		panic(err)
	}

	demod, err := am.Demodulate(reader, am.DemodulatorConfig{
		CenterFrequency: rf.Hz(0),
		Deviation:       5 * rf.KHz,
		Downsample:      5,
		Planner:         fftw.Plan,
	})
	if err != nil {
		panic(err)
	}

	speaker, err := pulseaudio.NewWriter(pulseaudio.Config{
		Format:     pulseaudio.SampleFormatFloat32NE,
		Rate:       demod.SampleRate(),
		AppName:    "rf",
		StreamName: "am",
		Channels:   1,
	})
	if err != nil {
		panic(err)
	}

	buf := make([]float32, 1024*64)
	for {
		i, err := demod.Read(buf)
		if err != nil {
			panic(err)
		}
		if i == 0 {
			panic("zero read")
		}
		for j := range buf[:i] {
			buf[j] *= 1000
		}
		if err := speaker.Write(buf[:i]); err != nil {
			panic(err)
		}
	}
}
